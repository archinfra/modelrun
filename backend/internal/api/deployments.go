package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/store"
)

func (a *API) handleDeployments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		deployments := data.Deployments
		if status := r.URL.Query().Get("status"); status != "" {
			deployments = make([]domain.DeploymentConfig, 0, len(data.Deployments))
			for _, deployment := range data.Deployments {
				if deployment.Status == status {
					deployments = append(deployments, deployment)
				}
			}
		}
		writeJSON(w, http.StatusOK, deployments)
	case http.MethodPost:
		deployment, err := a.readDeploymentCreate(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if err := a.store.Update(func(data *domain.Data) error {
			data.Deployments = append(data.Deployments, deployment)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusCreated, deployment)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleDeployment(w http.ResponseWriter, r *http.Request) {
	id, rest := pathParts(r.URL.Path, "/api/deployments/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	if len(rest) == 0 {
		a.handleDeploymentItem(w, r, id)
		return
	}
	if len(rest) != 1 {
		http.NotFound(w, r)
		return
	}

	switch rest[0] {
	case "start":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		taskIDs, err := a.executor.Start(id)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		taskID := ""
		if len(taskIDs) > 0 {
			taskID = taskIDs[0]
		}
		writeJSON(w, http.StatusOK, map[string]any{"taskId": taskID, "taskIds": taskIDs})
	case "stop":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		if err := a.executor.Stop(id); err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	case "logs":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		a.handleDeploymentLogs(w, r, id)
	case "metrics":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		a.handleDeploymentMetrics(w, id)
	default:
		http.NotFound(w, r)
	}
}

func (a *API) handleDeploymentItem(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		idx := findDeployment(data.Deployments, id)
		if idx < 0 {
			writeError(w, http.StatusNotFound, store.ErrNotFound)
			return
		}
		writeJSON(w, http.StatusOK, data.Deployments[idx])
	case http.MethodPut, http.MethodPatch:
		var patch map[string]json.RawMessage
		if err := readJSON(r, &patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		var deployment domain.DeploymentConfig
		err := a.store.Update(func(data *domain.Data) error {
			idx := findDeployment(data.Deployments, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			updated, err := mergeJSON(data.Deployments[idx], patch)
			if err != nil {
				return err
			}
			updated.ID = data.Deployments[idx].ID
			if updated.CreatedAt == "" {
				updated.CreatedAt = data.Deployments[idx].CreatedAt
			}
			updated.UpdatedAt = domain.Now()
			defaultDeployment(&updated)
			data.Deployments[idx] = updated
			deployment = updated
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, deployment)
	case http.MethodDelete:
		err := a.store.Update(func(data *domain.Data) error {
			idx := findDeployment(data.Deployments, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			data.Deployments = append(data.Deployments[:idx], data.Deployments[idx+1:]...)
			data.Tasks = filterDeploymentTasks(data.Tasks, id)
			data.Logs = filterDeploymentLogs(data.Logs, id)
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleDeploymentLogs(w http.ResponseWriter, r *http.Request, id string) {
	data := a.store.Snapshot()
	if findDeployment(data.Deployments, id) < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	serverID := r.URL.Query().Get("serverId")
	stepID := r.URL.Query().Get("stepId")
	logs := make([]domain.DeploymentLog, 0)
	for _, entry := range data.Logs {
		if entry.DeploymentID != id {
			continue
		}
		if serverID != "" && entry.ServerID != serverID {
			continue
		}
		if stepID != "" && entry.StepID != stepID {
			continue
		}
		logs = append(logs, entry)
	}

	writeJSON(w, http.StatusOK, logs)
}

func (a *API) handleDeploymentMetrics(w http.ResponseWriter, id string) {
	data := a.store.Snapshot()
	idx := findDeployment(data.Deployments, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	if data.Deployments[idx].Metrics == nil {
		writeJSON(w, http.StatusOK, domain.DeploymentMetrics{})
		return
	}

	writeJSON(w, http.StatusOK, data.Deployments[idx].Metrics)
}

type createDeploymentRequest struct {
	ID              string                            `json:"id"`
	Name            string                            `json:"name"`
	Status          string                            `json:"status"`
	Framework       string                            `json:"framework"`
	Model           *domain.ModelConfig               `json:"model"`
	ModelID         string                            `json:"modelId"`
	ServerIDs       []string                          `json:"serverIds"`
	Servers         []string                          `json:"servers"`
	Docker          domain.DockerConfig               `json:"docker"`
	VLLM            domain.VLLMParams                 `json:"vllm"`
	Ray             domain.DeploymentRayConfig        `json:"ray"`
	Runtime         domain.DeploymentRuntimeConfig    `json:"runtime"`
	ServerOverrides []domain.DeploymentServerOverride `json:"serverOverrides"`
	APIPort         int                               `json:"apiPort"`
	CreatedAt       string                            `json:"createdAt"`
	UpdatedAt       string                            `json:"updatedAt"`
}

func (a *API) readDeploymentCreate(r *http.Request) (domain.DeploymentConfig, error) {
	var req createDeploymentRequest
	if err := readJSON(r, &req); err != nil {
		return domain.DeploymentConfig{}, err
	}

	model := domain.ModelConfig{}
	if req.Model != nil {
		model = *req.Model
		defaultModel(&model)
	} else if req.ModelID != "" {
		data := a.store.Snapshot()
		idx := findModel(data.Models, req.ModelID)
		if idx >= 0 {
			model = data.Models[idx]
		} else {
			model = domain.ModelConfig{
				ID:      req.ModelID,
				Name:    baseName(req.ModelID),
				Source:  "local",
				ModelID: req.ModelID,
			}
			defaultModel(&model)
		}
	} else {
		return domain.DeploymentConfig{}, errors.New("model or modelId is required")
	}

	servers := req.Servers
	if len(servers) == 0 {
		servers = req.ServerIDs
	}
	if len(servers) == 0 {
		return domain.DeploymentConfig{}, errors.New("servers or serverIds is required")
	}

	deployment := domain.DeploymentConfig{
		ID:              req.ID,
		Name:            req.Name,
		Status:          req.Status,
		Framework:       req.Framework,
		Model:           model,
		Docker:          req.Docker,
		VLLM:            req.VLLM,
		Ray:             req.Ray,
		Runtime:         req.Runtime,
		ServerOverrides: req.ServerOverrides,
		Servers:         servers,
		APIPort:         req.APIPort,
		CreatedAt:       req.CreatedAt,
		UpdatedAt:       req.UpdatedAt,
	}
	defaultDeployment(&deployment)

	return deployment, nil
}

func findDeployment(deployments []domain.DeploymentConfig, id string) int {
	for i, deployment := range deployments {
		if deployment.ID == id {
			return i
		}
	}
	return -1
}

func filterDeploymentTasks(tasks []domain.DeploymentTask, deploymentID string) []domain.DeploymentTask {
	filtered := tasks[:0]
	for _, task := range tasks {
		if task.DeploymentID != deploymentID {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func filterDeploymentLogs(logs []domain.DeploymentLog, deploymentID string) []domain.DeploymentLog {
	filtered := logs[:0]
	for _, entry := range logs {
		if entry.DeploymentID != deploymentID {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

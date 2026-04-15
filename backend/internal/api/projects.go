package api

import (
	"encoding/json"
	"net/http"

	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/store"
)

func (a *API) handleProjects(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, a.store.Snapshot().Projects)
	case http.MethodPost:
		var project domain.Project
		if err := readJSON(r, &project); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		defaultProject(&project)

		if err := a.store.Update(func(data *domain.Data) error {
			data.Projects = append(data.Projects, project)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusCreated, project)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleProject(w http.ResponseWriter, r *http.Request) {
	id, rest := pathParts(r.URL.Path, "/api/projects/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	if len(rest) == 1 && rest[0] == "summary" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		a.handleProjectSummary(w, id)
		return
	}
	if len(rest) != 0 {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		idx := findProject(data.Projects, id)
		if idx < 0 {
			writeError(w, http.StatusNotFound, store.ErrNotFound)
			return
		}
		writeJSON(w, http.StatusOK, data.Projects[idx])
	case http.MethodPut, http.MethodPatch:
		var patch map[string]json.RawMessage
		if err := readJSON(r, &patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		var project domain.Project
		err := a.store.Update(func(data *domain.Data) error {
			idx := findProject(data.Projects, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			updated, err := mergeJSON(data.Projects[idx], patch)
			if err != nil {
				return err
			}
			updated.ID = data.Projects[idx].ID
			if updated.CreatedAt == "" {
				updated.CreatedAt = data.Projects[idx].CreatedAt
			}
			updated.UpdatedAt = domain.Now()
			data.Projects[idx] = updated
			project = updated
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, project)
	case http.MethodDelete:
		err := a.store.Update(func(data *domain.Data) error {
			idx := findProject(data.Projects, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			serverIDs := projectServerIDs(data.Servers, id)
			deploymentIDs := projectDeploymentIDs(data.Deployments, serverIDs)
			data.Projects = append(data.Projects[:idx], data.Projects[idx+1:]...)
			data.Servers = filterServersByProject(data.Servers, id)
			data.Deployments = filterDeployments(data.Deployments, deploymentIDs)
			data.Tasks = filterTasks(data.Tasks, deploymentIDs)
			data.Logs = filterLogs(data.Logs, deploymentIDs)
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

func (a *API) handleProjectSummary(w http.ResponseWriter, id string) {
	data := a.store.Snapshot()
	idx := findProject(data.Projects, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	serverIDs := projectServerIDs(data.Servers, id)
	deploymentIDs := projectDeploymentIDs(data.Deployments, serverIDs)
	gpuCount := 0
	onlineCount := 0
	for _, server := range data.Servers {
		if server.ProjectID != id {
			continue
		}
		gpuCount += len(server.GPUInfo)
		if server.Status == "online" {
			onlineCount++
		}
	}

	runningCount := 0
	for _, deployment := range data.Deployments {
		if deploymentIDs[deployment.ID] && deployment.Status == "running" {
			runningCount++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"project":       data.Projects[idx],
		"servers":       len(serverIDs),
		"onlineServers": onlineCount,
		"gpus":          gpuCount,
		"deployments":   len(deploymentIDs),
		"running":       runningCount,
	})
}

func findProject(projects []domain.Project, id string) int {
	for i, project := range projects {
		if project.ID == id {
			return i
		}
	}
	return -1
}

func filterServersByProject(servers []domain.ServerConfig, projectID string) []domain.ServerConfig {
	filtered := servers[:0]
	for _, server := range servers {
		if server.ProjectID != projectID {
			filtered = append(filtered, server)
		}
	}
	return filtered
}

func projectServerIDs(servers []domain.ServerConfig, projectID string) map[string]bool {
	ids := map[string]bool{}
	for _, server := range servers {
		if server.ProjectID == projectID {
			ids[server.ID] = true
		}
	}
	return ids
}

func projectDeploymentIDs(deployments []domain.DeploymentConfig, serverIDs map[string]bool) map[string]bool {
	ids := map[string]bool{}
	for _, deployment := range deployments {
		for _, serverID := range deployment.Servers {
			if serverIDs[serverID] {
				ids[deployment.ID] = true
				break
			}
		}
	}
	return ids
}

func filterDeployments(deployments []domain.DeploymentConfig, removed map[string]bool) []domain.DeploymentConfig {
	filtered := deployments[:0]
	for _, deployment := range deployments {
		if !removed[deployment.ID] {
			filtered = append(filtered, deployment)
		}
	}
	return filtered
}

func filterTasks(tasks []domain.DeploymentTask, removed map[string]bool) []domain.DeploymentTask {
	filtered := tasks[:0]
	for _, task := range tasks {
		if !removed[task.DeploymentID] {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func filterLogs(logs []domain.DeploymentLog, removed map[string]bool) []domain.DeploymentLog {
	filtered := logs[:0]
	for _, entry := range logs {
		if !removed[entry.DeploymentID] {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

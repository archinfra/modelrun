package deploy

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/realtime"
	"modelrun/backend/internal/store"
)

type Executor struct {
	store *store.Store
	hub   *realtime.Hub
}

func NewExecutor(st *store.Store, hub *realtime.Hub) *Executor {
	return &Executor{store: st, hub: hub}
}

func (e *Executor) Start(deploymentID string) ([]string, error) {
	var taskIDs []string
	var deployment domain.DeploymentConfig

	err := e.store.Update(func(data *domain.Data) error {
		idx := findDeployment(data.Deployments, deploymentID)
		if idx < 0 {
			return store.ErrNotFound
		}

		deployment = data.Deployments[idx]
		if len(deployment.Servers) == 0 {
			return errors.New("deployment has no servers")
		}

		data.Tasks = filterTasks(data.Tasks, deploymentID)
		for _, serverID := range deployment.Servers {
			task := domain.DeploymentTask{
				ID:           domain.NewID("task"),
				DeploymentID: deploymentID,
				ServerID:     serverID,
				Steps:        defaultSteps(),
			}
			data.Tasks = append(data.Tasks, task)
			taskIDs = append(taskIDs, task.ID)
		}

		data.Deployments[idx].Status = "deploying"
		data.Deployments[idx].UpdatedAt = domain.Now()
		data.Deployments[idx].Endpoints = nil
		data.Deployments[idx].Metrics = nil
		deployment = data.Deployments[idx]

		return nil
	})
	if err != nil {
		return nil, err
	}

	e.broadcastStatus(deployment)
	go e.run(deploymentID)

	return taskIDs, nil
}

func (e *Executor) Stop(deploymentID string) error {
	var deployment domain.DeploymentConfig

	err := e.store.Update(func(data *domain.Data) error {
		idx := findDeployment(data.Deployments, deploymentID)
		if idx < 0 {
			return store.ErrNotFound
		}

		data.Deployments[idx].Status = "stopped"
		data.Deployments[idx].UpdatedAt = domain.Now()
		for i := range data.Deployments[idx].Endpoints {
			data.Deployments[idx].Endpoints[i].Status = "unknown"
		}
		deployment = data.Deployments[idx]

		return nil
	})
	if err != nil {
		return err
	}

	e.addLog(deploymentID, "", "", "warn", "deployment stopped")
	e.broadcastStatus(deployment)

	return nil
}

func (e *Executor) run(deploymentID string) {
	snapshot := e.store.Snapshot()
	deployment, ok := getDeployment(snapshot.Deployments, deploymentID)
	if !ok {
		return
	}

	for _, serverID := range deployment.Servers {
		for stepIndex := range defaultSteps() {
			if e.isStopped(deploymentID) {
				return
			}

			e.startStep(deploymentID, serverID, stepIndex)
			stepID := defaultSteps()[stepIndex].ID
			e.addLog(deploymentID, serverID, stepID, "info", fmt.Sprintf("starting %s", stepID))

			for progress := 20; progress <= 100; progress += 20 {
				time.Sleep(300 * time.Millisecond)
				if e.isStopped(deploymentID) {
					return
				}
				e.updateStep(deploymentID, serverID, stepIndex, "running", progress)
			}

			e.completeStep(deploymentID, serverID, stepIndex)
			e.addLog(deploymentID, serverID, stepID, "info", fmt.Sprintf("completed %s", stepID))
		}
	}

	e.completeDeployment(deploymentID)
}

func (e *Executor) startStep(deploymentID, serverID string, stepIndex int) {
	now := domain.Now()
	_ = e.store.Update(func(data *domain.Data) error {
		taskIdx := findTask(data.Tasks, deploymentID, serverID)
		if taskIdx < 0 || stepIndex >= len(data.Tasks[taskIdx].Steps) {
			return nil
		}

		task := &data.Tasks[taskIdx]
		task.CurrentStep = stepIndex
		task.Steps[stepIndex].Status = "running"
		task.Steps[stepIndex].StartTime = now
		task.Steps[stepIndex].Progress = 0
		task.OverallProgress = calculateOverall(task.Steps)

		e.broadcastProgress(*task, task.Steps[stepIndex])
		return nil
	})
}

func (e *Executor) updateStep(deploymentID, serverID string, stepIndex int, status string, progress int) {
	_ = e.store.Update(func(data *domain.Data) error {
		taskIdx := findTask(data.Tasks, deploymentID, serverID)
		if taskIdx < 0 || stepIndex >= len(data.Tasks[taskIdx].Steps) {
			return nil
		}

		task := &data.Tasks[taskIdx]
		task.Steps[stepIndex].Status = status
		task.Steps[stepIndex].Progress = progress
		task.OverallProgress = calculateOverall(task.Steps)

		e.broadcastProgress(*task, task.Steps[stepIndex])
		return nil
	})
}

func (e *Executor) completeStep(deploymentID, serverID string, stepIndex int) {
	now := domain.Now()
	_ = e.store.Update(func(data *domain.Data) error {
		taskIdx := findTask(data.Tasks, deploymentID, serverID)
		if taskIdx < 0 || stepIndex >= len(data.Tasks[taskIdx].Steps) {
			return nil
		}

		task := &data.Tasks[taskIdx]
		task.Steps[stepIndex].Status = "completed"
		task.Steps[stepIndex].Progress = 100
		task.Steps[stepIndex].EndTime = now
		task.OverallProgress = calculateOverall(task.Steps)

		e.broadcastProgress(*task, task.Steps[stepIndex])
		return nil
	})
}

func (e *Executor) addLog(deploymentID, serverID, stepID, level, message string) {
	entry := domain.DeploymentLog{
		Timestamp:    domain.Now(),
		Level:        level,
		Message:      message,
		DeploymentID: deploymentID,
		ServerID:     serverID,
		StepID:       stepID,
	}

	_ = e.store.Update(func(data *domain.Data) error {
		data.Logs = append(data.Logs, entry)
		if len(data.Logs) > 2000 {
			data.Logs = data.Logs[len(data.Logs)-2000:]
		}

		taskIdx := findTask(data.Tasks, deploymentID, serverID)
		if taskIdx >= 0 {
			for i := range data.Tasks[taskIdx].Steps {
				if data.Tasks[taskIdx].Steps[i].ID == stepID {
					data.Tasks[taskIdx].Steps[i].Logs = append(data.Tasks[taskIdx].Steps[i].Logs, message)
					break
				}
			}
		}

		return nil
	})

	e.hub.Broadcast(realtime.Message{
		Type:         "log",
		DeploymentID: deploymentID,
		Data:         entry,
	})
}

func (e *Executor) completeDeployment(deploymentID string) {
	var deployment domain.DeploymentConfig

	_ = e.store.Update(func(data *domain.Data) error {
		idx := findDeployment(data.Deployments, deploymentID)
		if idx < 0 {
			return nil
		}

		dep := &data.Deployments[idx]
		dep.Status = "running"
		dep.UpdatedAt = domain.Now()
		dep.Endpoints = makeEndpoints(*dep, data.Servers)
		dep.Metrics = &domain.DeploymentMetrics{
			TotalRequests:     0,
			AvgLatency:        18,
			TokensPerSecond:   240,
			GPUUtilization:    35,
			MemoryUtilization: 42,
		}
		deployment = *dep

		return nil
	})

	e.addLog(deploymentID, "", "", "info", "deployment is running")
	e.broadcastStatus(deployment)
	e.hub.Broadcast(realtime.Message{
		Type:         "metric",
		DeploymentID: deploymentID,
		Data:         deployment.Metrics,
	})
}

func (e *Executor) isStopped(deploymentID string) bool {
	snapshot := e.store.Snapshot()
	deployment, ok := getDeployment(snapshot.Deployments, deploymentID)
	return ok && deployment.Status == "stopped"
}

func (e *Executor) broadcastProgress(task domain.DeploymentTask, step domain.DeploymentStep) {
	e.hub.Broadcast(realtime.Message{
		Type:         "progress",
		DeploymentID: task.DeploymentID,
		Data: map[string]any{
			"serverId":        task.ServerID,
			"stepId":          step.ID,
			"progress":        step.Progress,
			"status":          step.Status,
			"overallProgress": task.OverallProgress,
		},
	})
}

func (e *Executor) broadcastStatus(deployment domain.DeploymentConfig) {
	e.hub.Broadcast(realtime.Message{
		Type:         "status",
		DeploymentID: deployment.ID,
		Data: map[string]any{
			"status":    deployment.Status,
			"endpoints": deployment.Endpoints,
		},
	})
}

func defaultSteps() []domain.DeploymentStep {
	return []domain.DeploymentStep{
		{ID: "check_environment", Name: "Check environment", Description: "Check Docker, GPU runtime, disk and driver", Status: "pending", Logs: []string{}},
		{ID: "pull_image", Name: "Pull image", Description: "Prepare the configured inference image", Status: "pending", Logs: []string{}},
		{ID: "prepare_model", Name: "Prepare model", Description: "Validate or download model files", Status: "pending", Logs: []string{}},
		{ID: "start_container", Name: "Start container", Description: "Create and start inference container", Status: "pending", Logs: []string{}},
		{ID: "health_check", Name: "Health check", Description: "Wait for API readiness", Status: "pending", Logs: []string{}},
	}
}

func calculateOverall(steps []domain.DeploymentStep) int {
	if len(steps) == 0 {
		return 0
	}

	total := 0
	for _, step := range steps {
		total += step.Progress
	}
	return total / len(steps)
}

func makeEndpoints(deployment domain.DeploymentConfig, servers []domain.ServerConfig) []domain.DeploymentEndpoint {
	endpoints := make([]domain.DeploymentEndpoint, 0, len(deployment.Servers))
	for _, serverID := range deployment.Servers {
		host := serverID
		if server, ok := getServer(servers, serverID); ok && server.Host != "" {
			host = server.Host
		}
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimPrefix(host, "https://")
		endpoints = append(endpoints, domain.DeploymentEndpoint{
			ServerID: serverID,
			URL:      fmt.Sprintf("http://%s:%d/v1", host, deployment.APIPort),
			Status:   "healthy",
			Latency:  18,
		})
	}
	return endpoints
}

func findDeployment(deployments []domain.DeploymentConfig, id string) int {
	for i, deployment := range deployments {
		if deployment.ID == id {
			return i
		}
	}
	return -1
}

func getDeployment(deployments []domain.DeploymentConfig, id string) (domain.DeploymentConfig, bool) {
	if idx := findDeployment(deployments, id); idx >= 0 {
		return deployments[idx], true
	}
	return domain.DeploymentConfig{}, false
}

func findTask(tasks []domain.DeploymentTask, deploymentID, serverID string) int {
	for i, task := range tasks {
		if task.DeploymentID == deploymentID && task.ServerID == serverID {
			return i
		}
	}
	return -1
}

func filterTasks(tasks []domain.DeploymentTask, deploymentID string) []domain.DeploymentTask {
	filtered := tasks[:0]
	for _, task := range tasks {
		if task.DeploymentID != deploymentID {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

func getServer(servers []domain.ServerConfig, id string) (domain.ServerConfig, bool) {
	for _, server := range servers {
		if server.ID == id {
			return server, true
		}
	}
	return domain.ServerConfig{}, false
}

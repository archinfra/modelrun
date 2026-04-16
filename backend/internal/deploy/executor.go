package deploy

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"modelrun/backend/internal/collect"
	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/realtime"
	"modelrun/backend/internal/store"
)

type Executor struct {
	store     *store.Store
	hub       *realtime.Hub
	collector *collect.Collector
}

func NewExecutor(st *store.Store, hub *realtime.Hub) *Executor {
	return &Executor{
		store:     st,
		hub:       hub,
		collector: collect.New(),
	}
}

func (e *Executor) Start(deploymentID string) ([]string, error) {
	snapshot := e.store.Snapshot()
	deployment, ok := getDeployment(snapshot.Deployments, deploymentID)
	if !ok {
		return nil, store.ErrNotFound
	}
	if len(deployment.Servers) == 0 {
		return nil, errors.New("deployment has no servers")
	}

	serverList, err := deploymentServers(snapshot.Servers, deployment.Servers)
	if err != nil {
		return nil, err
	}

	plans := map[string][]plannedStep{}
	for _, server := range serverList {
		plan, err := buildPlan(deployment, server, serverList)
		if err != nil {
			return nil, err
		}
		plans[server.ID] = plan
	}

	taskIDs := make([]string, 0, len(serverList))
	err = e.store.Update(func(data *domain.Data) error {
		idx := findDeployment(data.Deployments, deploymentID)
		if idx < 0 {
			return store.ErrNotFound
		}

		data.Tasks = filterTasks(data.Tasks, deploymentID)
		for _, server := range serverList {
			task := domain.DeploymentTask{
				ID:           domain.NewID("task"),
				DeploymentID: deploymentID,
				ServerID:     server.ID,
				Steps:        stepsFromPlan(plans[server.ID]),
			}
			data.Tasks = append(data.Tasks, task)
			taskIDs = append(taskIDs, task.ID)
		}

		data.Deployments[idx].Status = "deploying"
		data.Deployments[idx].UpdatedAt = domain.Now()
		data.Deployments[idx].Endpoints = nil
		data.Deployments[idx].Metrics = nil
		return nil
	})
	if err != nil {
		return nil, err
	}

	updated, _ := getDeployment(e.store.Snapshot().Deployments, deploymentID)
	e.broadcastStatus(updated)
	go e.run(deploymentID)

	return taskIDs, nil
}

func (e *Executor) Stop(deploymentID string) error {
	snapshot := e.store.Snapshot()
	deployment, ok := getDeployment(snapshot.Deployments, deploymentID)
	if !ok {
		return store.ErrNotFound
	}

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
		return nil
	})
	if err != nil {
		return err
	}

	e.addLog(deploymentID, "", "", "warn", "deployment stopped")
	stopped, _ := getDeployment(e.store.Snapshot().Deployments, deploymentID)
	e.broadcastStatus(stopped)

	go e.stopRuntime(deployment)
	return nil
}

func (e *Executor) run(deploymentID string) {
	snapshot := e.store.Snapshot()
	deployment, ok := getDeployment(snapshot.Deployments, deploymentID)
	if !ok {
		return
	}

	serverList, err := deploymentServers(snapshot.Servers, deployment.Servers)
	if err != nil {
		e.failDeployment(deploymentID, err)
		return
	}

	var failed atomic.Bool
	var wg sync.WaitGroup
	for _, server := range serverList {
		server := server
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := e.runServer(deployment, server, serverList, &failed); err != nil {
				failed.Store(true)
				e.failDeployment(deploymentID, fmt.Errorf("%s: %w", firstNonEmpty(server.Name, server.ID), err))
			}
		}()
	}

	wg.Wait()
	if failed.Load() || e.isStopped(deploymentID) {
		return
	}
	e.completeDeployment(deploymentID)
}

func (e *Executor) runServer(deployment domain.DeploymentConfig, server domain.ServerConfig, servers []domain.ServerConfig, failed *atomic.Bool) error {
	snapshot := e.store.Snapshot()
	jump, err := resolveJumpHost(snapshot, server)
	if err != nil {
		return err
	}

	plan, err := buildPlan(deployment, server, servers)
	if err != nil {
		return err
	}

	for stepIndex, step := range plan {
		if failed.Load() || e.isStopped(deployment.ID) {
			return nil
		}

		e.startStep(deployment.ID, server.ID, stepIndex)
		e.addLog(deployment.ID, server.ID, step.step.ID, "info", "starting "+step.step.Name)

		result, err := e.collector.RunCommand(server, jump, step.command)
		if err != nil {
			e.addLog(deployment.ID, server.ID, step.step.ID, "error", err.Error())
			e.recordCommandOutput(deployment.ID, server.ID, step.step.ID, result)
			e.failStep(deployment.ID, server.ID, stepIndex, err)
			return err
		}

		e.recordCommandOutput(deployment.ID, server.ID, step.step.ID, result)
		e.completeStep(deployment.ID, server.ID, stepIndex)
		e.addLog(deployment.ID, server.ID, step.step.ID, "info", "completed "+step.step.Name)
	}

	return nil
}

func (e *Executor) stopRuntime(deployment domain.DeploymentConfig) {
	snapshot := e.store.Snapshot()
	serverList, err := deploymentServers(snapshot.Servers, deployment.Servers)
	if err != nil {
		return
	}
	template, ok := LookupTemplate(deployment.Framework)
	if !ok {
		return
	}
	runtime := mergedRuntimeConfig(template, deployment.Runtime)
	containerName := deploymentContainerName(deployment, runtime)
	command := withDockerPrivileges("(run_docker rm -f " + shellQuote(containerName) + " >/dev/null 2>&1 || true)")

	for _, server := range serverList {
		jump, err := resolveJumpHost(snapshot, server)
		if err != nil {
			continue
		}
		_, _ = e.collector.RunCommand(server, jump, command)
	}
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
		task.Steps[stepIndex].Progress = 20
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

func (e *Executor) failStep(deploymentID, serverID string, stepIndex int, stepErr error) {
	now := domain.Now()
	_ = e.store.Update(func(data *domain.Data) error {
		taskIdx := findTask(data.Tasks, deploymentID, serverID)
		if taskIdx < 0 || stepIndex >= len(data.Tasks[taskIdx].Steps) {
			return nil
		}

		task := &data.Tasks[taskIdx]
		task.Steps[stepIndex].Status = "failed"
		task.Steps[stepIndex].Progress = 100
		task.Steps[stepIndex].EndTime = now
		task.OverallProgress = calculateOverall(task.Steps)
		if stepErr != nil {
			task.Steps[stepIndex].Logs = append(task.Steps[stepIndex].Logs, stepErr.Error())
		}
		e.broadcastProgress(*task, task.Steps[stepIndex])
		return nil
	})
}

func (e *Executor) recordCommandOutput(deploymentID, serverID, stepID string, result collect.CommandResult) {
	lines := []string{}
	if strings.TrimSpace(result.Command) != "" {
		lines = append(lines, "$ "+strings.TrimSpace(result.Command))
	}
	if strings.TrimSpace(result.Stdout) != "" {
		lines = append(lines, strings.Split(strings.TrimSpace(result.Stdout), "\n")...)
	}
	if strings.TrimSpace(result.Stderr) != "" {
		lines = append(lines, strings.Split(strings.TrimSpace(result.Stderr), "\n")...)
	}
	if len(lines) == 0 {
		return
	}

	_ = e.store.Update(func(data *domain.Data) error {
		taskIdx := findTask(data.Tasks, deploymentID, serverID)
		if taskIdx < 0 {
			return nil
		}
		for i := range data.Tasks[taskIdx].Steps {
			if data.Tasks[taskIdx].Steps[i].ID != stepID {
				continue
			}
			data.Tasks[taskIdx].Steps[i].Logs = append(data.Tasks[taskIdx].Steps[i].Logs, lines...)
			break
		}
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
		if len(data.Logs) > 4000 {
			data.Logs = data.Logs[len(data.Logs)-4000:]
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
		if dep.Status == "stopped" || dep.Status == "failed" {
			deployment = *dep
			return nil
		}
		dep.Status = "running"
		dep.UpdatedAt = domain.Now()
		dep.Endpoints = makeEndpoints(*dep, data.Servers)
		dep.Metrics = &domain.DeploymentMetrics{}
		deployment = *dep
		return nil
	})

	e.addLog(deploymentID, "", "", "info", "deployment is running")
	e.broadcastStatus(deployment)
}

func (e *Executor) failDeployment(deploymentID string, runErr error) {
	var deployment domain.DeploymentConfig
	_ = e.store.Update(func(data *domain.Data) error {
		idx := findDeployment(data.Deployments, deploymentID)
		if idx < 0 {
			return nil
		}
		data.Deployments[idx].Status = "failed"
		data.Deployments[idx].UpdatedAt = domain.Now()
		for i := range data.Deployments[idx].Endpoints {
			data.Deployments[idx].Endpoints[i].Status = "unknown"
		}
		deployment = data.Deployments[idx]
		return nil
	})
	if runErr != nil {
		e.addLog(deploymentID, "", "", "error", runErr.Error())
	}
	e.broadcastStatus(deployment)
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

func deploymentServers(all []domain.ServerConfig, ids []string) ([]domain.ServerConfig, error) {
	servers := make([]domain.ServerConfig, 0, len(ids))
	for _, id := range ids {
		server, ok := getServer(all, id)
		if !ok {
			return nil, fmt.Errorf("server %q not found", id)
		}
		servers = append(servers, server)
	}
	return servers, nil
}

func makeEndpoints(deployment domain.DeploymentConfig, servers []domain.ServerConfig) []domain.DeploymentEndpoint {
	serverIDs := deployment.Servers
	if strings.EqualFold(strings.TrimSpace(deployment.Framework), "vllm-ascend") && deployment.Ray.Enabled {
		head := pickRayHeadServer(deployment, servers)
		if head.ID != "" {
			serverIDs = []string{head.ID}
		}
	}

	endpoints := make([]domain.DeploymentEndpoint, 0, len(serverIDs))
	for _, serverID := range serverIDs {
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
		})
	}
	return endpoints
}

func resolveJumpHost(data domain.Data, server domain.ServerConfig) (*collect.SSHConfig, error) {
	if collect.IsMockServer(server) || !server.UseJumpHost {
		return nil, nil
	}
	if server.JumpHostID == "" {
		return nil, errors.New("jumpHostId is required when useJumpHost is true")
	}
	if server.JumpHostID == server.ID {
		return nil, errors.New("server cannot use itself as jump host")
	}
	for _, candidate := range data.Servers {
		if candidate.ID == server.JumpHostID {
			config := collect.FromServer(candidate)
			return &config, nil
		}
	}
	for _, candidate := range data.JumpHosts {
		if candidate.ID == server.JumpHostID {
			config := collect.FromJumpHost(candidate)
			return &config, nil
		}
	}
	return nil, errors.New("jump host not found")
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

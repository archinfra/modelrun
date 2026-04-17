package deploy

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"modelrun/backend/internal/collect"
	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/logging"
	"modelrun/backend/internal/realtime"
	"modelrun/backend/internal/runstate"
	"modelrun/backend/internal/runtimefiles"
	"modelrun/backend/internal/store"
)

type Executor struct {
	store     *store.Store
	state     *runstate.State
	hub       *realtime.Hub
	collector *collect.Collector
	files     *runtimefiles.Manager
	activeMu  sync.Mutex
	active    map[string]map[string]func()
	runtimeMu sync.RWMutex
	runtime   map[string]*runtimeStepOutput
}

type runtimeStepOutput struct {
	DeploymentID string
	ServerID     string
	StepID       string
	Lines        []string
}

const maxRuntimeLinesPerStep = 2000

func NewExecutor(st *store.Store, state *runstate.State, hub *realtime.Hub, files *runtimefiles.Manager) *Executor {
	return &Executor{
		store:     st,
		state:     state,
		hub:       hub,
		collector: collect.New(),
		files:     files,
		active:    map[string]map[string]func(){},
		runtime:   map[string]*runtimeStepOutput{},
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
	tasks := make([]domain.DeploymentTask, 0, len(serverList))
	for _, server := range serverList {
		plan, err := buildPlanWithStepConfigs(deployment, server, serverList, snapshot.PipelineSteps)
		if err != nil {
			return nil, err
		}
		plans[server.ID] = plan
		tasks = append(tasks, domain.DeploymentTask{
			ID:           domain.NewID("task"),
			DeploymentID: deploymentID,
			ServerID:     server.ID,
			Steps:        stepsFromPlan(plan),
		})
	}

	taskIDs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
	}
	if err := e.files.ResetDeployment(deploymentID); err != nil {
		logging.Warnf("deploy", "reset runtime files for %s failed: %v", deploymentID, err)
	}
	e.clearRuntimeLogs(deploymentID)
	e.state.DeleteDeployment(deploymentID)
	e.state.SetTasks(deploymentID, tasks)
	e.state.SetDeploymentRuntime(deploymentID, "deploying", nil, nil)
	updated := e.state.OverlayDeployment(deployment)
	logging.Infof("deploy", "start deployment %s on %d server(s)", deploymentID, len(serverList))
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

	e.cancelDeploymentCommands(deploymentID)
	drained := e.drainDeploymentRuntimeLogs(deploymentID)
	stoppedTasks := e.state.Tasks(deploymentID)
	now := domain.Now()
	for i := range stoppedTasks {
		for j := range stoppedTasks[i].Steps {
			if lines := drained[taskStepKey(deploymentID, stoppedTasks[i].ServerID, stoppedTasks[i].Steps[j].ID)]; len(lines) > 0 {
				stoppedTasks[i].Steps[j].Logs = append(stoppedTasks[i].Steps[j].Logs, lines...)
			}
			switch stoppedTasks[i].Steps[j].Status {
			case "completed", "failed", "stopped":
				continue
			default:
				stoppedTasks[i].Steps[j].Status = "stopped"
				if stoppedTasks[i].Steps[j].StartTime == "" {
					stoppedTasks[i].Steps[j].StartTime = now
				}
				stoppedTasks[i].Steps[j].EndTime = now
			}
		}
		stoppedTasks[i].OverallProgress = calculateOverall(stoppedTasks[i].Steps)
	}
	e.state.SetTasks(deploymentID, stoppedTasks)
	e.state.SetDeploymentRuntime(deploymentID, "stopped", nil, nil)

	e.addLog(deploymentID, "", "", "warn", "deployment stopped")
	logging.Warnf("deploy", "deployment %s stopped by user", deploymentID)
	stopped := e.state.OverlayDeployment(deployment)
	for _, task := range stoppedTasks {
		for _, step := range task.Steps {
			if step.Status == "stopped" {
				e.broadcastProgress(task, step)
			}
		}
	}
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
		logging.Errorf("deploy", "resolve deployment %s servers failed: %v", deploymentID, err)
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
				logging.Errorf("deploy", "deployment %s server %s failed: %v", deploymentID, firstNonEmpty(server.Name, server.ID), err)
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

	plan, err := buildPlanWithStepConfigs(deployment, server, servers, snapshot.PipelineSteps)
	if err != nil {
		return err
	}

	for stepIndex, step := range plan {
		if failed.Load() || e.isStopped(deployment.ID) {
			return nil
		}

		e.startStep(deployment.ID, server.ID, stepIndex)
		e.addLog(deployment.ID, server.ID, step.step.ID, "info", "starting "+step.step.Name)
		commandPreview := strings.TrimSpace(step.step.CommandPreview)
		if commandPreview == "" {
			commandPreview = strings.TrimSpace(step.command)
		}
		e.writeStepMeta(deployment.ID, server.ID, step.step, commandPreview, step.command)
		logging.Infof("deploy", "deployment=%s server=%s step=%s start", deployment.ID, firstNonEmpty(server.Name, server.ID), step.step.ID)
		e.appendStepOutput(deployment.ID, server.ID, step.step.ID, []string{"$ " + commandPreview})
		stopCh, releaseStop := e.registerCommand(deployment.ID)
		_, err := e.collector.RunCommandStreamCancelable(server, jump, step.command, stopCh, func(item collect.CommandStreamLine) {
			line := strings.TrimRight(item.Line, "\r")
			if strings.TrimSpace(line) == "" {
				return
			}
			e.appendStepOutput(deployment.ID, server.ID, step.step.ID, []string{line})
		})
		releaseStop()
		if err != nil {
			if errors.Is(err, collect.ErrCommandCancelled) && e.isStopped(deployment.ID) {
				e.addLog(deployment.ID, server.ID, step.step.ID, "warn", "step stopped by user")
				logging.Warnf("deploy", "deployment=%s server=%s step=%s stopped", deployment.ID, firstNonEmpty(server.Name, server.ID), step.step.ID)
				return nil
			}
			e.addLog(deployment.ID, server.ID, step.step.ID, "error", err.Error())
			e.failStep(deployment.ID, server.ID, step.step.ID, stepIndex, err)
			logging.Errorf("deploy", "deployment=%s server=%s step=%s failed: %v", deployment.ID, firstNonEmpty(server.Name, server.ID), step.step.ID, err)
			return err
		}

		e.addLog(deployment.ID, server.ID, step.step.ID, "info", "completed "+step.step.Name)
		e.completeStep(deployment.ID, server.ID, step.step.ID, stepIndex)
		logging.Infof("deploy", "deployment=%s server=%s step=%s completed", deployment.ID, firstNonEmpty(server.Name, server.ID), step.step.ID)
	}

	return nil
}

func (e *Executor) registerCommand(deploymentID string) (<-chan struct{}, func()) {
	stopCh := make(chan struct{})
	cancel := sync.OnceFunc(func() {
		close(stopCh)
	})
	token := domain.NewID("cmd")

	e.activeMu.Lock()
	if e.active[deploymentID] == nil {
		e.active[deploymentID] = map[string]func(){}
	}
	e.active[deploymentID][token] = cancel
	e.activeMu.Unlock()

	return stopCh, func() {
		e.activeMu.Lock()
		if entries, ok := e.active[deploymentID]; ok {
			delete(entries, token)
			if len(entries) == 0 {
				delete(e.active, deploymentID)
			}
		}
		e.activeMu.Unlock()
	}
}

func (e *Executor) cancelDeploymentCommands(deploymentID string) {
	e.activeMu.Lock()
	entries := e.active[deploymentID]
	delete(e.active, deploymentID)
	e.activeMu.Unlock()

	for _, cancel := range entries {
		cancel()
	}
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
	updated := e.state.UpdateTask(deploymentID, serverID, func(task *domain.DeploymentTask) {
		if stepIndex >= len(task.Steps) {
			return
		}
		task.CurrentStep = stepIndex
		task.Steps[stepIndex].Status = "running"
		task.Steps[stepIndex].StartTime = now
		task.Steps[stepIndex].Progress = 20
		task.OverallProgress = calculateOverall(task.Steps)
	})
	if updated {
		tasks := e.state.Tasks(deploymentID)
		if task, ok := findRuntimeTask(tasks, deploymentID, serverID); ok && stepIndex < len(task.Steps) {
			e.broadcastProgress(task, task.Steps[stepIndex])
		}
	}
}

func (e *Executor) completeStep(deploymentID, serverID, stepID string, stepIndex int) {
	drained := e.drainRuntimeStepLogs(deploymentID, serverID, stepID)
	now := domain.Now()
	updated := e.state.UpdateTask(deploymentID, serverID, func(task *domain.DeploymentTask) {
		if stepIndex >= len(task.Steps) {
			return
		}
		task.Steps[stepIndex].Logs = append(task.Steps[stepIndex].Logs, drained...)
		task.Steps[stepIndex].Status = "completed"
		task.Steps[stepIndex].Progress = 100
		task.Steps[stepIndex].EndTime = now
		task.OverallProgress = calculateOverall(task.Steps)
	})
	if updated {
		tasks := e.state.Tasks(deploymentID)
		if task, ok := findRuntimeTask(tasks, deploymentID, serverID); ok && stepIndex < len(task.Steps) {
			e.broadcastProgress(task, task.Steps[stepIndex])
		}
	}
}

func (e *Executor) failStep(deploymentID, serverID, stepID string, stepIndex int, stepErr error) {
	drained := e.drainRuntimeStepLogs(deploymentID, serverID, stepID)
	now := domain.Now()
	updated := e.state.UpdateTask(deploymentID, serverID, func(task *domain.DeploymentTask) {
		if stepIndex >= len(task.Steps) {
			return
		}
		task.Steps[stepIndex].Logs = append(task.Steps[stepIndex].Logs, drained...)
		task.Steps[stepIndex].Status = "failed"
		task.Steps[stepIndex].Progress = 100
		task.Steps[stepIndex].EndTime = now
		task.OverallProgress = calculateOverall(task.Steps)
		if stepErr != nil && len(drained) == 0 {
			task.Steps[stepIndex].Logs = append(task.Steps[stepIndex].Logs, stepErr.Error())
		}
	})
	if updated {
		tasks := e.state.Tasks(deploymentID)
		if task, ok := findRuntimeTask(tasks, deploymentID, serverID); ok && stepIndex < len(task.Steps) {
			e.broadcastProgress(task, task.Steps[stepIndex])
		}
	}
}

func (e *Executor) appendStepOutput(deploymentID, serverID, stepID string, lines []string) {
	if len(lines) == 0 {
		return
	}
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) == 0 {
		return
	}
	e.appendRuntimeStepLogs(deploymentID, serverID, stepID, filtered)
	if err := e.files.AppendStepLines(deploymentID, serverID, stepID, filtered); err != nil {
		logging.Warnf("deploy", "append runtime file failed deployment=%s server=%s step=%s: %v", deploymentID, serverID, stepID, err)
	}

	e.hub.Broadcast(realtime.Message{
		Type:         "step_log",
		DeploymentID: deploymentID,
		Data: map[string]any{
			"serverId": serverID,
			"stepId":   stepID,
			"lines":    filtered,
		},
	})
}

func (e *Executor) appendRuntimeStepLogs(deploymentID, serverID, stepID string, lines []string) {
	key := taskStepKey(deploymentID, serverID, stepID)
	e.runtimeMu.Lock()
	entry := e.runtime[key]
	if entry == nil {
		entry = &runtimeStepOutput{
			DeploymentID: deploymentID,
			ServerID:     serverID,
			StepID:       stepID,
		}
		e.runtime[key] = entry
	}
	entry.Lines = append(entry.Lines, lines...)
	if len(entry.Lines) > maxRuntimeLinesPerStep {
		entry.Lines = append([]string{}, entry.Lines[len(entry.Lines)-maxRuntimeLinesPerStep:]...)
	}
	e.runtimeMu.Unlock()
}

func (e *Executor) drainRuntimeStepLogs(deploymentID, serverID, stepID string) []string {
	key := taskStepKey(deploymentID, serverID, stepID)
	e.runtimeMu.Lock()
	defer e.runtimeMu.Unlock()
	entry := e.runtime[key]
	if entry == nil {
		return nil
	}
	delete(e.runtime, key)
	return append([]string{}, entry.Lines...)
}

func (e *Executor) drainDeploymentRuntimeLogs(deploymentID string) map[string][]string {
	e.runtimeMu.Lock()
	defer e.runtimeMu.Unlock()
	drained := map[string][]string{}
	for key, entry := range e.runtime {
		if entry.DeploymentID != deploymentID {
			continue
		}
		drained[key] = append([]string{}, entry.Lines...)
		delete(e.runtime, key)
	}
	return drained
}

func (e *Executor) clearRuntimeLogs(deploymentID string) {
	e.runtimeMu.Lock()
	defer e.runtimeMu.Unlock()
	for key, entry := range e.runtime {
		if entry.DeploymentID == deploymentID {
			delete(e.runtime, key)
		}
	}
}

func (e *Executor) HydrateTasks(tasks []domain.DeploymentTask) []domain.DeploymentTask {
	e.runtimeMu.RLock()
	defer e.runtimeMu.RUnlock()

	out := make([]domain.DeploymentTask, 0, len(tasks))
	for _, task := range tasks {
		copyTask := task
		copyTask.Steps = append([]domain.DeploymentStep(nil), task.Steps...)
		for i := range copyTask.Steps {
			if len(task.Steps[i].Logs) > 0 {
				copyTask.Steps[i].Logs = append([]string{}, task.Steps[i].Logs...)
			} else {
				copyTask.Steps[i].Logs = e.files.ReadStepTail(task.DeploymentID, task.ServerID, copyTask.Steps[i].ID, maxRuntimeLinesPerStep)
			}
			if entry := e.runtime[taskStepKey(task.DeploymentID, task.ServerID, copyTask.Steps[i].ID)]; entry != nil {
				copyTask.Steps[i].Logs = append(copyTask.Steps[i].Logs, entry.Lines...)
			}
		}
		out = append(out, copyTask)
	}
	return out
}

func taskStepKey(deploymentID, serverID, stepID string) string {
	return deploymentID + "|" + serverID + "|" + stepID
}

func findRuntimeTask(tasks []domain.DeploymentTask, deploymentID, serverID string) (domain.DeploymentTask, bool) {
	for _, task := range tasks {
		if task.DeploymentID == deploymentID && task.ServerID == serverID {
			return task, true
		}
	}
	return domain.DeploymentTask{}, false
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

	e.state.AddLog(entry)

	if stepID != "" {
		e.appendStepOutput(deploymentID, serverID, stepID, []string{message})
	}

	e.hub.Broadcast(realtime.Message{
		Type:         "log",
		DeploymentID: deploymentID,
		Data:         entry,
	})
}

func (e *Executor) completeDeployment(deploymentID string) {
	snapshot := e.store.Snapshot()
	deployment, ok := getDeployment(snapshot.Deployments, deploymentID)
	if !ok {
		return
	}
	if status := e.state.DeploymentStatus(deploymentID); status == "stopped" || status == "failed" {
		return
	}
	e.state.SetDeploymentRuntime(deploymentID, "running", makeEndpoints(deployment, snapshot.Servers), &domain.DeploymentMetrics{})
	deployment = e.state.OverlayDeployment(deployment)

	e.addLog(deploymentID, "", "", "info", "deployment is running")
	logging.Infof("deploy", "deployment %s is running", deploymentID)
	e.broadcastStatus(deployment)
}

func (e *Executor) failDeployment(deploymentID string, runErr error) {
	snapshot := e.store.Snapshot()
	deployment, ok := getDeployment(snapshot.Deployments, deploymentID)
	if !ok {
		return
	}
	e.state.SetDeploymentRuntime(deploymentID, "failed", nil, nil)
	deployment = e.state.OverlayDeployment(deployment)
	if runErr != nil {
		e.addLog(deploymentID, "", "", "error", runErr.Error())
		logging.Errorf("deploy", "deployment %s failed: %v", deploymentID, runErr)
	}
	e.broadcastStatus(deployment)
}

func (e *Executor) writeStepMeta(deploymentID, serverID string, step domain.DeploymentStep, commandPreview, command string) {
	if err := e.files.WriteStepMeta(runtimefiles.StepMeta{
		DeploymentID:   deploymentID,
		ServerID:       serverID,
		StepID:         step.ID,
		StepName:       step.Name,
		Description:    step.Description,
		CommandPreview: commandPreview,
		Command:        command,
		StartedAt:      domain.Now(),
	}); err != nil {
		logging.Warnf("deploy", "write step meta failed deployment=%s server=%s step=%s: %v", deploymentID, serverID, step.ID, err)
	}
}

func (e *Executor) isStopped(deploymentID string) bool {
	return e.state.DeploymentStatus(deploymentID) == "stopped"
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

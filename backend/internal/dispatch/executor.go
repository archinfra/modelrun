package dispatch

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"modelrun/backend/internal/catalog"
	"modelrun/backend/internal/collect"
	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/runstate"
	"modelrun/backend/internal/store"
)

type Executor struct {
	store     *store.Store
	state     *runstate.State
	collector *collect.Collector
}

type CreateTaskRequest struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	ProjectID     string            `json:"projectId"`
	Scope         string            `json:"scope"`
	ExecutionType string            `json:"executionType"`
	Command       string            `json:"command"`
	ScriptURL     string            `json:"scriptUrl"`
	ScriptArgs    string            `json:"scriptArgs"`
	PresetID      string            `json:"presetId"`
	PresetArgs    map[string]string `json:"presetArgs"`
	ServerIDs     []string          `json:"serverIds"`
}

func New(st *store.Store, state *runstate.State, collector *collect.Collector) *Executor {
	return &Executor{store: st, state: state, collector: collector}
}

func (e *Executor) Presets() []domain.RemoteTaskPreset {
	snapshot := e.store.Snapshot()
	items := make([]domain.RemoteTaskPreset, 0, len(snapshot.ActionTemplates))
	for _, action := range snapshot.ActionTemplates {
		items = append(items, catalog.ToRemoteTaskPreset(action))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (e *Executor) Create(req CreateTaskRequest) (domain.RemoteTask, error) {
	snapshot := e.store.Snapshot()

	servers, err := resolveTargetServers(snapshot, req.Scope, req.ProjectID, req.ServerIDs)
	if err != nil {
		return domain.RemoteTask{}, err
	}

	commandPreview, err := e.resolveCommand(snapshot, req)
	if err != nil {
		return domain.RemoteTask{}, err
	}

	task := domain.RemoteTask{
		ID:             domain.NewID("remote_task"),
		Name:           e.defaultTaskName(snapshot, req, servers),
		Description:    strings.TrimSpace(req.Description),
		ProjectID:      strings.TrimSpace(req.ProjectID),
		Scope:          normalizeScope(req.Scope, req.ProjectID, req.ServerIDs),
		Status:         "pending",
		ExecutionType:  normalizeExecutionType(req.ExecutionType),
		CommandPreview: commandPreview,
		ScriptURL:      strings.TrimSpace(req.ScriptURL),
		ScriptArgs:     strings.TrimSpace(req.ScriptArgs),
		PresetID:       strings.TrimSpace(req.PresetID),
		PresetArgs:     cloneStringMap(req.PresetArgs),
		ServerIDs:      make([]string, 0, len(servers)),
		Runs:           make([]domain.RemoteTaskRun, 0, len(servers)),
		CreatedAt:      domain.Now(),
	}

	if strings.TrimSpace(req.Name) != "" {
		task.Name = strings.TrimSpace(req.Name)
	}
	for _, server := range servers {
		task.ServerIDs = append(task.ServerIDs, server.ID)
		task.Runs = append(task.Runs, domain.RemoteTaskRun{
			ServerID:   server.ID,
			ServerName: server.Name,
			Status:     "pending",
			Command:    commandPreview,
		})
	}

	e.state.PutRemoteTask(task)

	go e.run(task.ID)

	return task, nil
}

func (e *Executor) run(taskID string) {
	e.markTaskRunning(taskID)

	task, ok := e.state.RemoteTask(taskID)
	if !ok {
		return
	}
	snapshot := e.store.Snapshot()

	var wg sync.WaitGroup
	for _, run := range task.Runs {
		server, ok := findServer(snapshot.Servers, run.ServerID)
		if !ok {
			e.finishRun(taskID, run.ServerID, collect.CommandResult{Command: task.CommandPreview}, errors.New("server not found"))
			continue
		}

		wg.Add(1)
		go func(server domain.ServerConfig) {
			defer wg.Done()
			e.executeRun(taskID, server)
		}(server)
	}

	wg.Wait()
	e.markTaskFinished(taskID)
}

func (e *Executor) executeRun(taskID string, server domain.ServerConfig) {
	startedAt := domain.Now()
	e.state.UpdateRemoteTask(taskID, func(task *domain.RemoteTask) {
		runIdx := findRemoteTaskRun(task.Runs, server.ID)
		if runIdx < 0 {
			return
		}
		task.Runs[runIdx].Status = "running"
		task.Runs[runIdx].StartedAt = startedAt
		task.Runs[runIdx].Command = task.CommandPreview
		if task.StartedAt == "" {
			task.StartedAt = startedAt
		}
		task.Status = "running"
	})

	snapshot := e.store.Snapshot()
	latestServer, ok := findServer(snapshot.Servers, server.ID)
	if ok {
		server = latestServer
	}
	task, ok := e.state.RemoteTask(taskID)
	if !ok {
		return
	}

	jump, err := resolveJumpHost(snapshot, server)
	if err != nil {
		e.finishRun(taskID, server.ID, collect.CommandResult{Command: task.CommandPreview}, err)
		return
	}

	result, err := e.collector.RunCommand(server, jump, task.CommandPreview)
	e.finishRun(taskID, server.ID, result, err)
}

func (e *Executor) markTaskRunning(taskID string) {
	startedAt := domain.Now()
	e.state.UpdateRemoteTask(taskID, func(task *domain.RemoteTask) {
		if task.StartedAt == "" {
			task.StartedAt = startedAt
		}
		task.Status = "running"
	})
}

func (e *Executor) finishRun(taskID, serverID string, result collect.CommandResult, runErr error) {
	finishedAt := domain.Now()
	e.state.UpdateRemoteTask(taskID, func(task *domain.RemoteTask) {
		runIdx := findRemoteTaskRun(task.Runs, serverID)
		if runIdx < 0 {
			return
		}
		run := &task.Runs[runIdx]
		run.Command = firstNonEmpty(strings.TrimSpace(result.Command), run.Command)
		run.Output = firstNonEmpty(strings.TrimSpace(result.Stdout), strings.TrimSpace(result.Stderr))
		run.Error = ""
		run.ExitCode = result.ExitCode
		run.FinishedAt = finishedAt
		if run.StartedAt == "" {
			run.StartedAt = finishedAt
		}
		if runErr != nil {
			run.Status = "failed"
			run.Error = runErr.Error()
			if result.Stderr != "" {
				run.Output = strings.TrimSpace(result.Stderr)
			}
		} else {
			run.Status = "completed"
		}
		task.Status = "running"
	})
}

func (e *Executor) markTaskFinished(taskID string) {
	finishedAt := domain.Now()
	e.state.UpdateRemoteTask(taskID, func(task *domain.RemoteTask) {
		completed := 0
		failed := 0
		for _, run := range task.Runs {
			switch run.Status {
			case "completed":
				completed++
			case "failed":
				failed++
			}
		}

		switch {
		case completed == len(task.Runs):
			task.Status = "completed"
		case failed == len(task.Runs):
			task.Status = "failed"
		default:
			task.Status = "partial"
		}
		task.FinishedAt = finishedAt
		if task.StartedAt == "" {
			task.StartedAt = finishedAt
		}
	})
}

func resolveTargetServers(data domain.Data, scope, projectID string, serverIDs []string) ([]domain.ServerConfig, error) {
	scope = normalizeScope(scope, projectID, serverIDs)
	servers := make([]domain.ServerConfig, 0)

	switch scope {
	case "all":
		servers = append(servers, data.Servers...)
	case "project":
		projectID = strings.TrimSpace(projectID)
		if projectID == "" {
			return nil, errors.New("projectId is required for project scoped tasks")
		}
		for _, server := range data.Servers {
			if server.ProjectID == projectID {
				servers = append(servers, server)
			}
		}
	case "selected":
		if len(serverIDs) == 0 {
			return nil, errors.New("serverIds is required for selected scope")
		}
		indexed := map[string]domain.ServerConfig{}
		for _, server := range data.Servers {
			indexed[server.ID] = server
		}
		missing := []string{}
		for _, serverID := range serverIDs {
			server, ok := indexed[serverID]
			if !ok {
				missing = append(missing, serverID)
				continue
			}
			servers = append(servers, server)
		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("unknown serverIds: %s", strings.Join(missing, ", "))
		}
	default:
		return nil, fmt.Errorf("unsupported scope %q", scope)
	}

	if len(servers) == 0 {
		return nil, errors.New("no servers matched the selected scope")
	}

	sort.Slice(servers, func(i, j int) bool {
		left := firstNonEmpty(servers[i].Name, servers[i].Host, servers[i].ID)
		right := firstNonEmpty(servers[j].Name, servers[j].Host, servers[j].ID)
		return left < right
	})
	return servers, nil
}

func (e *Executor) resolveCommand(snapshot domain.Data, req CreateTaskRequest) (string, error) {
	switch normalizeExecutionType(req.ExecutionType) {
	case "command":
		command := strings.TrimSpace(req.Command)
		if command == "" {
			return "", errors.New("command is required")
		}
		return command, nil
	case "script_url":
		return collect.BuildScriptURLCommand(req.ScriptURL, req.ScriptArgs)
	case "preset":
		action, ok := catalog.LookupActionTemplate(snapshot.ActionTemplates, req.PresetID)
		if !ok {
			return "", fmt.Errorf("unknown preset %q", req.PresetID)
		}
		return catalog.BuildActionCommand(action, req.PresetArgs)
	default:
		return "", fmt.Errorf("unsupported executionType %q", req.ExecutionType)
	}
}

func (e *Executor) defaultTaskName(snapshot domain.Data, req CreateTaskRequest, servers []domain.ServerConfig) string {
	switch normalizeExecutionType(req.ExecutionType) {
	case "preset":
		if action, ok := catalog.LookupActionTemplate(snapshot.ActionTemplates, req.PresetID); ok {
			return fmt.Sprintf("%s (%d)", action.Name, len(servers))
		}
	case "script_url":
		return fmt.Sprintf("Run remote script (%d)", len(servers))
	default:
		command := strings.TrimSpace(req.Command)
		if command != "" {
			line := strings.Split(command, "\n")[0]
			if len(line) > 42 {
				line = line[:42] + "..."
			}
			return fmt.Sprintf("%s (%d)", line, len(servers))
		}
	}
	return fmt.Sprintf("Remote task (%d)", len(servers))
}

func normalizeScope(scope, projectID string, serverIDs []string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope != "" {
		return scope
	}
	if len(serverIDs) > 0 {
		return "selected"
	}
	if strings.TrimSpace(projectID) != "" {
		return "project"
	}
	return "all"
}

func normalizeExecutionType(executionType string) string {
	executionType = strings.ToLower(strings.TrimSpace(executionType))
	if executionType == "" {
		return "command"
	}
	return executionType
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

func findRemoteTaskRun(runs []domain.RemoteTaskRun, serverID string) int {
	for i, run := range runs {
		if run.ServerID == serverID {
			return i
		}
	}
	return -1
}

func findServer(servers []domain.ServerConfig, id string) (domain.ServerConfig, bool) {
	for _, server := range servers {
		if server.ID == id {
			return server, true
		}
	}
	return domain.ServerConfig{}, false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

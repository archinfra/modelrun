package runstate

import (
	"sync"

	"modelrun/backend/internal/domain"
)

type State struct {
	mu          sync.RWMutex
	deployments map[string]deploymentRuntime
	tasks       map[string][]domain.DeploymentTask
	logs        []domain.DeploymentLog
	servers     map[string]serverRuntime
	remoteTasks map[string]domain.RemoteTask
}

type deploymentRuntime struct {
	Status    string
	Endpoints []domain.DeploymentEndpoint
	Metrics   *domain.DeploymentMetrics
}

type serverRuntime struct {
	GPUInfo              []domain.GPUInfo
	DriverVersion        string
	CUDAVersion          string
	DockerVersion        string
	NPUExporterEndpoint  string
	NPUExporterStatus    string
	NPUExporterLastCheck string
	NetdataEndpoint      string
	NetdataStatus        string
	NetdataLastCheck     string
	LastCheck            string
	Status               string
}

func New() *State {
	return &State{
		deployments: map[string]deploymentRuntime{},
		tasks:       map[string][]domain.DeploymentTask{},
		logs:        []domain.DeploymentLog{},
		servers:     map[string]serverRuntime{},
		remoteTasks: map[string]domain.RemoteTask{},
	}
}

func (s *State) OverlayDeployments(items []domain.DeploymentConfig) []domain.DeploymentConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.DeploymentConfig, 0, len(items))
	for _, item := range items {
		out = append(out, s.overlayDeploymentLocked(item))
	}
	return out
}

func (s *State) OverlayDeployment(item domain.DeploymentConfig) domain.DeploymentConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.overlayDeploymentLocked(item)
}

func (s *State) overlayDeploymentLocked(item domain.DeploymentConfig) domain.DeploymentConfig {
	if runtime, ok := s.deployments[item.ID]; ok {
		if runtime.Status != "" {
			item.Status = runtime.Status
		}
		if runtime.Endpoints != nil {
			item.Endpoints = append([]domain.DeploymentEndpoint{}, runtime.Endpoints...)
		}
		if runtime.Metrics != nil {
			metrics := *runtime.Metrics
			item.Metrics = &metrics
		} else {
			item.Metrics = nil
		}
	}
	return item
}

func (s *State) SetDeploymentRuntime(id, status string, endpoints []domain.DeploymentEndpoint, metrics *domain.DeploymentMetrics) {
	s.mu.Lock()
	defer s.mu.Unlock()

	runtime := s.deployments[id]
	runtime.Status = status
	if endpoints != nil {
		runtime.Endpoints = append([]domain.DeploymentEndpoint{}, endpoints...)
	} else {
		runtime.Endpoints = nil
	}
	if metrics != nil {
		copyMetrics := *metrics
		runtime.Metrics = &copyMetrics
	} else {
		runtime.Metrics = nil
	}
	s.deployments[id] = runtime
}

func (s *State) DeploymentStatus(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.deployments[id].Status
}

func (s *State) DeploymentMetrics(id string) *domain.DeploymentMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	runtime, ok := s.deployments[id]
	if !ok || runtime.Metrics == nil {
		return nil
	}
	copyMetrics := *runtime.Metrics
	return &copyMetrics
}

func (s *State) DeleteDeployment(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.deployments, id)
	delete(s.tasks, id)
	filtered := s.logs[:0]
	for _, item := range s.logs {
		if item.DeploymentID != id {
			filtered = append(filtered, item)
		}
	}
	s.logs = append([]domain.DeploymentLog{}, filtered...)
}

func (s *State) SetTasks(deploymentID string, tasks []domain.DeploymentTask) {
	s.mu.Lock()
	defer s.mu.Unlock()

	copied := make([]domain.DeploymentTask, 0, len(tasks))
	for _, task := range tasks {
		copied = append(copied, cloneTask(task))
	}
	s.tasks[deploymentID] = copied
}

func (s *State) Tasks(deploymentID string) []domain.DeploymentTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.tasks[deploymentID]
	out := make([]domain.DeploymentTask, 0, len(items))
	for _, item := range items {
		out = append(out, cloneTask(item))
	}
	return out
}

func (s *State) TaskByID(id string) (domain.DeploymentTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, items := range s.tasks {
		for _, item := range items {
			if item.ID == id {
				return cloneTask(item), true
			}
		}
	}
	return domain.DeploymentTask{}, false
}

func (s *State) UpdateTask(deploymentID, serverID string, fn func(*domain.DeploymentTask)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := s.tasks[deploymentID]
	for i := range items {
		if items[i].ServerID != serverID {
			continue
		}
		fn(&items[i])
		s.tasks[deploymentID] = items
		return true
	}
	return false
}

func (s *State) AddLog(entry domain.DeploymentLog) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs = append(s.logs, entry)
	if len(s.logs) > 4000 {
		s.logs = append([]domain.DeploymentLog{}, s.logs[len(s.logs)-4000:]...)
	}
}

func (s *State) Logs(deploymentID, serverID, stepID string, limit int) []domain.DeploymentLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 200
	}
	items := make([]domain.DeploymentLog, 0, len(s.logs))
	for _, item := range s.logs {
		if deploymentID != "" && item.DeploymentID != deploymentID {
			continue
		}
		if serverID != "" && item.ServerID != serverID {
			continue
		}
		if stepID != "" && item.StepID != stepID {
			continue
		}
		items = append(items, item)
	}
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	out := make([]domain.DeploymentLog, 0, len(items))
	out = append(out, items...)
	return out
}

func (s *State) SetServerRuntime(serverID string, item domain.ServerConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.servers[serverID] = serverRuntime{
		GPUInfo:              append([]domain.GPUInfo{}, item.GPUInfo...),
		DriverVersion:        item.DriverVersion,
		CUDAVersion:          item.CUDAVersion,
		DockerVersion:        item.DockerVersion,
		NPUExporterEndpoint:  item.NPUExporterEndpoint,
		NPUExporterStatus:    item.NPUExporterStatus,
		NPUExporterLastCheck: item.NPUExporterLastCheck,
		NetdataEndpoint:      item.NetdataEndpoint,
		NetdataStatus:        item.NetdataStatus,
		NetdataLastCheck:     item.NetdataLastCheck,
		LastCheck:            item.LastCheck,
		Status:               item.Status,
	}
}

func (s *State) DeleteServerRuntime(serverID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.servers, serverID)
}

func (s *State) OverlayServers(items []domain.ServerConfig) []domain.ServerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]domain.ServerConfig, 0, len(items))
	for _, item := range items {
		out = append(out, s.overlayServerLocked(item))
	}
	return out
}

func (s *State) OverlayServer(item domain.ServerConfig) domain.ServerConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.overlayServerLocked(item)
}

func (s *State) PutRemoteTask(item domain.RemoteTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.remoteTasks[item.ID] = cloneRemoteTask(item)
}

func (s *State) RemoteTasks(projectID string) []domain.RemoteTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]domain.RemoteTask, 0, len(s.remoteTasks))
	for _, item := range s.remoteTasks {
		if projectID != "" && item.ProjectID != projectID {
			continue
		}
		items = append(items, cloneRemoteTask(item))
	}
	return items
}

func (s *State) RemoteTask(id string) (domain.RemoteTask, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.remoteTasks[id]
	if !ok {
		return domain.RemoteTask{}, false
	}
	return cloneRemoteTask(item), true
}

func (s *State) UpdateRemoteTask(id string, fn func(*domain.RemoteTask)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.remoteTasks[id]
	if !ok {
		return false
	}
	fn(&item)
	s.remoteTasks[id] = cloneRemoteTask(item)
	return true
}

func (s *State) DeleteRemoteTask(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.remoteTasks, id)
}

func (s *State) overlayServerLocked(item domain.ServerConfig) domain.ServerConfig {
	if runtime, ok := s.servers[item.ID]; ok {
		item.GPUInfo = append([]domain.GPUInfo{}, runtime.GPUInfo...)
		item.DriverVersion = runtime.DriverVersion
		item.CUDAVersion = runtime.CUDAVersion
		item.DockerVersion = runtime.DockerVersion
		item.NPUExporterEndpoint = runtime.NPUExporterEndpoint
		item.NPUExporterStatus = runtime.NPUExporterStatus
		item.NPUExporterLastCheck = runtime.NPUExporterLastCheck
		item.NetdataEndpoint = runtime.NetdataEndpoint
		item.NetdataStatus = runtime.NetdataStatus
		item.NetdataLastCheck = runtime.NetdataLastCheck
		item.LastCheck = runtime.LastCheck
		item.Status = runtime.Status
	}
	return item
}

func cloneTask(item domain.DeploymentTask) domain.DeploymentTask {
	copyTask := item
	copyTask.Steps = append([]domain.DeploymentStep{}, item.Steps...)
	for i := range copyTask.Steps {
		copyTask.Steps[i].Logs = append([]string{}, item.Steps[i].Logs...)
	}
	return copyTask
}

func cloneRemoteTask(item domain.RemoteTask) domain.RemoteTask {
	copyTask := item
	copyTask.ServerIDs = append([]string{}, item.ServerIDs...)
	copyTask.Runs = append([]domain.RemoteTaskRun{}, item.Runs...)
	if len(item.PresetArgs) > 0 {
		copyTask.PresetArgs = make(map[string]string, len(item.PresetArgs))
		for key, value := range item.PresetArgs {
			copyTask.PresetArgs[key] = value
		}
	}
	return copyTask
}

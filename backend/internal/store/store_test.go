package store

import (
	"path/filepath"
	"testing"

	"modelrun/backend/internal/domain"
)

func TestStoreDoesNotPersistRuntimeState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "modelrun.db")

	st, err := New(path)
	if err != nil {
		t.Fatal(err)
	}

	err = st.Update(func(data *domain.Data) error {
		data.Projects = append(data.Projects, domain.Project{
			ID:   "project_1",
			Name: "demo",
		})
		data.Servers = append(data.Servers, domain.ServerConfig{
			ID:                   "server_1",
			Name:                 "gpu-node",
			Host:                 "10.0.0.10",
			Status:               "online",
			GPUInfo:              []domain.GPUInfo{{Name: "Ascend 910B", MemoryTotal: 65536}},
			DriverVersion:        "24.1",
			CUDAVersion:          "12.4",
			DockerVersion:        "26.0",
			NPUExporterEndpoint:  "http://10.0.0.10:8080/metrics",
			NPUExporterStatus:    "healthy",
			NPUExporterLastCheck: domain.Now(),
			LastCheck:            domain.Now(),
		})
		data.Deployments = append(data.Deployments, domain.DeploymentConfig{
			ID:        "deployment_1",
			Name:      "qwen",
			Status:    "running",
			Endpoints: []domain.DeploymentEndpoint{{ServerID: "server_1", URL: "http://10.0.0.10:8000/v1", Status: "ready"}},
			Metrics:   &domain.DeploymentMetrics{GPUUtilization: 72},
		})
		data.Tasks = append(data.Tasks, domain.DeploymentTask{
			ID:           "task_1",
			DeploymentID: "deployment_1",
		})
		data.RemoteTasks = append(data.RemoteTasks, domain.RemoteTask{
			ID:        "remote_task_1",
			Name:      "install exporter",
			Status:    "running",
			ServerIDs: []string{"server_1"},
			Runs:      []domain.RemoteTaskRun{{ServerID: "server_1", Status: "running"}},
			CreatedAt: domain.Now(),
		})
		data.Logs = append(data.Logs, domain.DeploymentLog{
			DeploymentID: "deployment_1",
			ServerID:     "server_1",
			Level:        "info",
			Message:      "runtime log",
			Timestamp:    domain.Now(),
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := reopened.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	snapshot := reopened.Snapshot()
	if len(snapshot.Projects) != 1 {
		t.Fatalf("expected projects to persist, got %d", len(snapshot.Projects))
	}
	if len(snapshot.Servers) != 1 {
		t.Fatalf("expected servers to persist, got %d", len(snapshot.Servers))
	}
	if len(snapshot.Deployments) != 1 {
		t.Fatalf("expected deployments to persist, got %d", len(snapshot.Deployments))
	}
	if got := snapshot.Servers[0].Status; got != "unknown" {
		t.Fatalf("server runtime status should be stripped, got %q", got)
	}
	if len(snapshot.Servers[0].GPUInfo) != 0 {
		t.Fatalf("server GPU info should not persist: %#v", snapshot.Servers[0].GPUInfo)
	}
	if got := snapshot.Deployments[0].Status; got != "draft" {
		t.Fatalf("deployment runtime status should reset to draft, got %q", got)
	}
	if snapshot.Deployments[0].Endpoints != nil {
		t.Fatalf("deployment endpoints should not persist: %#v", snapshot.Deployments[0].Endpoints)
	}
	if snapshot.Deployments[0].Metrics != nil {
		t.Fatalf("deployment metrics should not persist: %#v", snapshot.Deployments[0].Metrics)
	}
	if len(snapshot.Tasks) != 0 {
		t.Fatalf("deployment tasks should not persist, got %d", len(snapshot.Tasks))
	}
	if len(snapshot.RemoteTasks) != 0 {
		t.Fatalf("remote tasks should not persist, got %d", len(snapshot.RemoteTasks))
	}
	if len(snapshot.Logs) != 0 {
		t.Fatalf("logs should not persist, got %d", len(snapshot.Logs))
	}
}

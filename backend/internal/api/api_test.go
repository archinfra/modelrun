package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"modelrun/backend/internal/deploy"
	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/realtime"
	"modelrun/backend/internal/store"
)

func TestCreateModelRunResources(t *testing.T) {
	handler := newTestHandler(t)

	project := mustRequest[domain.Project](t, handler, http.MethodPost, "/api/projects", map[string]any{
		"name": "demo",
	})
	if project.ID == "" {
		t.Fatal("project id was not generated")
	}

	server := mustRequest[domain.ServerConfig](t, handler, http.MethodPost, "/api/servers", map[string]any{
		"projectId": project.ID,
		"name":      "gpu-node-01",
		"host":      "mock-node",
		"sshPort":   22,
		"username":  "root",
	})
	if server.ProjectID != project.ID {
		t.Fatalf("server project mismatch: got %q want %q", server.ProjectID, project.ID)
	}

	model := mustRequest[domain.ModelConfig](t, handler, http.MethodPost, "/api/models", map[string]any{
		"name":     "Qwen2.5-7B-Instruct",
		"source":   "modelscope",
		"modelId":  "Qwen/Qwen2.5-7B-Instruct",
		"revision": "main",
	})
	if model.Parameters != "7B" {
		t.Fatalf("model parameter inference failed: got %q", model.Parameters)
	}

	deployment := mustRequest[domain.DeploymentConfig](t, handler, http.MethodPost, "/api/deployments", map[string]any{
		"name":    "qwen-demo",
		"model":   model,
		"servers": []string{server.ID},
		"apiPort": 8000,
		"docker": map[string]any{
			"image": "vllm/vllm-openai",
			"tag":   "latest",
		},
	})
	if deployment.Status != "draft" {
		t.Fatalf("deployment status: got %q want draft", deployment.Status)
	}
	if len(deployment.Servers) != 1 || deployment.Servers[0] != server.ID {
		t.Fatalf("deployment servers mismatch: %#v", deployment.Servers)
	}
}

func TestCreateRemoteTask(t *testing.T) {
	handler := newTestHandler(t)

	project := mustRequest[domain.Project](t, handler, http.MethodPost, "/api/projects", map[string]any{
		"name": "task-demo",
	})
	serverOne := mustRequest[domain.ServerConfig](t, handler, http.MethodPost, "/api/servers", map[string]any{
		"projectId": project.ID,
		"name":      "robot-01",
		"host":      "mock-robot-01",
		"sshPort":   22,
		"username":  "root",
	})
	serverTwo := mustRequest[domain.ServerConfig](t, handler, http.MethodPost, "/api/servers", map[string]any{
		"projectId": project.ID,
		"name":      "robot-02",
		"host":      "mock-robot-02",
		"sshPort":   22,
		"username":  "root",
	})

	task := mustRequest[domain.RemoteTask](t, handler, http.MethodPost, "/api/remote-tasks", map[string]any{
		"name":          "install exporter",
		"projectId":     project.ID,
		"scope":         "all",
		"executionType": "preset",
		"presetId":      "docker_install_npu_exporter",
		"presetArgs": map[string]any{
			"image": "swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v2.0.1",
		},
	})
	if task.ID == "" {
		t.Fatal("remote task id was not generated")
	}
	if len(task.Runs) != 2 {
		t.Fatalf("expected 2 pending runs, got %d", len(task.Runs))
	}

	final := waitForRemoteTask(t, handler, task.ID)
	if final.Status != "completed" {
		t.Fatalf("remote task status: got %q want completed", final.Status)
	}
	if len(final.Runs) != 2 {
		t.Fatalf("expected 2 final runs, got %d", len(final.Runs))
	}

	serverIDs := map[string]bool{serverOne.ID: true, serverTwo.ID: true}
	for _, run := range final.Runs {
		if !serverIDs[run.ServerID] {
			t.Fatalf("unexpected server run: %#v", run)
		}
		if run.Status != "completed" {
			t.Fatalf("run status: got %q want completed", run.Status)
		}
		if !strings.Contains(run.Output, "mock executed") {
			t.Fatalf("unexpected run output: %q", run.Output)
		}
		if run.Command == "" {
			t.Fatalf("expected recorded command for run %#v", run)
		}
	}
}

func TestListRemoteTasksEmpty(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/remote-tasks", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/remote-tasks returned %d: %s", rec.Code, rec.Body.String())
	}

	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Fatalf("expected empty array body, got %q", body)
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	previousFake := os.Getenv("MODELRUN_FAKE_CONNECT")
	if err := os.Setenv("MODELRUN_FAKE_CONNECT", "1"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if previousFake == "" {
			_ = os.Unsetenv("MODELRUN_FAKE_CONNECT")
			return
		}
		_ = os.Setenv("MODELRUN_FAKE_CONNECT", previousFake)
	})

	st, err := store.New(filepath.Join(t.TempDir(), "modelrun.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := st.Close(); err != nil {
			t.Fatal(err)
		}
	})
	hub := realtime.NewHub()
	executor := deploy.NewExecutor(st, hub)

	return New(st, executor, hub, "").Routes()
}

func mustRequest[T any](t *testing.T, handler http.Handler, method, path string, body any) T {
	t.Helper()

	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("%s %s returned %d: %s", method, path, rec.Code, rec.Body.String())
	}

	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func waitForRemoteTask(t *testing.T, handler http.Handler, taskID string) domain.RemoteTask {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		task := mustRequest[domain.RemoteTask](t, handler, http.MethodGet, "/api/remote-tasks/"+taskID, map[string]any{})
		if task.Status == "completed" || task.Status == "failed" || task.Status == "partial" {
			return task
		}
		time.Sleep(20 * time.Millisecond)
	}

	return mustRequest[domain.RemoteTask](t, handler, http.MethodGet, "/api/remote-tasks/"+taskID, map[string]any{})
}

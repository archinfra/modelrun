package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	st, err := store.New(filepath.Join(t.TempDir(), "modelrun.json"))
	if err != nil {
		t.Fatal(err)
	}
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

package deploy

import (
	"strings"
	"testing"

	"modelrun/backend/internal/domain"
)

func TestBuildPrepareModelCommandUsesOptionalSudoForManagedPaths(t *testing.T) {
	deployment := domain.DeploymentConfig{
		Model: domain.ModelConfig{
			Source:   "modelscope",
			ModelID:  "Qwen/Qwen3.5-32B",
			Revision: "main",
		},
	}

	command, err := buildPrepareModelCommand(
		deployment,
		"/opt/modelrun/deployments/deployment_test",
		"/opt/modelrun/models/deployment_test",
	)
	if err != nil {
		t.Fatalf("buildPrepareModelCommand returned error: %v", err)
	}

	for _, needle := range []string{
		"run_with_optional_sudo",
		"can_write_target",
		"sudo -n sh -lc",
		"modelscope download --model",
		"Qwen/Qwen3.5-32B",
		"/opt/modelrun/models/deployment_test",
	} {
		if !strings.Contains(command, needle) {
			t.Fatalf("expected command to contain %q, got %q", needle, command)
		}
	}
}

func TestBuildLaunchRuntimeCommandUsesOptionalSudoForWritableDirs(t *testing.T) {
	template, ok := LookupTemplate("vllm-ascend")
	if !ok {
		t.Fatal("expected vllm-ascend template")
	}

	deployment := domain.DeploymentConfig{
		ID:        "deployment_test",
		Name:      "demo",
		Framework: "vllm-ascend",
		Model: domain.ModelConfig{
			Source:   "modelscope",
			ModelID:  "Qwen/Qwen3.5-32B",
			Revision: "main",
		},
		Docker:  template.DefaultDocker,
		Runtime: template.DefaultRuntime,
		VLLM:    template.DefaultVLLM,
		APIPort: 8000,
	}

	command, err := buildLaunchRuntimeCommand(
		template,
		deployment,
		template.DefaultDocker,
		template.DefaultRuntime,
		domain.ServerConfig{ID: "server-1", Host: "10.0.0.1"},
		[]domain.ServerConfig{{ID: "server-1", Host: "10.0.0.1"}},
		"/data/models/deployment_test",
		"/data/deployments/deployment_test",
		"/data/cache/deployment_test",
	)
	if err != nil {
		t.Fatalf("buildLaunchRuntimeCommand returned error: %v", err)
	}

	for _, needle := range []string{
		"run_with_optional_sudo",
		"/data/deployments/deployment_test",
		"/data/cache/deployment_test",
		"run_docker run -d --name",
	} {
		if !strings.Contains(command, needle) {
			t.Fatalf("expected command to contain %q, got %q", needle, command)
		}
	}
}

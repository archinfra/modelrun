package deploy

import (
	"strings"
	"testing"

	"modelrun/backend/internal/domain"
)

func TestBuildPrepareModelCommandUsesOptionalSudoForManagedPaths(t *testing.T) {
	deployment := domain.DeploymentConfig{
		Model: domain.ModelConfig{
			Source:  "modelscope",
			ModelID: "Qwen/Qwen3.5-32B",
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
	if strings.Contains(command, "--revision") {
		t.Fatalf("expected modelscope download command to omit implicit revision, got %q", command)
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

func TestBuildVLLMAscendLaunchScriptUsesRayHeadAndWorkerRoles(t *testing.T) {
	deployment := domain.DeploymentConfig{
		ID:        "deployment_test",
		Name:      "demo",
		Framework: "vllm-ascend",
		Ray: domain.DeploymentRayConfig{
			Enabled:       true,
			HeadServerID:  "server-head",
			NICName:       "eth0",
			Port:          6380,
			DashboardPort: 8266,
		},
		VLLM: domain.VLLMParams{
			TensorParallelSize:   8,
			PipelineParallelSize: 2,
			MaxModelLen:          32768,
			GPUMemoryUtilization: 0.92,
			Dtype:                "bfloat16",
			TrustRemoteCode:      true,
			EnablePrefixCaching:  true,
			EnableExpertParallel: true,
			MaxNumSeqs:           64,
			MaxNumBatchedTokens:  16384,
		},
		ServerOverrides: []domain.DeploymentServerOverride{
			{ServerID: "server-head", NodeIP: "10.0.0.11", VisibleDevices: "0,1,2,3,4,5,6,7"},
			{ServerID: "server-worker", NodeIP: "10.0.0.12", VisibleDevices: "0,1,2,3,4,5,6,7", RayStartArgs: []string{"--resources", "{\"worker\": 1}"}},
		},
		APIPort: 8000,
		Runtime: domain.DeploymentRuntimeConfig{},
	}
	servers := []domain.ServerConfig{
		{ID: "server-head", Host: "10.0.0.11"},
		{ID: "server-worker", Host: "10.0.0.12"},
	}

	headScript := buildVLLMAscendLaunchScript(deployment, servers[0], servers)
	for _, needle := range []string{
		"'ray' 'start' '--head' '--port' '6380' '--dashboard-host' '0.0.0.0' '--dashboard-port' '8266' '--node-ip-address' '10.0.0.11'",
		"export HCCL_IF_IP='10.0.0.11'",
		"export HCCL_SOCKET_IFNAME='eth0'",
		"'--distributed-executor-backend' 'ray'",
		"--enable-expert-parallel",
		"export RAY_ADDRESS=auto",
	} {
		if !strings.Contains(headScript, needle) {
			t.Fatalf("expected head script to contain %q, got %q", needle, headScript)
		}
	}

	workerScript := buildVLLMAscendLaunchScript(deployment, servers[1], servers)
	for _, needle := range []string{
		"'ray' 'start' '--address' '10.0.0.11:6380' '--node-ip-address' '10.0.0.12' '--resources' '{\"worker\": 1}'",
		"export HCCL_IF_IP='10.0.0.12'",
		"exec tail -f /dev/null",
	} {
		if !strings.Contains(workerScript, needle) {
			t.Fatalf("expected worker script to contain %q, got %q", needle, workerScript)
		}
	}
	if strings.Contains(workerScript, "vllm serve /model") {
		t.Fatalf("expected worker script to stay idle after joining ray, got %q", workerScript)
	}
}

func TestBuildVerifyCommandUsesRayStatusForWorkerNode(t *testing.T) {
	deployment := domain.DeploymentConfig{
		ID:        "deployment_test",
		Name:      "demo",
		Framework: "vllm-ascend",
		Ray: domain.DeploymentRayConfig{
			Enabled:      true,
			HeadServerID: "server-head",
		},
		APIPort: 8000,
	}
	runtime := domain.DeploymentRuntimeConfig{ContainerName: "modelrun-demo"}
	servers := []domain.ServerConfig{
		{ID: "server-head", Host: "10.0.0.11"},
		{ID: "server-worker", Host: "10.0.0.12"},
	}

	command := buildVerifyCommand(deployment, runtime, servers[1], servers)
	for _, needle := range []string{
		"run_docker inspect 'modelrun-demo' >/dev/null 2>&1",
		"run_docker exec 'modelrun-demo' bash -lc 'ray status >/dev/null'",
		"run_docker logs --tail 80 'modelrun-demo' >&2 || true",
	} {
		if !strings.Contains(command, needle) {
			t.Fatalf("expected worker verify command to contain %q, got %q", needle, command)
		}
	}
}

func TestBuildVLLMRayCompatiblePreviewUsesRayScriptStyle(t *testing.T) {
	deployment := domain.DeploymentConfig{
		ID:        "deployment_test",
		Name:      "demo",
		Framework: "vllm-ascend",
		Model: domain.ModelConfig{
			Source:  "modelscope",
			ModelID: "Qwen/Qwen3.5-397B-A17B",
		},
		Ray: domain.DeploymentRayConfig{
			Enabled:      true,
			HeadServerID: "server-head",
			NICName:      "bond1.117",
		},
		VLLM: domain.VLLMParams{
			TensorParallelSize:   8,
			PipelineParallelSize: 2,
			MaxModelLen:          32768,
		},
		Runtime: domain.DeploymentRuntimeConfig{
			ExtraArgs: []string{"--trust-remote-code", "--max-num-seqs", "256"},
		},
		ServerOverrides: []domain.DeploymentServerOverride{
			{ServerID: "server-head", NodeIP: "172.20.14.20", VisibleDevices: "0,1,2,3,4,5,6,7"},
			{ServerID: "server-worker", NodeIP: "172.20.14.21", VisibleDevices: "0,1,2,3,4,5,6,7", RayStartArgs: []string{"--resources", "{\"worker\": 1}"}},
		},
	}
	servers := []domain.ServerConfig{
		{ID: "server-head", Host: "172.20.14.20"},
		{ID: "server-worker", Host: "172.20.14.21"},
	}

	headPreview := buildVLLMRayCompatiblePreview(deployment, servers[0], servers)
	for _, needle := range []string{
		"./ray.sh start \\",
		"--node-role head \\",
		"--head-ip 172.20.14.20 \\",
		"--node-ip 172.20.14.20 \\",
		"--nic-name bond1.117 \\",
		"--cards 0,1,2,3,4,5,6,7 \\",
		"--tp 8 \\",
		"--pp 2 \\",
		"--vllm-args '--trust-remote-code --max-num-seqs 256'",
	} {
		if !strings.Contains(headPreview, needle) {
			t.Fatalf("expected head preview to contain %q, got %q", needle, headPreview)
		}
	}

	workerPreview := buildVLLMRayCompatiblePreview(deployment, servers[1], servers)
	for _, needle := range []string{
		"--node-role worker \\",
		"--head-ip 172.20.14.20 \\",
		"--node-ip 172.20.14.21 \\",
		"# 节点附加 Ray 参数: --resources {\"worker\": 1}",
	} {
		if !strings.Contains(workerPreview, needle) {
			t.Fatalf("expected worker preview to contain %q, got %q", needle, workerPreview)
		}
	}
}

func TestBuildVerifyCommandUsesContainerDiagnosticsForHTTPChecks(t *testing.T) {
	deployment := domain.DeploymentConfig{
		ID:        "deployment_test",
		Name:      "demo",
		Framework: "tei",
		APIPort:   8080,
	}
	runtime := domain.DeploymentRuntimeConfig{ContainerName: "modelrun-demo"}
	server := domain.ServerConfig{ID: "server-1", Host: "10.0.0.11"}

	command := buildVerifyCommand(deployment, runtime, server, []domain.ServerConfig{server})
	for _, needle := range []string{
		"run_docker inspect 'modelrun-demo' >/dev/null 2>&1",
		"curl -fsS --max-time 10 'http://127.0.0.1:8080/docs' >/dev/null && exit 0;",
		"run_docker logs --tail 80 'modelrun-demo' >&2 || true",
	} {
		if !strings.Contains(command, needle) {
			t.Fatalf("expected http verify command to contain %q, got %q", needle, command)
		}
	}
}

func TestDeploymentModelHostPathUsesNormalizedModelID(t *testing.T) {
	deployment := domain.DeploymentConfig{
		Model: domain.ModelConfig{
			Source:  "modelscope",
			ModelID: "Qwen/Qwen3.5-397B-A17B",
		},
	}
	runtime := domain.DeploymentRuntimeConfig{ModelDir: "/data/models"}

	got := deploymentModelHostPath(deployment, runtime)
	if got != "/data/models/qwen/qwen3.5-397b-a17b" {
		t.Fatalf("deploymentModelHostPath() = %q, want %q", got, "/data/models/qwen/qwen3.5-397b-a17b")
	}
}

func TestOptionalRevisionArg(t *testing.T) {
	if got := optionalRevisionArg(""); got != "" {
		t.Fatalf("optionalRevisionArg(\"\") = %q, want empty", got)
	}
	if got := optionalRevisionArg("main"); got != "" {
		t.Fatalf("optionalRevisionArg(\"main\") = %q, want empty", got)
	}
	if got := optionalRevisionArg("master"); got != " --revision 'master'" {
		t.Fatalf("optionalRevisionArg(\"master\") = %q", got)
	}
}

func TestWithPathPrivilegesOnlyPrintsHintWhenSudoCheckFails(t *testing.T) {
	command := withPathPrivileges("echo hello", []string{"/opt/modelrun"}, "permission hint")
	if !strings.Contains(command, "sudo -n true >/dev/null 2>&1 || { echo 'permission hint' >&2; return 1; };") {
		t.Fatalf("expected sudo availability check in command, got %q", command)
	}
	if strings.Contains(command, "status=$?; if [ $status -ne 0 ]; then") {
		t.Fatalf("expected command to stop appending generic hint to all sudo failures, got %q", command)
	}
}

func TestBuildConfiguredStepUsesCustomStepTemplate(t *testing.T) {
	template, ok := LookupTemplate("tei")
	if !ok {
		t.Fatal("expected tei template")
	}

	step := buildConfiguredStep(template, []domain.PipelineStepTemplate{
		{
			Framework:       "tei",
			StepID:          "pull_image",
			Name:            "自定义拉取镜像",
			Description:     "先输出再拉镜像",
			CommandTemplate: "echo before && {{pullImageCommand}}",
			PreviewTemplate: "preview {{imageRef}}",
		},
	}, "pull_image", buildPipelineRenderValues(
		domain.DeploymentConfig{ID: "deployment-1", Name: "demo", Framework: "tei", APIPort: 8080},
		domain.ServerConfig{ID: "server-1", Name: "node-a", Host: "10.0.0.1"},
		domain.DeploymentRuntimeConfig{ContainerName: "demo-container"},
		"/data/models/demo",
		"/data/work/demo",
		"/data/cache/demo",
		"ghcr.io/example/demo:latest",
		stepCommandSet{
			PullImageCommand: "docker pull 'ghcr.io/example/demo:latest'",
			PullImagePreview: "docker pull 'ghcr.io/example/demo:latest'",
		},
	))

	if step.step.Name != "自定义拉取镜像" {
		t.Fatalf("step name = %q", step.step.Name)
	}
	if !strings.Contains(step.command, "echo before && docker pull 'ghcr.io/example/demo:latest'") {
		t.Fatalf("step command = %q", step.command)
	}
	if step.step.CommandPreview != "preview ghcr.io/example/demo:latest" {
		t.Fatalf("step preview = %q", step.step.CommandPreview)
	}
}

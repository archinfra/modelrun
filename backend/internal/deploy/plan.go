package deploy

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"modelrun/backend/internal/domain"
)

type plannedStep struct {
	step    domain.DeploymentStep
	command string
}

func buildPlan(deployment domain.DeploymentConfig, server domain.ServerConfig, servers []domain.ServerConfig) ([]plannedStep, error) {
	template, ok := LookupTemplate(deployment.Framework)
	if !ok {
		return nil, fmt.Errorf("unsupported framework %q", deployment.Framework)
	}

	runtime := mergedRuntimeConfig(template, deployment.Runtime)
	docker := mergedDockerConfig(template, deployment.Docker)
	modelHostPath := deploymentModelHostPath(deployment, runtime)
	workDir := deploymentWorkDir(runtime, deployment)
	cacheDir := deploymentCacheDir(runtime, deployment)

	prepareCommand, err := buildPrepareModelCommand(deployment, workDir, modelHostPath)
	if err != nil {
		return nil, err
	}
	imageRef := dockerImageRef(docker)
	launchCommand, err := buildLaunchRuntimeCommand(template, deployment, docker, runtime, server, servers, modelHostPath, workDir, cacheDir)
	if err != nil {
		return nil, err
	}

	steps := []plannedStep{
		{
			step: domain.DeploymentStep{
				ID:             "prepare_model",
				Name:           "准备模型",
				Description:    "下载模型到托管目录，或校验目标服务器上的本地模型目录。",
				CommandPreview: prepareCommand,
				Status:         "pending",
				Logs:           []string{},
			},
			command: prepareCommand,
		},
		{
			step: domain.DeploymentStep{
				ID:             "pull_image",
				Name:           "拉取镜像",
				Description:    "在目标服务器上拉取当前配置的运行时镜像。",
				CommandPreview: withDockerPrivileges("run_docker pull " + shellQuote(imageRef)),
				Status:         "pending",
				Logs:           []string{},
			},
			command: withDockerPrivileges("run_docker pull " + shellQuote(imageRef)),
		},
		{
			step: domain.DeploymentStep{
				ID:             "launch_runtime",
				Name:           "启动服务",
				Description:    templateStepDescription(template, "launch_runtime"),
				CommandPreview: launchCommand,
				AutoManaged:    true,
				Status:         "pending",
				Logs:           []string{},
			},
			command: launchCommand,
		},
		{
			step: domain.DeploymentStep{
				ID:             "verify_service",
				Name:           "验证服务",
				Description:    templateStepDescription(template, "verify_service"),
				CommandPreview: buildVerifyCommand(deployment, runtime, server, servers),
				AutoManaged:    true,
				Status:         "pending",
				Logs:           []string{},
			},
			command: buildVerifyCommand(deployment, runtime, server, servers),
		},
	}

	if template.SupportsRay {
		for i := range steps {
			if steps[i].step.ID == "launch_runtime" && deployment.Ray.Enabled {
				steps[i].step.Optional = false
				continue
			}
			if steps[i].step.ID == "launch_runtime" {
				steps[i].step.Optional = true
			}
		}
	}

	return steps, nil
}

func stepsFromPlan(plan []plannedStep) []domain.DeploymentStep {
	steps := make([]domain.DeploymentStep, 0, len(plan))
	for _, item := range plan {
		steps = append(steps, item.step)
	}
	return steps
}

func mergedDockerConfig(template domain.PipelineTemplate, current domain.DockerConfig) domain.DockerConfig {
	out := template.DefaultDocker
	if current.Image != "" {
		out.Image = current.Image
	}
	if current.Tag != "" {
		out.Tag = current.Tag
	}
	if current.Registry != "" {
		out.Registry = current.Registry
	}
	if current.GPUDevices != "" {
		out.GPUDevices = current.GPUDevices
	}
	if current.ShmSize != "" {
		out.ShmSize = current.ShmSize
	}
	if current.Network != "" {
		out.Network = current.Network
	}
	if current.IPC != "" {
		out.IPC = current.IPC
	}
	if current.Runtime != "" {
		out.Runtime = current.Runtime
	}
	out.Privileged = out.Privileged || current.Privileged
	if current.EnvironmentVars != nil {
		if out.EnvironmentVars == nil {
			out.EnvironmentVars = map[string]string{}
		}
		for key, value := range current.EnvironmentVars {
			out.EnvironmentVars[key] = value
		}
	}
	if len(current.Volumes) > 0 {
		out.Volumes = append([]domain.VolumeMount{}, current.Volumes...)
	}
	return out
}

func mergedRuntimeConfig(template domain.PipelineTemplate, current domain.DeploymentRuntimeConfig) domain.DeploymentRuntimeConfig {
	out := template.DefaultRuntime
	if current.ContainerName != "" {
		out.ContainerName = current.ContainerName
	}
	if current.WorkDir != "" {
		out.WorkDir = current.WorkDir
	}
	if current.ModelDir != "" {
		out.ModelDir = current.ModelDir
	}
	if current.CacheDir != "" {
		out.CacheDir = current.CacheDir
	}
	if current.SharedCacheDir != "" {
		out.SharedCacheDir = current.SharedCacheDir
	}
	if len(current.ExtraArgs) > 0 {
		out.ExtraArgs = append([]string{}, current.ExtraArgs...)
	}
	if current.EnableAutoRestart {
		out.EnableAutoRestart = true
	}
	return out
}

func deploymentContainerName(deployment domain.DeploymentConfig, runtime domain.DeploymentRuntimeConfig) string {
	if strings.TrimSpace(runtime.ContainerName) != "" {
		return strings.TrimSpace(runtime.ContainerName)
	}
	name := strings.ToLower(strings.TrimSpace(deployment.Name))
	name = strings.NewReplacer(" ", "-", "_", "-", "/", "-").Replace(name)
	name = strings.Trim(name, "-")
	if name == "" {
		name = deployment.ID
	}
	return "modelrun-" + name
}

func deploymentModelHostPath(deployment domain.DeploymentConfig, runtime domain.DeploymentRuntimeConfig) string {
	if deployment.Model.Source == "local" && strings.TrimSpace(deployment.Model.LocalPath) != "" {
		return strings.TrimSpace(deployment.Model.LocalPath)
	}
	if modelPath := remoteModelRelativePath(deployment.Model.ModelID); modelPath != "" {
		return path.Join(strings.TrimRight(runtime.ModelDir, "/"), modelPath)
	}
	return path.Join(strings.TrimRight(runtime.ModelDir, "/"), deployment.ID)
}

func deploymentWorkDir(runtime domain.DeploymentRuntimeConfig, deployment domain.DeploymentConfig) string {
	return path.Join(strings.TrimRight(runtime.WorkDir, "/"), deployment.ID)
}

func deploymentCacheDir(runtime domain.DeploymentRuntimeConfig, deployment domain.DeploymentConfig) string {
	base := runtime.CacheDir
	if strings.TrimSpace(runtime.SharedCacheDir) != "" {
		base = runtime.SharedCacheDir
	}
	return path.Join(strings.TrimRight(base, "/"), deployment.ID)
}

func buildPrepareModelCommand(deployment domain.DeploymentConfig, workDir, modelHostPath string) (string, error) {
	commands := []string{"set -e", "mkdir -p " + shellQuote(workDir)}
	privilegedPaths := []string{workDir}

	switch strings.ToLower(strings.TrimSpace(deployment.Model.Source)) {
	case "", "local":
		target := strings.TrimSpace(deployment.Model.LocalPath)
		if target == "" {
			target = modelHostPath
		}
		commands = append(commands,
			"test -e "+shellQuote(target)+" || { echo 'model path not found: "+escapeForSingleQuotedMessage(target)+"' >&2; exit 1; }",
			"echo 'using local model path "+escapeForSingleQuotedMessage(target)+"'",
		)
	case "modelscope":
		if strings.TrimSpace(deployment.Model.ModelID) == "" {
			return "", fmt.Errorf("modelId is required for modelscope source")
		}
		privilegedPaths = append(privilegedPaths, modelHostPath)
		revisionArg := optionalRevisionArg(deployment.Model.Revision)
		commands = append(commands,
			"mkdir -p "+shellQuote(modelHostPath),
			"export PATH=\"$PATH:$HOME/.local/bin\"",
			"command -v modelscope >/dev/null 2>&1 || { command -v python3 >/dev/null 2>&1 || { echo 'python3 is required to install modelscope' >&2; exit 127; }; python3 -m pip install --user modelscope; }",
			"modelscope download --model "+shellQuote(deployment.Model.ModelID)+revisionArg+" --local_dir "+shellQuote(modelHostPath),
		)
	case "huggingface":
		if strings.TrimSpace(deployment.Model.ModelID) == "" {
			return "", fmt.Errorf("modelId is required for huggingface source")
		}
		privilegedPaths = append(privilegedPaths, modelHostPath)
		revisionArg := optionalRevisionArg(deployment.Model.Revision)
		commands = append(commands,
			"mkdir -p "+shellQuote(modelHostPath),
			"export PATH=\"$PATH:$HOME/.local/bin\"",
			"command -v huggingface-cli >/dev/null 2>&1 || { command -v python3 >/dev/null 2>&1 || { echo 'python3 is required to install huggingface-cli' >&2; exit 127; }; python3 -m pip install --user 'huggingface_hub[cli]'; }",
			"huggingface-cli download "+shellQuote(deployment.Model.ModelID)+revisionArg+" --local-dir "+shellQuote(modelHostPath),
		)
	default:
		return "", fmt.Errorf("unsupported model source %q", deployment.Model.Source)
	}

	return withPathPrivileges(
		strings.Join(commands, " && "),
		privilegedPaths,
		"model preparation requires write access to the managed runtime directories. Configure a writable runtime path or allow passwordless sudo for the SSH user.",
	), nil
}

func buildLaunchRuntimeCommand(template domain.PipelineTemplate, deployment domain.DeploymentConfig, docker domain.DockerConfig, runtime domain.DeploymentRuntimeConfig, server domain.ServerConfig, servers []domain.ServerConfig, modelHostPath, workDir, cacheDir string) (string, error) {
	containerName := deploymentContainerName(deployment, runtime)
	launchScript, configJSON, err := frameworkLaunchAssets(template, deployment, server, servers)
	if err != nil {
		return "", err
	}

	scriptHostPath := path.Join(workDir, "launch.sh")
	commands := []string{
		"set -e",
		"mkdir -p " + shellQuote(workDir),
		"mkdir -p " + shellQuote(cacheDir),
		"cat > " + shellQuote(scriptHostPath) + " <<'EOF'\n" + launchScript + "\nEOF",
		"chmod +x " + shellQuote(scriptHostPath),
	}
	if strings.TrimSpace(configJSON) != "" {
		configHostPath := path.Join(workDir, "config.json")
		commands = append(commands, "cat > "+shellQuote(configHostPath)+" <<'EOF'\n"+configJSON+"\nEOF")
	}

	dockerCommand := buildDockerRunCommand(template, deployment, docker, runtime, containerName, modelHostPath, workDir, cacheDir)
	commands = append(commands, dockerCommand)
	return withPathPrivileges(
		strings.Join(commands, " && "),
		[]string{workDir, cacheDir},
		"runtime launch needs write access to the deployment work or cache directory. Configure a writable runtime path or allow passwordless sudo for the SSH user.",
	), nil
}

func frameworkLaunchAssets(template domain.PipelineTemplate, deployment domain.DeploymentConfig, server domain.ServerConfig, servers []domain.ServerConfig) (string, string, error) {
	switch template.Framework {
	case "tei":
		return buildTEILaunchScript(deployment), "", nil
	case "vllm-ascend":
		return buildVLLMAscendLaunchScript(deployment, server, servers), "", nil
	case "mindie":
		configJSON, err := buildMindIEConfigJSON(deployment, server, servers)
		if err != nil {
			return "", "", err
		}
		return buildMindIELaunchScript(deployment), configJSON, nil
	default:
		return "", "", fmt.Errorf("unsupported framework %q", template.Framework)
	}
}

func buildTEILaunchScript(deployment domain.DeploymentConfig) string {
	args := append([]string{
		"text-embeddings-router",
		"--model-id", "/model",
		"--hostname", "0.0.0.0",
		"--port", strconv.Itoa(deployment.APIPort),
	}, deployment.Runtime.ExtraArgs...)
	return strings.Join([]string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"exec " + joinShellArgs(args...),
	}, "\n")
}

func buildVLLMAscendLaunchScript(deployment domain.DeploymentConfig, server domain.ServerConfig, servers []domain.ServerConfig) string {
	override := serverOverrideFor(deployment, server.ID)
	nodeIP := effectiveRayNodeIP(server, override)
	head := pickRayHeadServer(deployment, servers)
	headNodeIP := effectiveRayNodeIP(head, serverOverrideFor(deployment, head.ID))

	lines := []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"export HF_HOME=/opt/modelrun/cache",
		"export HUGGINGFACE_HUB_CACHE=/opt/modelrun/cache",
		"export PYTHONUNBUFFERED=1",
	}
	if value := effectiveVisibleDevices(deployment, override); value != "" {
		lines = append(lines, "export ASCEND_RT_VISIBLE_DEVICES="+shellQuote(value))
	}
	if nodeIP != "" {
		lines = append(lines, "export HCCL_IF_IP="+shellQuote(nodeIP))
	}
	if value := strings.TrimSpace(deployment.Ray.NICName); value != "" {
		lines = append(lines,
			"export HCCL_SOCKET_IFNAME="+shellQuote(value),
			"export GLOO_SOCKET_IFNAME="+shellQuote(value),
			"export TP_SOCKET_IFNAME="+shellQuote(value),
		)
	}
	if deployment.Ray.Enabled {
		lines = append(lines, "ray stop >/dev/null 2>&1 || true")
		if head.ID == server.ID {
			rayArgs := []string{
				"ray", "start",
				"--head",
				"--port", strconv.Itoa(defaultRayPort(deployment.Ray.Port)),
				"--dashboard-host", "0.0.0.0",
				"--dashboard-port", strconv.Itoa(defaultDashboardPort(deployment.Ray.DashboardPort)),
			}
			if nodeIP != "" {
				rayArgs = append(rayArgs, "--node-ip-address", nodeIP)
			}
			if len(override.RayStartArgs) > 0 {
				rayArgs = append(rayArgs, override.RayStartArgs...)
			}
			lines = append(lines,
				joinShellArgs(rayArgs...),
				"export RAY_EXPERIMENTAL_NOSET_ASCEND_RT_VISIBLE_DEVICES=1",
				"export RAY_ADDRESS=auto",
			)
		} else {
			rayArgs := []string{
				"ray", "start",
				"--address", fmt.Sprintf("%s:%d", firstNonEmpty(headNodeIP, "127.0.0.1"), defaultRayPort(deployment.Ray.Port)),
			}
			if nodeIP != "" {
				rayArgs = append(rayArgs, "--node-ip-address", nodeIP)
			}
			if len(override.RayStartArgs) > 0 {
				rayArgs = append(rayArgs, override.RayStartArgs...)
			}
			lines = append(lines,
				joinShellArgs(rayArgs...),
				"export RAY_EXPERIMENTAL_NOSET_ASCEND_RT_VISIBLE_DEVICES=1",
				"exec tail -f /dev/null",
			)
			return strings.Join(lines, "\n")
		}
	}

	args := []string{
		"vllm", "serve", "/model",
		"--host", "0.0.0.0",
		"--port", strconv.Itoa(deployment.APIPort),
		"--tensor-parallel-size", strconv.Itoa(maxInt(1, deployment.VLLM.TensorParallelSize)),
		"--pipeline-parallel-size", strconv.Itoa(maxInt(1, deployment.VLLM.PipelineParallelSize)),
		"--max-model-len", strconv.Itoa(maxInt(1, deployment.VLLM.MaxModelLen)),
		"--gpu-memory-utilization", trimFloat(deployment.VLLM.GPUMemoryUtilization, "0.90"),
		"--dtype", firstNonEmpty(deployment.VLLM.Dtype, "auto"),
		"--max-num-seqs", strconv.Itoa(maxInt(1, deployment.VLLM.MaxNumSeqs)),
		"--max-num-batched-tokens", strconv.Itoa(maxInt(1, deployment.VLLM.MaxNumBatchedTokens)),
	}
	if deployment.Ray.Enabled {
		args = append(args, "--distributed-executor-backend", "ray")
	}
	if deployment.VLLM.TrustRemoteCode {
		args = append(args, "--trust-remote-code")
	}
	if deployment.VLLM.EnablePrefixCaching {
		args = append(args, "--enable-prefix-caching")
	}
	if deployment.VLLM.EnableExpertParallel {
		args = append(args, "--enable-expert-parallel")
	}
	if deployment.VLLM.Quantization != "" {
		args = append(args, "--quantization", deployment.VLLM.Quantization)
	}
	if deployment.VLLM.SwapSpace > 0 {
		args = append(args, "--swap-space", strconv.Itoa(deployment.VLLM.SwapSpace))
	}
	if deployment.VLLM.EnforceEager {
		args = append(args, "--enforce-eager")
	}
	if deployment.VLLM.EnableChunkedPrefill {
		args = append(args, "--enable-chunked-prefill")
	}
	if deployment.VLLM.SpeculativeModel != "" {
		args = append(args, "--speculative-model", deployment.VLLM.SpeculativeModel)
	}
	if deployment.VLLM.NumSpeculativeTokens > 0 {
		args = append(args, "--num-speculative-tokens", strconv.Itoa(deployment.VLLM.NumSpeculativeTokens))
	}
	if len(deployment.Runtime.ExtraArgs) > 0 {
		args = append(args, deployment.Runtime.ExtraArgs...)
	}
	lines = append(lines, "exec "+joinShellArgs(args...))
	return strings.Join(lines, "\n")
}

func buildMindIELaunchScript(deployment domain.DeploymentConfig) string {
	return strings.Join([]string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"[ -f /usr/local/Ascend/ascend-toolkit/set_env.sh ] && . /usr/local/Ascend/ascend-toolkit/set_env.sh || true",
		"[ -f /usr/local/Ascend/nnal/atb/set_env.sh ] && . /usr/local/Ascend/nnal/atb/set_env.sh || true",
		"[ -f /usr/local/Ascend/mindie/latest/mindie-service/set_env.sh ] && . /usr/local/Ascend/mindie/latest/mindie-service/set_env.sh || true",
		"cp /opt/modelrun/runtime/config.json /usr/local/Ascend/mindie/latest/mindie-service/conf/config.json",
		"cd /usr/local/Ascend/mindie/latest/mindie-service",
		"exec ./bin/mindieservice_daemon",
	}, "\n")
}

func buildMindIEConfigJSON(deployment domain.DeploymentConfig, server domain.ServerConfig, servers []domain.ServerConfig) (string, error) {
	serverConfig := map[string]any{
		"ipAddress":               firstNonEmpty(server.Host, "127.0.0.1"),
		"managementIpAddress":     firstNonEmpty(server.Host, "127.0.0.1"),
		"port":                    deployment.APIPort,
		"managementPort":          deployment.APIPort + 1,
		"metricsPort":             deployment.APIPort + 2,
		"allowAllZeroIpListening": false,
		"maxLinkNum":              1000,
		"httpsEnabled":            false,
		"fullTextEnabled":         false,
		"openAiSupport":           "vllm",
	}
	modelEntry := map[string]any{
		"modelName":       sanitizeModelName(deployment.Name),
		"modelWeightPath": "/model",
		"backendType":     "atb",
		"worldSize":       maxInt(1, len(servers)),
	}
	payload := map[string]any{
		"Version": "1.0.0",
		"ServerConfig": map[string]any{
			"ipAddress":               serverConfig["ipAddress"],
			"managementIpAddress":     serverConfig["managementIpAddress"],
			"port":                    serverConfig["port"],
			"managementPort":          serverConfig["managementPort"],
			"metricsPort":             serverConfig["metricsPort"],
			"allowAllZeroIpListening": serverConfig["allowAllZeroIpListening"],
			"maxLinkNum":              serverConfig["maxLinkNum"],
			"httpsEnabled":            serverConfig["httpsEnabled"],
			"fullTextEnabled":         serverConfig["fullTextEnabled"],
			"openAiSupport":           serverConfig["openAiSupport"],
		},
		"BackendConfig": map[string]any{
			"multiNodesInferEnabled": len(servers) > 1,
			"ModelDeployConfig": map[string]any{
				"ModelConfig": []map[string]any{modelEntry},
			},
		},
	}
	if len(deployment.Runtime.ExtraArgs) > 0 {
		payload["ExtraArgs"] = deployment.Runtime.ExtraArgs
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func buildDockerRunCommand(template domain.PipelineTemplate, deployment domain.DeploymentConfig, docker domain.DockerConfig, runtime domain.DeploymentRuntimeConfig, containerName, modelHostPath, workDir, cacheDir string) string {
	imageRef := dockerImageRef(docker)
	runParts := []string{
		"(run_docker rm -f " + shellQuote(containerName) + " >/dev/null 2>&1 || true)",
		"run_docker run -d --name " + shellQuote(containerName),
	}
	if runtime.EnableAutoRestart {
		runParts = append(runParts, "--restart unless-stopped")
	}
	runParts = append(runParts, "--network "+shellQuote(firstNonEmpty(docker.Network, "host")))
	runParts = append(runParts, "--ipc "+shellQuote(firstNonEmpty(docker.IPC, "host")))
	if docker.ShmSize != "" {
		runParts = append(runParts, "--shm-size "+shellQuote(docker.ShmSize))
	}
	if docker.Privileged {
		runParts = append(runParts, "--privileged")
	}
	if docker.Runtime != "" {
		runParts = append(runParts, "--runtime "+shellQuote(docker.Runtime))
	}
	for _, env := range sortedEnvPairs(docker.EnvironmentVars) {
		runParts = append(runParts, "-e "+shellQuote(env))
	}
	runParts = append(runParts,
		"-v "+shellQuote(workDir+":/opt/modelrun/runtime"),
		"-v "+shellQuote(cacheDir+":/opt/modelrun/cache"),
		"-v "+shellQuote(modelHostPath+":/model"),
	)
	for _, vol := range docker.Volumes {
		if strings.TrimSpace(vol.Host) == "" || strings.TrimSpace(vol.Container) == "" {
			continue
		}
		runParts = append(runParts, "-v "+shellQuote(strings.TrimSpace(vol.Host)+":"+strings.TrimSpace(vol.Container)))
	}
	runParts = append(runParts, shellQuote(imageRef), "bash -lc "+shellQuote("/opt/modelrun/runtime/launch.sh"))
	return withDockerPrivileges(strings.Join(runParts, " "))
}

func buildVerifyCommand(deployment domain.DeploymentConfig, runtime domain.DeploymentRuntimeConfig, server domain.ServerConfig, servers []domain.ServerConfig) string {
	if strings.EqualFold(strings.TrimSpace(deployment.Framework), "vllm-ascend") && deployment.Ray.Enabled {
		head := pickRayHeadServer(deployment, servers)
		if head.ID != "" && head.ID != server.ID {
			containerName := deploymentContainerName(deployment, runtime)
			return withDockerPrivileges(
				"run_docker exec " + shellQuote(containerName) + " bash -lc " + shellQuote("ray status >/dev/null"),
			)
		}
	}

	var url string
	switch strings.ToLower(strings.TrimSpace(deployment.Framework)) {
	case "tei":
		url = fmt.Sprintf("http://127.0.0.1:%d/docs", deployment.APIPort)
	default:
		url = fmt.Sprintf("http://127.0.0.1:%d/v1/models", deployment.APIPort)
	}
	return strings.Join([]string{
		"if command -v curl >/dev/null 2>&1; then",
		"curl -fsS --max-time 10 " + shellQuote(url) + " >/dev/null;",
		"elif command -v wget >/dev/null 2>&1; then",
		"wget -q -T 10 -O - " + shellQuote(url) + " >/dev/null;",
		"else",
		"echo 'curl or wget is required for runtime verification' >&2; exit 127;",
		"fi",
	}, " ")
}

func templateStepDescription(template domain.PipelineTemplate, id string) string {
	for _, step := range template.Steps {
		if step.ID == id {
			return step.Description
		}
	}
	return ""
}

func pickRayHeadServer(deployment domain.DeploymentConfig, servers []domain.ServerConfig) domain.ServerConfig {
	if deployment.Ray.HeadServerID != "" {
		for _, server := range servers {
			if server.ID == deployment.Ray.HeadServerID {
				return server
			}
		}
	}
	if len(servers) > 0 {
		return servers[0]
	}
	return domain.ServerConfig{}
}

func serverOverrideFor(deployment domain.DeploymentConfig, serverID string) domain.DeploymentServerOverride {
	for _, item := range deployment.ServerOverrides {
		if item.ServerID == serverID {
			return item
		}
	}
	return domain.DeploymentServerOverride{}
}

func effectiveRayNodeIP(server domain.ServerConfig, override domain.DeploymentServerOverride) string {
	return firstNonEmpty(strings.TrimSpace(override.NodeIP), strings.TrimSpace(server.Host))
}

func effectiveVisibleDevices(deployment domain.DeploymentConfig, override domain.DeploymentServerOverride) string {
	return firstNonEmpty(strings.TrimSpace(override.VisibleDevices), strings.TrimSpace(deployment.Ray.VisibleDevices))
}

func defaultRayPort(port int) int {
	if port == 0 {
		return 6379
	}
	return port
}

func defaultDashboardPort(port int) int {
	if port == 0 {
		return 8265
	}
	return port
}

func dockerImageRef(docker domain.DockerConfig) string {
	image := strings.TrimSpace(docker.Image)
	tag := strings.TrimSpace(docker.Tag)
	registry := strings.Trim(strings.TrimSpace(docker.Registry), "/")
	if registry != "" && !strings.Contains(image, registry+"/") {
		image = registry + "/" + strings.TrimLeft(image, "/")
	}
	if tag == "" || strings.Contains(path.Base(image), ":") {
		return image
	}
	return image + ":" + tag
}

func sortedEnvPairs(values map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}
	return out
}

func withDockerPrivileges(body string) string {
	return strings.Join([]string{
		"command -v docker >/dev/null 2>&1 || { echo 'docker is not installed' >&2; exit 127; };",
		"run_docker(){",
		"if [ \"$(id -u)\" -eq 0 ]; then",
		"docker \"$@\";",
		"return $?;",
		"fi;",
		"if command -v sudo >/dev/null 2>&1; then",
		"sudo -n true >/dev/null 2>&1 || { echo 'docker command requires sudo privileges for the current SSH user, or the user must be added to the docker group.' >&2; return 1; };",
		"sudo -n docker \"$@\";",
		"return $?;",
		"fi;",
		"echo 'docker command requires sudo privileges because the current SSH user is not root and sudo is unavailable.' >&2;",
		"return 1;",
		"};",
		body,
	}, " ")
}

func withPathPrivileges(body string, paths []string, failureHint string) string {
	candidates := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, value := range paths {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		candidates = append(candidates, shellQuote(value))
	}
	if len(candidates) == 0 {
		return body
	}

	return strings.Join([]string{
		"can_write_target(){",
		"target=\"$1\";",
		"current=\"$target\";",
		"while [ ! -e \"$current\" ] && [ \"$current\" != \"/\" ]; do current=$(dirname \"$current\"); done;",
		"[ -w \"$current\" ];",
		"};",
		"run_with_optional_sudo(){",
		"if [ \"$(id -u)\" -eq 0 ]; then",
		"sh -lc " + shellQuote(body) + ";",
		"return $?;",
		"fi;",
		"need_sudo=0;",
		"for target in " + strings.Join(candidates, " ") + "; do",
		"if ! can_write_target \"$target\"; then need_sudo=1; break; fi;",
		"done;",
		"if [ \"$need_sudo\" -eq 0 ]; then",
		"sh -lc " + shellQuote(body) + ";",
		"return $?;",
		"fi;",
		"if command -v sudo >/dev/null 2>&1; then",
		"sudo -n true >/dev/null 2>&1 || { echo " + shellQuote(failureHint) + " >&2; return 1; };",
		"sudo -n sh -lc " + shellQuote(body) + ";",
		"return $?;",
		"fi;",
		"echo " + shellQuote(failureHint) + " >&2;",
		"return 1;",
		"};",
		"run_with_optional_sudo",
	}, " ")
}

func remoteModelRelativePath(modelID string) string {
	modelID = strings.TrimSpace(strings.ReplaceAll(modelID, "\\", "/"))
	if modelID == "" {
		return ""
	}
	parts := strings.Split(modelID, "/")
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = sanitizePathSegment(part)
		if part == "" {
			continue
		}
		cleaned = append(cleaned, part)
	}
	if len(cleaned) == 0 {
		return ""
	}
	return path.Join(cleaned...)
}

func sanitizePathSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '.' || r == '_' || r == '-':
			builder.WriteRune(r)
		default:
			builder.WriteByte('-')
		}
	}
	out := strings.Trim(builder.String(), "-.")
	if out == "" {
		return "model"
	}
	return out
}

func optionalRevisionArg(revision string) string {
	revision = strings.TrimSpace(revision)
	if revision == "" || strings.EqualFold(revision, "main") {
		return ""
	}
	return " --revision " + shellQuote(revision)
}

func joinShellArgs(values ...string) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, shellQuote(value))
	}
	return strings.Join(out, " ")
}

func trimFloat(value float64, fallback string) string {
	if value == 0 {
		return fallback
	}
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if text == "" {
		return fallback
	}
	return text
}

func sanitizeModelName(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.NewReplacer(" ", "-", "_", "-", "/", "-").Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "modelrun"
	}
	return value
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func escapeForSingleQuotedMessage(value string) string {
	return strings.ReplaceAll(value, "'", "")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

package deploy

import (
	"fmt"
	"strconv"
	"strings"

	"modelrun/backend/internal/domain"
)

func buildPlanWithStepConfigs(deployment domain.DeploymentConfig, server domain.ServerConfig, servers []domain.ServerConfig, stepConfigs []domain.PipelineStepTemplate) ([]plannedStep, error) {
	template, ok := LookupTemplate(deployment.Framework)
	if !ok {
		return nil, fmt.Errorf("unsupported framework %q", deployment.Framework)
	}

	runtime := mergedRuntimeConfig(template, deployment.Runtime)
	docker := mergedDockerConfig(template, deployment.Docker)
	modelHostPath := deploymentModelHostPath(deployment, runtime)
	workDir := deploymentWorkDir(runtime, deployment)
	cacheDir := deploymentCacheDir(runtime, deployment)
	imageRef := dockerImageRef(docker)

	launchCommand, err := buildLaunchRuntimeCommand(template, deployment, docker, runtime, server, servers, modelHostPath, workDir, cacheDir)
	if err != nil {
		return nil, err
	}
	verifyCommand := buildVerifyCommand(deployment, runtime, server, servers)
	checkCommand, err := buildCheckModelTargetCommand(deployment, modelHostPath)
	if err != nil {
		return nil, err
	}
	fetcherCommand, err := buildPrepareModelFetcherCommand(deployment)
	if err != nil {
		return nil, err
	}
	syncCommand, err := buildSyncModelCommand(deployment, runtime, modelHostPath)
	if err != nil {
		return nil, err
	}

	renderValues := buildPipelineRenderValues(
		deployment,
		server,
		runtime,
		modelHostPath,
		workDir,
		cacheDir,
		imageRef,
		stepCommandSet{
			CheckModelTargetCommand:    checkCommand,
			CheckModelTargetPreview:    buildCheckModelTargetPreview(deployment, modelHostPath),
			PrepareModelFetcherCommand: fetcherCommand,
			PrepareModelFetcherPreview: buildPrepareModelFetcherPreview(deployment),
			SyncModelCommand:           syncCommand,
			SyncModelPreview:           buildSyncModelPreview(deployment, modelHostPath),
			PullImageCommand:           withDockerPrivileges("run_docker pull " + shellQuote(imageRef)),
			PullImagePreview:           "docker pull " + shellQuote(imageRef),
			LaunchRuntimeCommand:       launchCommand,
			LaunchRuntimePreview:       buildLaunchRuntimePreview(template, deployment, runtime, server, servers),
			VerifyServiceCommand:       verifyCommand,
			VerifyServicePreview:       verifyCommand,
		},
	)

	steps := []plannedStep{
		buildConfiguredStep(template, stepConfigs, "check_model_target", renderValues),
		buildConfiguredStep(template, stepConfigs, "prepare_model_fetcher", renderValues),
		buildConfiguredStep(template, stepConfigs, "sync_model", renderValues),
		buildConfiguredStep(template, stepConfigs, "pull_image", renderValues),
		buildConfiguredStep(template, stepConfigs, "launch_runtime", renderValues),
		buildConfiguredStep(template, stepConfigs, "verify_service", renderValues),
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

func buildLaunchRuntimePreview(template domain.PipelineTemplate, deployment domain.DeploymentConfig, runtime domain.DeploymentRuntimeConfig, server domain.ServerConfig, servers []domain.ServerConfig) string {
	if template.Framework == "vllm-ascend" {
		return buildVLLMRayCompatiblePreview(deployment, server, servers)
	}
	return buildVerifyFriendlyLaunchPreview(template, deployment, runtime)
}

func buildVerifyFriendlyLaunchPreview(template domain.PipelineTemplate, deployment domain.DeploymentConfig, runtime domain.DeploymentRuntimeConfig) string {
	containerName := deploymentContainerName(deployment, runtime)
	imageRef := dockerImageRef(mergedDockerConfig(template, deployment.Docker))
	lines := []string{
		"角色: Runtime Launcher",
		"容器: " + containerName,
		"镜像: " + imageRef,
	}
	switch template.Framework {
	case "tei":
		lines = append(lines,
			"入口: text-embeddings-router --model-id /model",
			fmt.Sprintf("端口: %d", deployment.APIPort),
		)
	case "mindie":
		lines = append(lines,
			"入口: mindieservice_daemon",
			fmt.Sprintf("端口: %d", deployment.APIPort),
		)
	}
	return strings.Join(lines, "\n")
}

func buildVLLMRayCompatiblePreview(deployment domain.DeploymentConfig, server domain.ServerConfig, servers []domain.ServerConfig) string {
	override := serverOverrideFor(deployment, server.ID)
	head := pickRayHeadServer(deployment, servers)
	headOverride := serverOverrideFor(deployment, head.ID)
	nodeIP := firstNonEmpty(strings.TrimSpace(override.NodeIP), strings.TrimSpace(server.Host), "127.0.0.1")
	headIP := firstNonEmpty(strings.TrimSpace(headOverride.NodeIP), strings.TrimSpace(head.Host), "127.0.0.1")
	modelID := firstNonEmpty(strings.TrimSpace(deployment.Model.ModelID), strings.TrimSpace(deployment.Model.Name), deployment.ID)
	segments := []string{
		"start",
		"-m " + previewShellValue(modelID),
		"--distributed-executor-backend " + map[bool]string{true: "ray", false: "mp"}[deployment.Ray.Enabled],
	}
	if deployment.Ray.Enabled {
		role := "worker"
		if head.ID == server.ID {
			role = "head"
		}
		segments = append(segments,
			"--node-role "+role,
			"--num-nodes "+strconv.Itoa(maxInt(1, len(servers))),
			"--head-ip "+previewShellValue(headIP),
			"--node-ip "+previewShellValue(nodeIP),
		)
		if nic := strings.TrimSpace(deployment.Ray.NICName); nic != "" {
			segments = append(segments, "--nic-name "+previewShellValue(nic))
		}
	}
	if cards := effectiveVisibleDevices(deployment, override); cards != "" {
		segments = append(segments, "--cards "+previewShellValue(cards))
	}
	segments = append(segments,
		"--tp "+strconv.Itoa(maxInt(1, deployment.VLLM.TensorParallelSize)),
		"--pp "+strconv.Itoa(maxInt(1, deployment.VLLM.PipelineParallelSize)),
	)
	if deployment.VLLM.MaxModelLen > 0 {
		segments = append(segments, "--max-model-len "+strconv.Itoa(deployment.VLLM.MaxModelLen))
	}
	if value := strings.TrimSpace(deployment.VLLM.Dtype); value != "" && value != "auto" {
		segments = append(segments, "--dtype "+previewShellValue(value))
	}
	if deployment.VLLM.GPUMemoryUtilization > 0 {
		segments = append(segments, "--gpu-memory-utilization "+trimFloat(deployment.VLLM.GPUMemoryUtilization, "0.90"))
	}
	if len(deployment.Runtime.ExtraArgs) > 0 {
		segments = append(segments, "--vllm-args "+shellQuote(strings.Join(deployment.Runtime.ExtraArgs, " ")))
	}
	lines := []string{"兼容启动命令", formatScriptStyleCommand("./ray.sh", segments...)}
	if len(override.RayStartArgs) > 0 {
		lines = append(lines, "# 节点附加 Ray 参数: "+strings.Join(override.RayStartArgs, " "))
	}
	return strings.Join(lines, "\n")
}

func formatScriptStyleCommand(command string, args ...string) string {
	parts := []string{command}
	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			continue
		}
		parts = append(parts, arg)
	}
	if len(parts) == 1 {
		return command
	}
	lines := make([]string, 0, len(parts))
	lines = append(lines, parts[0]+" "+parts[1]+" \\")
	for i := 2; i < len(parts); i++ {
		suffix := ""
		if i < len(parts)-1 {
			suffix = " \\"
		}
		lines = append(lines, "  "+parts[i]+suffix)
	}
	return strings.Join(lines, "\n")
}

func previewShellValue(value string) string {
	if strings.ContainsAny(value, " \t'\"\\") {
		return shellQuote(value)
	}
	return value
}

func buildConfiguredStep(template domain.PipelineTemplate, stepConfigs []domain.PipelineStepTemplate, stepID string, renderValues map[string]string) plannedStep {
	configured := resolvePipelineStep(template, stepConfigs, stepID)
	command := renderStepTemplate(configured.CommandTemplate, renderValues)
	previewTemplate := configured.PreviewTemplate
	if previewTemplate == "" {
		previewTemplate = configured.CommandTemplate
	}
	preview := renderStepTemplate(previewTemplate, renderValues)
	if preview == "" {
		preview = command
	}

	return plannedStep{
		step: domain.DeploymentStep{
			ID:             stepID,
			Name:           configured.Name,
			Description:    configured.Description,
			CommandPreview: preview,
			Optional:       configured.Optional,
			AutoManaged:    configured.AutoManaged,
			Status:         "pending",
			Logs:           []string{},
		},
		command: command,
	}
}

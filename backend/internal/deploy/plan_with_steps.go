package deploy

import (
	"fmt"

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
			LaunchRuntimePreview:       launchCommand,
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

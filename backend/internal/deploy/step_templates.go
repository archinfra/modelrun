package deploy

import (
	"strconv"
	"strings"

	"modelrun/backend/internal/domain"
)

type stepCommandSet struct {
	CheckModelTargetCommand    string
	CheckModelTargetPreview    string
	PrepareModelFetcherCommand string
	PrepareModelFetcherPreview string
	SyncModelCommand           string
	SyncModelPreview           string
	PullImageCommand           string
	PullImagePreview           string
	LaunchRuntimeCommand       string
	LaunchRuntimePreview       string
	VerifyServiceCommand       string
	VerifyServicePreview       string
}

func PipelineTemplatesWithStepConfigs(stepConfigs []domain.PipelineStepTemplate) []domain.PipelineTemplate {
	templates := PipelineTemplates()
	for i := range templates {
		for j := range templates[i].Steps {
			if configured, ok := configuredPipelineStep(stepConfigs, templates[i].Framework, templates[i].Steps[j].ID); ok {
				templates[i].Steps[j].Name = firstNonEmpty(strings.TrimSpace(configured.Name), templates[i].Steps[j].Name)
				templates[i].Steps[j].Description = firstNonEmpty(strings.TrimSpace(configured.Description), templates[i].Steps[j].Description)
				if len(configured.Details) > 0 {
					templates[i].Steps[j].Details = append([]string{}, configured.Details...)
				}
				templates[i].Steps[j].Optional = configured.Optional
				templates[i].Steps[j].AutoManaged = configured.AutoManaged
			}
		}
	}
	return templates
}

func configuredPipelineStep(stepConfigs []domain.PipelineStepTemplate, framework, stepID string) (domain.PipelineStepTemplate, bool) {
	for i := len(stepConfigs) - 1; i >= 0; i-- {
		item := stepConfigs[i]
		if item.Framework == framework && item.StepID == stepID {
			return item, true
		}
	}
	return domain.PipelineStepTemplate{}, false
}

func resolvePipelineStep(template domain.PipelineTemplate, stepConfigs []domain.PipelineStepTemplate, stepID string) domain.PipelineStepTemplate {
	base := domain.PipelineStepTemplate{
		ID:              template.Framework + "_" + stepID,
		Framework:       template.Framework,
		StepID:          stepID,
		CommandTemplate: defaultStepPlaceholder(stepID),
		PreviewTemplate: defaultStepPreviewPlaceholder(stepID),
		CreatedAt:       domain.Now(),
		UpdatedAt:       domain.Now(),
	}
	for _, step := range template.Steps {
		if step.ID != stepID {
			continue
		}
		base.Name = step.Name
		base.Description = step.Description
		base.Optional = step.Optional
		base.AutoManaged = step.AutoManaged
		base.Details = append([]string{}, step.Details...)
		break
	}
	if configured, ok := configuredPipelineStep(stepConfigs, template.Framework, stepID); ok {
		if strings.TrimSpace(configured.Name) != "" {
			base.Name = configured.Name
		}
		if strings.TrimSpace(configured.Description) != "" {
			base.Description = configured.Description
		}
		if strings.TrimSpace(configured.CommandTemplate) != "" {
			base.CommandTemplate = configured.CommandTemplate
		}
		if strings.TrimSpace(configured.PreviewTemplate) != "" {
			base.PreviewTemplate = configured.PreviewTemplate
		}
		if len(configured.Details) > 0 {
			base.Details = append([]string{}, configured.Details...)
		}
		base.Optional = configured.Optional
		base.AutoManaged = configured.AutoManaged
		base.BuiltIn = configured.BuiltIn
		base.ID = configured.ID
		base.CreatedAt = configured.CreatedAt
		base.UpdatedAt = configured.UpdatedAt
	}
	return base
}

func defaultStepPlaceholder(stepID string) string {
	switch stepID {
	case "check_model_target":
		return "{{checkModelTargetCommand}}"
	case "prepare_model_fetcher":
		return "{{prepareModelFetcherCommand}}"
	case "sync_model":
		return "{{syncModelCommand}}"
	case "pull_image":
		return "{{pullImageCommand}}"
	case "launch_runtime":
		return "{{launchRuntimeCommand}}"
	case "verify_service":
		return "{{verifyServiceCommand}}"
	default:
		return ""
	}
}

func defaultStepPreviewPlaceholder(stepID string) string {
	switch stepID {
	case "check_model_target":
		return "{{checkModelTargetPreview}}"
	case "prepare_model_fetcher":
		return "{{prepareModelFetcherPreview}}"
	case "sync_model":
		return "{{syncModelPreview}}"
	case "pull_image":
		return "{{pullImagePreview}}"
	case "launch_runtime":
		return "{{launchRuntimePreview}}"
	case "verify_service":
		return "{{verifyServicePreview}}"
	default:
		return ""
	}
}

func buildPipelineRenderValues(deployment domain.DeploymentConfig, server domain.ServerConfig, runtime domain.DeploymentRuntimeConfig, modelHostPath, workDir, cacheDir, imageRef string, commands stepCommandSet) map[string]string {
	values := map[string]string{
		"framework":                    deployment.Framework,
		"frameworkQuoted":              shellQuote(deployment.Framework),
		"deploymentId":                 deployment.ID,
		"deploymentIdQuoted":           shellQuote(deployment.ID),
		"deploymentName":               deployment.Name,
		"deploymentNameQuoted":         shellQuote(deployment.Name),
		"serverId":                     server.ID,
		"serverIdQuoted":               shellQuote(server.ID),
		"serverName":                   server.Name,
		"serverNameQuoted":             shellQuote(server.Name),
		"serverHost":                   server.Host,
		"serverHostQuoted":             shellQuote(server.Host),
		"modelSource":                  deployment.Model.Source,
		"modelSourceQuoted":            shellQuote(deployment.Model.Source),
		"modelId":                      deployment.Model.ModelID,
		"modelIdQuoted":                shellQuote(deployment.Model.ModelID),
		"modelRevision":                deployment.Model.Revision,
		"modelRevisionQuoted":          shellQuote(deployment.Model.Revision),
		"modelHostPath":                modelHostPath,
		"modelHostPathQuoted":          shellQuote(modelHostPath),
		"containerName":                deploymentContainerName(deployment, runtime),
		"containerNameQuoted":          shellQuote(deploymentContainerName(deployment, runtime)),
		"imageRef":                     imageRef,
		"imageRefQuoted":               shellQuote(imageRef),
		"workDir":                      workDir,
		"workDirQuoted":                shellQuote(workDir),
		"cacheDir":                     cacheDir,
		"cacheDirQuoted":               shellQuote(cacheDir),
		"apiPort":                      strconv.Itoa(deployment.APIPort),
		"apiPortQuoted":                shellQuote(strconv.Itoa(deployment.APIPort)),
		"modelscopeRuntimeImage":       modelscopeRuntimeImage,
		"modelscopeRuntimeImageQuoted": shellQuote(modelscopeRuntimeImage),
		"checkModelTargetCommand":      commands.CheckModelTargetCommand,
		"checkModelTargetPreview":      commands.CheckModelTargetPreview,
		"prepareModelFetcherCommand":   commands.PrepareModelFetcherCommand,
		"prepareModelFetcherPreview":   commands.PrepareModelFetcherPreview,
		"syncModelCommand":             commands.SyncModelCommand,
		"syncModelPreview":             commands.SyncModelPreview,
		"pullImageCommand":             commands.PullImageCommand,
		"pullImagePreview":             commands.PullImagePreview,
		"launchRuntimeCommand":         commands.LaunchRuntimeCommand,
		"launchRuntimePreview":         commands.LaunchRuntimePreview,
		"verifyServiceCommand":         commands.VerifyServiceCommand,
		"verifyServicePreview":         commands.VerifyServicePreview,
	}
	return values
}

func renderStepTemplate(template string, values map[string]string) string {
	rendered := template
	for key, value := range values {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", value)
	}
	return strings.TrimSpace(rendered)
}

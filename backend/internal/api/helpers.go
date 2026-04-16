package api

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"modelrun/backend/internal/deploy"
	"modelrun/backend/internal/domain"
)

func defaultProject(project *domain.Project) {
	now := domain.Now()
	if project.ID == "" {
		project.ID = domain.NewID("project")
	}
	if project.Name == "" {
		project.Name = "New Project"
	}
	if project.Color == "" {
		project.Color = "bg-blue-500"
	}
	if project.CreatedAt == "" {
		project.CreatedAt = now
	}
	project.UpdatedAt = now
	if project.ServerIDs == nil {
		project.ServerIDs = []string{}
	}
}

func defaultServer(server *domain.ServerConfig, projectID string) {
	if server.ID == "" {
		server.ID = domain.NewID("server")
	}
	if server.ProjectID == "" {
		server.ProjectID = projectID
	}
	if server.SSHPort == 0 {
		server.SSHPort = 22
	}
	if server.Username == "" {
		server.Username = "root"
	}
	if server.AuthType == "" {
		server.AuthType = "password"
	}
	if server.Status == "" {
		server.Status = "offline"
	}
	if !server.UseJumpHost {
		server.JumpHostID = ""
	}
}

func defaultModel(model *domain.ModelConfig) {
	if model.ID == "" {
		model.ID = domain.NewID("model")
	}
	if model.Source == "" {
		model.Source = "local"
	}
	if model.Name == "" {
		switch {
		case model.ModelID != "":
			model.Name = baseName(model.ModelID)
		case model.LocalPath != "":
			model.Name = baseName(model.LocalPath)
		default:
			model.Name = "Unnamed Model"
		}
	}
	if model.Format == "" {
		model.Format = inferFormat(*model)
	}
	if model.Parameters == "" {
		model.Parameters = inferParameters(model.Name + " " + model.ModelID)
	}
	if model.Quantization == "" {
		model.Quantization = inferQuantization(model.Name + " " + model.ModelID)
	}
}

func defaultDeployment(deployment *domain.DeploymentConfig) {
	now := domain.Now()
	if deployment.ID == "" {
		deployment.ID = domain.NewID("deployment")
	}
	if deployment.Framework == "" {
		deployment.Framework = "vllm-ascend"
	}
	if deployment.Name == "" {
		deployment.Name = deployment.Model.Name
		if deployment.Name == "" {
			deployment.Name = "New Deployment"
		}
	}
	if deployment.Status == "" {
		deployment.Status = "draft"
	}
	if deployment.CreatedAt == "" {
		deployment.CreatedAt = now
	}
	deployment.UpdatedAt = now
	defaultModel(&deployment.Model)
	if template, ok := deploy.LookupTemplate(deployment.Framework); ok {
		if deployment.APIPort == 0 {
			deployment.APIPort = template.DefaultPort
		}
		if deployment.Docker.Image == "" {
			deployment.Docker.Image = template.DefaultDocker.Image
		}
		if deployment.Docker.Tag == "" {
			deployment.Docker.Tag = template.DefaultDocker.Tag
		}
		if deployment.Docker.Network == "" {
			deployment.Docker.Network = template.DefaultDocker.Network
		}
		if deployment.Docker.IPC == "" {
			deployment.Docker.IPC = template.DefaultDocker.IPC
		}
		if deployment.Docker.ShmSize == "" {
			deployment.Docker.ShmSize = template.DefaultDocker.ShmSize
		}
		if !deployment.Docker.Privileged && template.DefaultDocker.Privileged {
			deployment.Docker.Privileged = true
		}
		if len(deployment.Docker.Volumes) == 0 && len(template.DefaultDocker.Volumes) > 0 {
			deployment.Docker.Volumes = append([]domain.VolumeMount{}, template.DefaultDocker.Volumes...)
		}
		defaultRay(&deployment.Ray, template.DefaultRay)
		defaultRuntime(&deployment.Runtime, template.DefaultRuntime)
	}
	if deployment.APIPort == 0 {
		deployment.APIPort = 8000
	}
	defaultDocker(&deployment.Docker)
	defaultVLLM(&deployment.VLLM)
}

func defaultDocker(docker *domain.DockerConfig) {
	if docker.Image == "" {
		docker.Image = "vllm/vllm-openai"
	}
	if docker.Tag == "" {
		docker.Tag = "latest"
	}
	if docker.GPUDevices == "" {
		docker.GPUDevices = "all"
	}
	if docker.ShmSize == "" {
		docker.ShmSize = "16g"
	}
	if docker.EnvironmentVars == nil {
		docker.EnvironmentVars = map[string]string{}
	}
	if docker.Volumes == nil {
		docker.Volumes = []domain.VolumeMount{}
	}
}

func defaultVLLM(params *domain.VLLMParams) {
	if params.TensorParallelSize == 0 {
		params.TensorParallelSize = 1
	}
	if params.PipelineParallelSize == 0 {
		params.PipelineParallelSize = 1
	}
	if params.MaxModelLen == 0 {
		params.MaxModelLen = 4096
	}
	if params.GPUMemoryUtilization == 0 {
		params.GPUMemoryUtilization = 0.9
	}
	if params.Dtype == "" {
		params.Dtype = "auto"
	}
	if params.MaxNumSeqs == 0 {
		params.MaxNumSeqs = 256
	}
	if params.MaxNumBatchedTokens == 0 {
		params.MaxNumBatchedTokens = 8192
	}
}

func defaultRay(ray *domain.DeploymentRayConfig, defaults domain.DeploymentRayConfig) {
	if ray.Port == 0 {
		ray.Port = defaults.Port
	}
	if ray.DashboardPort == 0 {
		ray.DashboardPort = defaults.DashboardPort
	}
}

func defaultRuntime(runtime *domain.DeploymentRuntimeConfig, defaults domain.DeploymentRuntimeConfig) {
	if runtime.WorkDir == "" {
		runtime.WorkDir = defaults.WorkDir
	}
	if runtime.ModelDir == "" {
		runtime.ModelDir = defaults.ModelDir
	}
	if runtime.CacheDir == "" {
		runtime.CacheDir = defaults.CacheDir
	}
	if runtime.SharedCacheDir == "" {
		runtime.SharedCacheDir = defaults.SharedCacheDir
	}
	if runtime.EnableAutoRestart || defaults.EnableAutoRestart {
		runtime.EnableAutoRestart = true
	}
	if runtime.ExtraArgs == nil {
		runtime.ExtraArgs = append([]string{}, defaults.ExtraArgs...)
	}
}

func inferFormat(model domain.ModelConfig) string {
	text := strings.ToLower(model.Name + " " + model.ModelID + " " + model.LocalPath)
	switch {
	case strings.Contains(text, "awq"):
		return "awq"
	case strings.Contains(text, "gguf"):
		return "gguf"
	case strings.Contains(text, "safetensor"):
		return "safetensors"
	default:
		return "safetensors"
	}
}

func inferParameters(text string) string {
	text = strings.ToLower(text)
	for _, size := range []string{"405b", "236b", "72b", "70b", "32b", "14b", "13b", "8b", "7b", "3b", "1.5b"} {
		if strings.Contains(text, size) {
			return strings.ToUpper(size)
		}
	}
	return ""
}

func inferQuantization(text string) string {
	text = strings.ToLower(text)
	switch {
	case strings.Contains(text, "awq"):
		return "AWQ-4bit"
	case strings.Contains(text, "gptq"):
		return "GPTQ"
	case strings.Contains(text, "int8"):
		return "INT8"
	case strings.Contains(text, "int4"):
		return "INT4"
	default:
		return ""
	}
}

func baseName(value string) string {
	value = strings.TrimRight(filepath.ToSlash(value), "/")
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "/")
	return parts[len(parts)-1]
}

func pathParts(path, prefix string) (string, []string) {
	rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if rest == "" {
		return "", nil
	}
	parts := strings.Split(rest, "/")
	return parts[0], parts[1:]
}

func mergeJSON[T any](base T, patch map[string]json.RawMessage) (T, error) {
	raw, err := json.Marshal(base)
	if err != nil {
		return base, err
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return base, err
	}
	for key, value := range patch {
		fields[key] = value
	}

	raw, err = json.Marshal(fields)
	if err != nil {
		return base, err
	}

	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return base, err
	}
	return out, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func removeString(values []string, target string) []string {
	filtered := values[:0]
	for _, value := range values {
		if value != target {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

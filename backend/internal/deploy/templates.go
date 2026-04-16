package deploy

import "modelrun/backend/internal/domain"

func PipelineTemplates() []domain.PipelineTemplate {
	return []domain.PipelineTemplate{
		{
			ID:          "tei",
			Name:        "TEI Embedding Runtime",
			Framework:   "tei",
			Description: "Text Embeddings Inference pipeline with model preparation, image pull, container bootstrap, and health verification.",
			DefaultPort: 8080,
			DefaultDocker: domain.DockerConfig{
				Image:      "ghcr.io/huggingface/text-embeddings-inference",
				Tag:        "cpu-1.8",
				GPUDevices: "all",
				ShmSize:    "4g",
				Network:    "host",
				IPC:        "host",
				Volumes:    []domain.VolumeMount{},
			},
			DefaultVLLM: domain.VLLMParams{},
			DefaultRay:  domain.DeploymentRayConfig{},
			DefaultRuntime: domain.DeploymentRuntimeConfig{
				WorkDir:           "/opt/modelrun/deployments",
				ModelDir:          "/opt/modelrun/models",
				CacheDir:          "/opt/modelrun/cache",
				EnableAutoRestart: true,
				ExtraArgs:         []string{},
			},
			Steps: []domain.PipelineTemplateStep{
				{
					ID:          "prepare_model",
					Name:        "Prepare model",
					Description: "Download the embedding model or validate the local model path on the target server.",
					Details: []string{
						"ModelScope and Hugging Face sources use CLI download tools on the target server.",
						"Local source only validates the existing path and mounts it directly into the container.",
					},
				},
				{
					ID:          "pull_image",
					Name:        "Pull runtime image",
					Description: "Pull the TEI container image configured for this deployment.",
				},
				{
					ID:          "launch_runtime",
					Name:        "Launch runtime",
					Description: "Write the runtime bootstrap script, recreate the container, and start the TEI service with auto restart.",
					AutoManaged: true,
					Details: []string{
						"Container recreation and service start happen in a single managed command.",
						"Container restart policy is set to unless-stopped so the service comes back automatically.",
					},
				},
				{
					ID:          "verify_service",
					Name:        "Verify service",
					Description: "Probe the local OpenAPI endpoint to confirm the service is responding.",
					AutoManaged: true,
				},
			},
		},
		{
			ID:          "vllm-ascend",
			Name:        "vLLM Ascend Runtime",
			Framework:   "vllm-ascend",
			Description: "vLLM Ascend pipeline with optional Ray cluster bootstrap and OpenAI-compatible service startup.",
			SupportsRay: true,
			DefaultPort: 8000,
			DefaultDocker: domain.DockerConfig{
				Image:      "quay.io/ascend/vllm-ascend",
				Tag:        "v0.11.0rc0",
				GPUDevices: "all",
				ShmSize:    "16g",
				Network:    "host",
				IPC:        "host",
				Privileged: true,
				Volumes: []domain.VolumeMount{
					{Host: "/usr/local/Ascend/driver/lib64", Container: "/usr/local/Ascend/driver/lib64"},
					{Host: "/usr/local/Ascend/driver/version.info", Container: "/usr/local/Ascend/driver/version.info"},
					{Host: "/usr/local/Ascend/ascend_install.info", Container: "/usr/local/Ascend/ascend_install.info"},
					{Host: "/usr/local/bin/npu-smi", Container: "/usr/local/bin/npu-smi"},
				},
			},
			DefaultVLLM: domain.VLLMParams{
				TensorParallelSize:   1,
				PipelineParallelSize: 1,
				MaxModelLen:          4096,
				GPUMemoryUtilization: 0.9,
				Dtype:                "auto",
				TrustRemoteCode:      true,
				EnablePrefixCaching:  true,
				MaxNumSeqs:           256,
				MaxNumBatchedTokens:  8192,
			},
			DefaultRay: domain.DeploymentRayConfig{
				Enabled:       false,
				Port:          6379,
				DashboardPort: 8265,
			},
			DefaultRuntime: domain.DeploymentRuntimeConfig{
				WorkDir:           "/opt/modelrun/deployments",
				ModelDir:          "/opt/modelrun/models",
				CacheDir:          "/opt/modelrun/cache",
				EnableAutoRestart: true,
				ExtraArgs:         []string{},
			},
			Steps: []domain.PipelineTemplateStep{
				{
					ID:          "prepare_model",
					Name:        "Prepare model",
					Description: "Download the model to the managed runtime path or validate the existing model directory.",
				},
				{
					ID:          "pull_image",
					Name:        "Pull runtime image",
					Description: "Pull the selected vLLM Ascend image to every target server.",
				},
				{
					ID:          "launch_runtime",
					Name:        "Launch runtime",
					Description: "Generate the startup script, optionally bootstrap Ray, then start vLLM inside the container.",
					AutoManaged: true,
					Details: []string{
						"Ray bootstrap is optional and only runs when the deployment enables it.",
						"vLLM service startup is bundled with container launch so restart behavior stays consistent.",
					},
				},
				{
					ID:          "verify_service",
					Name:        "Verify service",
					Description: "Check the OpenAI-compatible endpoint on the configured API port.",
					AutoManaged: true,
				},
			},
		},
		{
			ID:          "mindie",
			Name:        "MindIE Service Runtime",
			Framework:   "mindie",
			Description: "MindIE service pipeline with managed config file generation, container bootstrap, and service health verification.",
			DefaultPort: 1025,
			DefaultDocker: domain.DockerConfig{
				Image:      "mindie",
				Tag:        "1.0.0-800I-A2-py311-openeuler24.03-lts",
				GPUDevices: "all",
				ShmSize:    "16g",
				Network:    "host",
				IPC:        "host",
				Privileged: true,
				Volumes: []domain.VolumeMount{
					{Host: "/usr/local/Ascend/driver", Container: "/usr/local/Ascend/driver"},
					{Host: "/usr/local/Ascend/ascend-toolkit", Container: "/usr/local/Ascend/ascend-toolkit"},
					{Host: "/usr/local/Ascend/nnal", Container: "/usr/local/Ascend/nnal"},
				},
			},
			DefaultVLLM: domain.VLLMParams{},
			DefaultRay:  domain.DeploymentRayConfig{},
			DefaultRuntime: domain.DeploymentRuntimeConfig{
				WorkDir:           "/opt/modelrun/deployments",
				ModelDir:          "/opt/modelrun/models",
				CacheDir:          "/opt/modelrun/cache",
				EnableAutoRestart: true,
				ExtraArgs:         []string{},
			},
			Steps: []domain.PipelineTemplateStep{
				{
					ID:          "prepare_model",
					Name:        "Prepare model",
					Description: "Download the model to the managed runtime path or validate the existing model directory.",
				},
				{
					ID:          "pull_image",
					Name:        "Pull runtime image",
					Description: "Pull the configured MindIE image to every target server.",
				},
				{
					ID:          "launch_runtime",
					Name:        "Launch runtime",
					Description: "Generate MindIE config.json, recreate the container, and start the service in one managed action.",
					AutoManaged: true,
					Details: []string{
						"Container launch and MindIE service startup are executed together for restart consistency.",
						"Generated config.json is stored under the managed deployment work directory on the host.",
					},
				},
				{
					ID:          "verify_service",
					Name:        "Verify service",
					Description: "Send an HTTP request to the generated endpoint to confirm the service is reachable.",
					AutoManaged: true,
				},
			},
		},
	}
}

func LookupTemplate(id string) (domain.PipelineTemplate, bool) {
	for _, item := range PipelineTemplates() {
		if item.ID == id || item.Framework == id {
			return item, true
		}
	}
	return domain.PipelineTemplate{}, false
}

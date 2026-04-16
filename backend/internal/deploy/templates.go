package deploy

import "modelrun/backend/internal/domain"

func PipelineTemplates() []domain.PipelineTemplate {
	return []domain.PipelineTemplate{
		{
			ID:          "tei",
			Name:        "TEI 向量服务",
			Framework:   "tei",
			Description: "适用于嵌入模型的 TEI 部署流水线，包含模型准备、镜像拉取、容器启动和健康检查。",
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
					Name:        "准备模型",
					Description: "下载嵌入模型，或校验目标服务器上的本地模型路径。",
					Details: []string{
						"ModelScope 和 Hugging Face 源会在目标服务器上调用 CLI 工具下载模型。",
						"本地源只校验已有目录，然后直接挂载到容器中。",
					},
				},
				{
					ID:          "pull_image",
					Name:        "拉取镜像",
					Description: "在目标服务器上拉取当前部署配置的 TEI 镜像。",
				},
				{
					ID:          "launch_runtime",
					Name:        "启动服务",
					Description: "生成启动脚本、重建容器，并以自动重启方式拉起 TEI 服务。",
					AutoManaged: true,
					Details: []string{
						"容器重建和服务启动会在同一个托管动作里完成。",
						"容器会使用 unless-stopped 重启策略，主机重启后服务也会自动恢复。",
					},
				},
				{
					ID:          "verify_service",
					Name:        "验证服务",
					Description: "探测本地 OpenAPI 端点，确认服务已经可用。",
					AutoManaged: true,
				},
			},
		},
		{
			ID:          "vllm-ascend",
			Name:        "vLLM Ascend 推理服务",
			Framework:   "vllm-ascend",
			Description: "适用于 Ascend/NPU 的 vLLM 流水线，支持 Ray 组网、容器启动和 OpenAI 兼容接口拉起。",
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
					Name:        "准备模型",
					Description: "将模型下载到托管目录，或校验现有模型目录是否可用。",
				},
				{
					ID:          "pull_image",
					Name:        "拉取镜像",
					Description: "将选定的 vLLM Ascend 镜像拉取到每一台目标服务器。",
				},
				{
					ID:          "launch_runtime",
					Name:        "启动服务",
					Description: "生成启动脚本，按需初始化 Ray 集群，并在容器内拉起 vLLM 服务。",
					AutoManaged: true,
					Details: []string{
						"启用 Ray 后，系统会自动区分 head/worker 节点并下发不同启动命令。",
						"容器启动与 vLLM 拉起是同一个托管动作，容器重启后会按同样方式自动恢复。",
					},
				},
				{
					ID:          "verify_service",
					Name:        "验证服务",
					Description: "检查配置的 API 端口，确认 OpenAI 兼容接口已经就绪。",
					AutoManaged: true,
				},
			},
		},
		{
			ID:          "mindie",
			Name:        "MindIE 推理服务",
			Framework:   "mindie",
			Description: "适用于 MindIE 的部署流水线，自动生成配置文件、启动容器并校验服务状态。",
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
					Name:        "准备模型",
					Description: "将模型下载到托管目录，或校验现有模型目录是否可用。",
				},
				{
					ID:          "pull_image",
					Name:        "拉取镜像",
					Description: "将当前配置的 MindIE 镜像拉取到每一台目标服务器。",
				},
				{
					ID:          "launch_runtime",
					Name:        "启动服务",
					Description: "生成 MindIE config.json、重建容器，并在一次托管动作中完成服务启动。",
					AutoManaged: true,
					Details: []string{
						"容器拉起与 MindIE 服务启动会一起执行，便于保持重启行为一致。",
						"生成的 config.json 会保存在主机侧的部署工作目录中。",
					},
				},
				{
					ID:          "verify_service",
					Name:        "验证服务",
					Description: "向生成的服务端点发送 HTTP 请求，确认服务可访问。",
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

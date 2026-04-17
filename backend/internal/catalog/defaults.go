package catalog

import (
	"errors"
	"fmt"
	"strings"

	"modelrun/backend/internal/collect"
	"modelrun/backend/internal/domain"
)

const defaultNetdataImage = "netdata/netdata:stable"

func EnsureDefaults(data *domain.Data) bool {
	changed := false

	actionIndex := map[string]int{}
	for i, item := range data.ActionTemplates {
		actionIndex[item.ID] = i
	}
	for _, item := range DefaultActionTemplates() {
		if idx, ok := actionIndex[item.ID]; ok {
			if refreshBuiltInActionTemplate(&data.ActionTemplates[idx], item) {
				changed = true
			}
			continue
		}
		data.ActionTemplates = append(data.ActionTemplates, item)
		changed = true
	}

	bootstrapIndex := map[string]int{}
	for i, item := range data.BootstrapConfigs {
		bootstrapIndex[item.ID] = i
	}
	for _, item := range DefaultBootstrapConfigs() {
		if idx, ok := bootstrapIndex[item.ID]; ok {
			if refreshBuiltInBootstrapConfig(&data.BootstrapConfigs[idx], item) {
				changed = true
			}
			continue
		}
		data.BootstrapConfigs = append(data.BootstrapConfigs, item)
		changed = true
	}

	pipelineStepIndex := map[string]int{}
	for i, item := range data.PipelineSteps {
		pipelineStepIndex[item.ID] = i
	}
	for _, item := range DefaultPipelineStepTemplates() {
		if _, ok := pipelineStepIndex[item.ID]; ok {
			continue
		}
		data.PipelineSteps = append(data.PipelineSteps, item)
		changed = true
	}

	return changed
}

func DefaultActionTemplates() []domain.ActionTemplate {
	now := domain.Now()
	return []domain.ActionTemplate{
		{
			ID:            "install_node_exporter",
			Name:          "Install node exporter",
			Description:   "Pull and run node_exporter with host metrics mounts and restart policy.",
			Category:      "observability",
			BuiltIn:       true,
			ExecutionType: "command",
			CommandTemplate: withDockerPrivilegesCommand(
				"(run_docker rm -f {{containerName}} >/dev/null 2>&1 || true) && " +
					"run_docker run -d --name {{containerName}} --restart unless-stopped --network host --pid host " +
					"-v /:/host:ro,rslave {{image}} --path.rootfs=/host",
			),
			Fields: []domain.ActionTemplateField{
				{
					Key:          "image",
					Label:        "Image",
					Description:  "node_exporter image to run on the target server.",
					Required:     true,
					DefaultValue: "quay.io/prometheus/node-exporter:v1.8.2",
					Placeholder:  "registry/path/image:tag",
				},
				{
					Key:          "containerName",
					Label:        "Container name",
					Description:  "Container name used on the target server.",
					DefaultValue: "modelrun-node-exporter",
					Placeholder:  "modelrun-node-exporter",
				},
			},
			Tags:      []string{"exporter", "node", "builtin"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:            "install_npu_exporter",
			Name:          "Install NPU exporter",
			Description:   "Pull and run the Ascend NPU exporter with host networking and restart policy.",
			Category:      "observability",
			BuiltIn:       true,
			ExecutionType: "command",
			CommandTemplate: withDockerPrivilegesCommand(
				"(run_docker rm -f {{containerName}} >/dev/null 2>&1 || true) && " +
					"run_docker run -d --name {{containerName}} --restart unless-stopped --network host --privileged " +
					"-v /dev:/dev " +
					"-v /usr/local/Ascend:/usr/local/Ascend:ro " +
					"-v /usr/local/dcmi:/usr/local/dcmi:ro " +
					"-v /sys:/sys:ro " +
					"-v /tmp:/tmp " +
					"-v /var/run/docker.sock:/var/run/docker.sock " +
					"-v /etc/localtime:/etc/localtime:ro {{image}} " +
					"-ip={{listenIP}} -port={{port}} -containerMode=docker",
			),
			Fields: []domain.ActionTemplateField{
				{
					Key:          "image",
					Label:        "Image",
					Description:  "Exporter image to run on the target server.",
					Required:     true,
					DefaultValue: collect.DefaultNPUExporterImage(),
					Placeholder:  "registry/path/image:tag",
				},
				{
					Key:          "containerName",
					Label:        "Container name",
					Description:  "Container name used on the target server.",
					DefaultValue: "modelrun-npu-exporter",
					Placeholder:  "modelrun-npu-exporter",
				},
				{
					Key:          "listenIP",
					Label:        "Listen IP",
					Description:  "Listen IP passed to npu-exporter.",
					DefaultValue: "0.0.0.0",
					Placeholder:  "0.0.0.0",
				},
				{
					Key:          "port",
					Label:        "Listen Port",
					Description:  "Listen port passed to npu-exporter.",
					DefaultValue: "8082",
					Placeholder:  "8082",
				},
			},
			Tags:      []string{"exporter", "npu", "builtin"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:            "install_netdata",
			Name:          "Install Netdata",
			Description:   "Pull and run Netdata in Docker with host access and restart policy.",
			Category:      "observability",
			BuiltIn:       true,
			ExecutionType: "command",
			CommandTemplate: withDockerPrivilegesCommand(
				"(run_docker rm -f {{containerName}} >/dev/null 2>&1 || true) && " +
					"run_docker run -d --name {{containerName}} --restart unless-stopped --network host --pid host --cap-add SYS_PTRACE --cap-add SYS_ADMIN " +
					"-e DO_NOT_TRACK=1 " +
					"-v /:/host/root:ro,rslave " +
					"-v /var/run/docker.sock:/var/run/docker.sock:ro " +
					"-v /proc:/host/proc:ro " +
					"-v /sys:/host/sys:ro " +
					"-v /etc/passwd:/host/etc/passwd:ro " +
					"-v /etc/group:/host/etc/group:ro " +
					"-v /etc/localtime:/etc/localtime:ro " +
					"{{image}}",
			),
			Fields: []domain.ActionTemplateField{
				{
					Key:          "image",
					Label:        "Image",
					Description:  "Netdata image to run on the target server.",
					Required:     true,
					DefaultValue: defaultNetdataImage,
					Placeholder:  "registry/path/image:tag",
				},
				{
					Key:          "containerName",
					Label:        "Container name",
					Description:  "Container name used on the target server.",
					DefaultValue: "modelrun-netdata",
					Placeholder:  "modelrun-netdata",
				},
			},
			Tags:      []string{"netdata", "observability", "builtin"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:            "install_modelscope_cli",
			Name:          "Install ModelScope CLI",
			Description:   "Install the ModelScope command line tools into the current user's environment.",
			Category:      "runtime",
			BuiltIn:       true,
			ExecutionType: "command",
			CommandTemplate: strings.Join([]string{
				"export PATH=\"$PATH:$HOME/.local/bin\";",
				"if command -v modelscope >/dev/null 2>&1; then",
				"echo 'modelscope already installed';",
				"else",
				"command -v python3 >/dev/null 2>&1 || { echo 'python3 is required to install modelscope' >&2; exit 127; };",
				"python3 -m pip install --user modelscope;",
				"modelscope --help >/dev/null 2>&1;",
				"fi",
			}, " "),
			Tags:      []string{"model", "modelscope", "builtin"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:            "install_huggingface_cli",
			Name:          "Install Hugging Face CLI",
			Description:   "Install huggingface-cli into the current user's environment.",
			Category:      "runtime",
			BuiltIn:       true,
			ExecutionType: "command",
			CommandTemplate: strings.Join([]string{
				"export PATH=\"$PATH:$HOME/.local/bin\";",
				"if command -v huggingface-cli >/dev/null 2>&1; then",
				"echo 'huggingface-cli already installed';",
				"else",
				"command -v python3 >/dev/null 2>&1 || { echo 'python3 is required to install huggingface-cli' >&2; exit 127; };",
				"python3 -m pip install --user 'huggingface_hub[cli]';",
				"huggingface-cli --help >/dev/null 2>&1;",
				"fi",
			}, " "),
			Tags:      []string{"model", "huggingface", "builtin"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:            "docker_pull_image",
			Name:          "Docker pull image",
			Description:   "Pull a container image on the target server.",
			Category:      "runtime",
			BuiltIn:       true,
			ExecutionType: "command",
			CommandTemplate: withDockerPrivilegesCommand(
				"run_docker pull {{image}}",
			),
			Fields: []domain.ActionTemplateField{
				{
					Key:         "image",
					Label:       "Image",
					Description: "Full image name including tag.",
					Required:    true,
					Placeholder: "registry/path/image:tag",
				},
			},
			Tags:      []string{"docker", "image", "builtin"},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:            "docker_restart_service",
			Name:          "Restart Docker service",
			Description:   "Restart the Docker daemon using systemd or service fallback.",
			Category:      "maintenance",
			BuiltIn:       true,
			ExecutionType: "command",
			CommandTemplate: withSudoIfNonRoot(
				"if command -v systemctl >/dev/null 2>&1; then systemctl restart docker; else service docker restart; fi",
				"docker service restart requires sudo privileges for the current SSH user.",
			),
			Tags:      []string{"docker", "maintenance", "builtin"},
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
}

func DefaultBootstrapConfigs() []domain.BootstrapConfig {
	now := domain.Now()
	return []domain.BootstrapConfig{
		{
			ID:               "bootstrap_node_exporter",
			Name:             "Node Exporter",
			Description:      "Host metrics exporter for CPU, memory, filesystem, and network.",
			ServiceType:      "node_exporter",
			Category:         "observability",
			BuiltIn:          true,
			ActionTemplateID: "install_node_exporter",
			DefaultArgs: map[string]string{
				"image":         "quay.io/prometheus/node-exporter:v1.8.2",
				"containerName": "modelrun-node-exporter",
			},
			Endpoint:  "http://127.0.0.1:9100/metrics",
			Port:      9100,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:               "bootstrap_npu_exporter",
			Name:             "NPU Exporter",
			Description:      "Ascend NPU metrics exporter scraped by the backend over SSH from the target server.",
			ServiceType:      "npu_exporter",
			Category:         "observability",
			BuiltIn:          true,
			ActionTemplateID: "install_npu_exporter",
			DefaultArgs: map[string]string{
				"image":         collect.DefaultNPUExporterImage(),
				"containerName": "modelrun-npu-exporter",
				"listenIP":      "0.0.0.0",
				"port":          "8082",
			},
			Endpoint:  collect.DefaultNPUExporterEndpoint(),
			Port:      8082,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:               "bootstrap_netdata",
			Name:             "Netdata",
			Description:      "Real-time host monitoring dashboard and lightweight status source.",
			ServiceType:      "netdata",
			Category:         "observability",
			BuiltIn:          true,
			ActionTemplateID: "install_netdata",
			DefaultArgs: map[string]string{
				"image":         defaultNetdataImage,
				"containerName": "modelrun-netdata",
			},
			Endpoint:  collect.DefaultNetdataEndpoint(),
			Port:      19999,
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:               "bootstrap_modelscope_cli",
			Name:             "ModelScope CLI",
			Description:      "Model download tooling used by ModelScope-backed deployment pipelines.",
			ServiceType:      "modelscope_cli",
			Category:         "runtime",
			BuiltIn:          true,
			ActionTemplateID: "install_modelscope_cli",
			DefaultArgs:      map[string]string{},
			CreatedAt:        now,
			UpdatedAt:        now,
		},
		{
			ID:               "bootstrap_huggingface_cli",
			Name:             "Hugging Face CLI",
			Description:      "Model download tooling used by Hugging Face-backed deployment pipelines.",
			ServiceType:      "huggingface_cli",
			Category:         "runtime",
			BuiltIn:          true,
			ActionTemplateID: "install_huggingface_cli",
			DefaultArgs:      map[string]string{},
			CreatedAt:        now,
			UpdatedAt:        now,
		},
	}
}

func DefaultPipelineStepTemplates() []domain.PipelineStepTemplate {
	now := domain.Now()
	items := make([]domain.PipelineStepTemplate, 0, 18)
	for _, framework := range []string{"tei", "vllm-ascend", "mindie"} {
		for _, step := range defaultPipelineStepMetadata(framework) {
			commandTemplate, previewTemplate := defaultPipelineStepCommands(step.ID)
			items = append(items, domain.PipelineStepTemplate{
				ID:              framework + "_" + step.ID,
				Framework:       framework,
				StepID:          step.ID,
				Name:            step.Name,
				Description:     step.Description,
				Optional:        step.Optional,
				AutoManaged:     step.AutoManaged,
				BuiltIn:         true,
				CommandTemplate: commandTemplate,
				PreviewTemplate: previewTemplate,
				Details:         append([]string{}, step.Details...),
				CreatedAt:       now,
				UpdatedAt:       now,
			})
		}
	}
	return items
}

func defaultPipelineStepMetadata(framework string) []domain.PipelineTemplateStep {
	launchDescription := "Generate the runtime launch assets and start the service in Docker."
	launchDetails := []string{
		"The container is recreated with restart policy unless-stopped.",
		"The same launch command is reused after host or container restart.",
	}
	verifyDescription := "Probe the local API endpoint and collect container diagnostics if the probe fails."

	switch framework {
	case "vllm-ascend":
		launchDescription = "Generate launch assets, initialize Ray when enabled, and start the vLLM runtime in Docker."
		launchDetails = []string{
			"内置模板会展示与 ./ray.sh start 兼容的 head / worker 启动命令，便于核对参数。",
			"Ray head 和 worker 会自动区分角色；worker 只加入集群并常驻，不重复执行 vLLM serve。",
			"容器重启后会继续沿用同一套启动脚本和参数，行为和手工脚本保持一致。",
		}
		verifyDescription = "Probe the OpenAI compatible API on the head node, or run ray status checks on worker nodes."
	case "mindie":
		launchDescription = "Generate config.json, recreate the container, and start MindIE in one managed step."
		launchDetails = []string{
			"The generated config.json is written under the deployment work directory on the host.",
			"The container restart policy keeps the runtime behavior consistent after reboot.",
		}
		verifyDescription = "Probe the generated service endpoint and print container diagnostics when startup is incomplete."
	}

	return []domain.PipelineTemplateStep{
		{
			ID:          "check_model_target",
			Name:        "检查模型目录",
			Description: "确认本次部署将使用的模型目录或本地模型路径。",
			Details: []string{
				"远端模型会展示目标目录是否已有模型文件。",
				"本地模型会直接校验路径是否存在。",
			},
		},
		{
			ID:          "prepare_model_fetcher",
			Name:        "准备模型下载器",
			Description: "按模型来源准备下载工具；ModelScope 缺失时会自动切换到容器化命令。",
			Details: []string{
				"ModelScope 优先使用远端机器本地的 modelscope 命令。",
				"如果远端没有 modelscope，则自动使用 registry.cn-beijing.aliyuncs.com/ainfracn/modelscope:1.35.0。",
			},
		},
		{
			ID:          "sync_model",
			Name:        "同步模型",
			Description: "模型文件存在时直接复用，不存在时再执行下载或校验。",
		},
		{
			ID:          "pull_image",
			Name:        "拉取镜像",
			Description: "在目标服务器拉取当前部署所需的运行时镜像。",
		},
		{
			ID:          "launch_runtime",
			Name:        "启动服务",
			Description: launchDescription,
			AutoManaged: true,
			Details:     launchDetails,
		},
		{
			ID:          "verify_service",
			Name:        "验证服务",
			Description: verifyDescription,
			AutoManaged: true,
		},
	}
}

func defaultPipelineStepCommands(stepID string) (string, string) {
	switch stepID {
	case "check_model_target":
		return "{{checkModelTargetCommand}}", "{{checkModelTargetPreview}}"
	case "prepare_model_fetcher":
		return "{{prepareModelFetcherCommand}}", "{{prepareModelFetcherPreview}}"
	case "sync_model":
		return "{{syncModelCommand}}", "{{syncModelPreview}}"
	case "pull_image":
		return "{{pullImageCommand}}", "{{pullImagePreview}}"
	case "launch_runtime":
		return "{{launchRuntimeCommand}}", "{{launchRuntimePreview}}"
	case "verify_service":
		return "{{verifyServiceCommand}}", "{{verifyServicePreview}}"
	default:
		return "", ""
	}
}

func ToRemoteTaskPreset(action domain.ActionTemplate) domain.RemoteTaskPreset {
	fields := make([]domain.RemoteTaskPresetField, 0, len(action.Fields))
	for _, field := range action.Fields {
		fields = append(fields, domain.RemoteTaskPresetField{
			Key:          field.Key,
			Label:        field.Label,
			Description:  field.Description,
			Required:     field.Required,
			DefaultValue: field.DefaultValue,
			Placeholder:  field.Placeholder,
		})
	}
	return domain.RemoteTaskPreset{
		ID:          action.ID,
		Name:        action.Name,
		Description: action.Description,
		Fields:      fields,
	}
}

func LookupActionTemplate(actions []domain.ActionTemplate, id string) (domain.ActionTemplate, bool) {
	id = strings.TrimSpace(id)
	if alias, ok := actionTemplateAliases[id]; ok {
		id = alias
	}
	for _, action := range actions {
		if action.ID == id {
			return action, true
		}
	}
	return domain.ActionTemplate{}, false
}

var actionTemplateAliases = map[string]string{
	"docker_install_npu_exporter":  "install_npu_exporter",
	"docker_install_node_exporter": "install_node_exporter",
}

func refreshBuiltInActionTemplate(current *domain.ActionTemplate, defaults domain.ActionTemplate) bool {
	if !current.BuiltIn {
		return false
	}
	changed := false
	switch current.ID {
	case "install_npu_exporter":
		if !strings.Contains(current.CommandTemplate, "-ip={{listenIP}}") {
			current.CommandTemplate = defaults.CommandTemplate
			current.Description = defaults.Description
			current.Fields = defaults.Fields
			changed = true
			break
		}
		for i := range current.Fields {
			switch current.Fields[i].Key {
			case "image":
				if current.Fields[i].DefaultValue == "" || current.Fields[i].DefaultValue == "swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v2.0.1" {
					if current.Fields[i].DefaultValue != collect.DefaultNPUExporterImage() {
						current.Fields[i].DefaultValue = collect.DefaultNPUExporterImage()
						changed = true
					}
				}
			case "listenIP":
				if strings.TrimSpace(current.Fields[i].DefaultValue) == "" {
					current.Fields[i].DefaultValue = "0.0.0.0"
					changed = true
				}
			case "port":
				if strings.TrimSpace(current.Fields[i].DefaultValue) == "" {
					current.Fields[i].DefaultValue = "8082"
					changed = true
				}
			}
		}
	case "install_netdata":
		if strings.Contains(current.CommandTemplate, "kickstart.sh") {
			current.CommandTemplate = defaults.CommandTemplate
			current.Description = defaults.Description
			current.Fields = defaults.Fields
			changed = true
		}
	}
	if changed {
		current.UpdatedAt = domain.Now()
	}
	return changed
}

func refreshBuiltInBootstrapConfig(current *domain.BootstrapConfig, defaults domain.BootstrapConfig) bool {
	if !current.BuiltIn || current.ID != "bootstrap_npu_exporter" {
		if !current.BuiltIn || current.ID != "bootstrap_netdata" {
			return false
		}
	}
	changed := false
	switch current.ID {
	case "bootstrap_npu_exporter":
		if current.DefaultArgs == nil {
			current.DefaultArgs = map[string]string{}
		}
		if value := strings.TrimSpace(current.DefaultArgs["image"]); value == "" || value == "swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v2.0.1" {
			if current.DefaultArgs["image"] != collect.DefaultNPUExporterImage() {
				current.DefaultArgs["image"] = collect.DefaultNPUExporterImage()
				changed = true
			}
		}
		if strings.TrimSpace(current.Endpoint) == "" || current.Endpoint == "http://127.0.0.1:9101/metrics" {
			if current.Endpoint != defaults.Endpoint {
				current.Endpoint = defaults.Endpoint
				changed = true
			}
		}
		if current.Port == 0 || current.Port == 9101 {
			if current.Port != defaults.Port {
				current.Port = defaults.Port
				changed = true
			}
		}
		if strings.TrimSpace(current.DefaultArgs["listenIP"]) == "" {
			current.DefaultArgs["listenIP"] = "0.0.0.0"
			changed = true
		}
		if strings.TrimSpace(current.DefaultArgs["port"]) == "" {
			current.DefaultArgs["port"] = "8082"
			changed = true
		}
	case "bootstrap_netdata":
		if current.DefaultArgs == nil {
			current.DefaultArgs = map[string]string{}
		}
		if strings.TrimSpace(current.DefaultArgs["image"]) == "" {
			current.DefaultArgs["image"] = defaultNetdataImage
			changed = true
		}
		if strings.TrimSpace(current.DefaultArgs["containerName"]) == "" {
			current.DefaultArgs["containerName"] = "modelrun-netdata"
			changed = true
		}
	}
	if changed {
		current.UpdatedAt = domain.Now()
	}
	return changed
}

func BuildActionCommand(action domain.ActionTemplate, args map[string]string) (string, error) {
	values, err := resolvedTemplateValues(action.Fields, args)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(strings.TrimSpace(action.ExecutionType)) {
	case "", "command":
		if strings.TrimSpace(action.CommandTemplate) == "" {
			return "", errors.New("commandTemplate is required")
		}
		return renderShellTemplate(action.CommandTemplate, values), nil
	case "script_url":
		url := renderRawTemplate(action.ScriptURL, values)
		if strings.TrimSpace(url) == "" {
			return "", errors.New("scriptUrl is required")
		}
		scriptArgs := renderRawTemplate(action.ScriptArgsTemplate, values)
		return collect.BuildScriptURLCommand(url, scriptArgs)
	default:
		return "", fmt.Errorf("unsupported action executionType %q", action.ExecutionType)
	}
}

func resolvedTemplateValues(fields []domain.ActionTemplateField, args map[string]string) (map[string]string, error) {
	values := map[string]string{}
	for _, field := range fields {
		value := strings.TrimSpace(args[field.Key])
		if value == "" {
			value = strings.TrimSpace(field.DefaultValue)
		}
		if field.Required && value == "" {
			return nil, fmt.Errorf("field %q is required", field.Key)
		}
		values[field.Key] = value
	}
	for key, value := range args {
		if _, ok := values[key]; ok {
			continue
		}
		values[key] = strings.TrimSpace(value)
	}
	return values, nil
}

func renderShellTemplate(template string, values map[string]string) string {
	rendered := template
	for key, value := range values {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", shellQuote(value))
	}
	return rendered
}

func renderRawTemplate(template string, values map[string]string) string {
	rendered := template
	for key, value := range values {
		rendered = strings.ReplaceAll(rendered, "{{"+key+"}}", value)
	}
	return rendered
}

func withDockerPrivilegesCommand(body string) string {
	return strings.Join([]string{
		"command -v docker >/dev/null 2>&1 || { echo 'docker is not installed' >&2; exit 127; };",
		"run_docker(){",
		"if [ \"$(id -u)\" -eq 0 ]; then",
		"docker \"$@\";",
		"return $?;",
		"fi;",
		"if command -v sudo >/dev/null 2>&1; then",
		"sudo -n docker \"$@\";",
		"status=$?;",
		"if [ $status -ne 0 ]; then",
		"echo 'docker command requires sudo privileges for the current SSH user, or the user must be added to the docker group.' >&2;",
		"fi;",
		"return $status;",
		"fi;",
		"echo 'docker command requires sudo privileges because the current SSH user is not root and sudo is unavailable.' >&2;",
		"return 1;",
		"};",
		body,
	}, " ")
}

func withSudoIfNonRoot(command, failureHint string) string {
	return strings.Join([]string{
		"if [ \"$(id -u)\" -eq 0 ]; then",
		command + ";",
		"elif command -v sudo >/dev/null 2>&1; then",
		"sudo -n sh -lc " + shellQuote(command) + ";",
		"status=$?;",
		"if [ $status -ne 0 ]; then",
		"echo " + shellQuote(failureHint) + " >&2;",
		"fi;",
		"exit $status;",
		"else",
		"echo " + shellQuote(failureHint) + " >&2;",
		"exit 1;",
		"fi",
	}, " ")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

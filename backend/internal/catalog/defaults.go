package catalog

import (
	"errors"
	"fmt"
	"strings"

	"modelrun/backend/internal/collect"
	"modelrun/backend/internal/domain"
)

func EnsureDefaults(data *domain.Data) bool {
	changed := false

	actionIndex := map[string]int{}
	for i, item := range data.ActionTemplates {
		actionIndex[item.ID] = i
	}
	for _, item := range DefaultActionTemplates() {
		if _, ok := actionIndex[item.ID]; ok {
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
		if _, ok := bootstrapIndex[item.ID]; ok {
			continue
		}
		data.BootstrapConfigs = append(data.BootstrapConfigs, item)
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
					"-v /etc/localtime:/etc/localtime:ro {{image}}",
			),
			Fields: []domain.ActionTemplateField{
				{
					Key:          "image",
					Label:        "Image",
					Description:  "Exporter image to run on the target server.",
					Required:     true,
					DefaultValue: "swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v2.0.1",
					Placeholder:  "registry/path/image:tag",
				},
				{
					Key:          "containerName",
					Label:        "Container name",
					Description:  "Container name used on the target server.",
					DefaultValue: "modelrun-npu-exporter",
					Placeholder:  "modelrun-npu-exporter",
				},
			},
			Tags:      []string{"exporter", "npu", "builtin"},
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
				"image":         "swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v2.0.1",
				"containerName": "modelrun-npu-exporter",
			},
			Endpoint:  "http://127.0.0.1:9101/metrics",
			Port:      9101,
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

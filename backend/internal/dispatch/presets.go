package dispatch

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"modelrun/backend/internal/domain"
)

type presetDefinition struct {
	meta  domain.RemoteTaskPreset
	build func(args map[string]string) (string, error)
}

var presetCatalog = map[string]presetDefinition{
	"docker_install_npu_exporter": {
		meta: domain.RemoteTaskPreset{
			ID:          "docker_install_npu_exporter",
			Name:        "Docker install NPU exporter",
			Description: "Pull and run the Ascend NPU exporter container with host networking and restart policy.",
			Fields: []domain.RemoteTaskPresetField{
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
					Description:  "Container name used on every target server.",
					DefaultValue: "modelrun-npu-exporter",
					Placeholder:  "modelrun-npu-exporter",
				},
			},
		},
		build: func(args map[string]string) (string, error) {
			image := strings.TrimSpace(args["image"])
			if image == "" {
				return "", errors.New("preset arg image is required")
			}
			containerName := strings.TrimSpace(args["containerName"])
			if containerName == "" {
				containerName = "modelrun-npu-exporter"
			}
			return withDockerPrivileges(
				"(run_docker rm -f " + collectShellQuote(containerName) + " >/dev/null 2>&1 || true) && " +
					"run_docker run -d --name " + collectShellQuote(containerName) + " --restart unless-stopped --network host --privileged " +
					"-v /dev:/dev " +
					"-v /usr/local/Ascend:/usr/local/Ascend:ro " +
					"-v /etc/localtime:/etc/localtime:ro " +
					collectShellQuote(image),
			), nil
		},
	},
	"docker_pull_image": {
		meta: domain.RemoteTaskPreset{
			ID:          "docker_pull_image",
			Name:        "Docker pull image",
			Description: "Pull a container image on every selected robot.",
			Fields: []domain.RemoteTaskPresetField{
				{
					Key:         "image",
					Label:       "Image",
					Description: "Full image name including tag.",
					Required:    true,
					Placeholder: "vllm/vllm-openai:latest",
				},
			},
		},
		build: func(args map[string]string) (string, error) {
			image := strings.TrimSpace(args["image"])
			if image == "" {
				return "", errors.New("preset arg image is required")
			}
			return withDockerPrivileges("run_docker pull " + collectShellQuote(image)), nil
		},
	},
	"docker_restart_service": {
		meta: domain.RemoteTaskPreset{
			ID:          "docker_restart_service",
			Name:        "Restart Docker service",
			Description: "Restart the Docker daemon using systemd or service fallback.",
		},
		build: func(_ map[string]string) (string, error) {
			return withSudoIfNonRoot(
				"if command -v systemctl >/dev/null 2>&1; then systemctl restart docker; else service docker restart; fi",
				"docker service restart requires sudo privileges for the current SSH user.",
			), nil
		},
	},
}

func Presets() []domain.RemoteTaskPreset {
	presets := make([]domain.RemoteTaskPreset, 0, len(presetCatalog))
	for _, preset := range presetCatalog {
		presets = append(presets, preset.meta)
	}
	sort.Slice(presets, func(i, j int) bool { return presets[i].Name < presets[j].Name })
	return presets
}

func LookupPreset(id string) (domain.RemoteTaskPreset, bool) {
	preset, ok := presetCatalog[strings.TrimSpace(id)]
	if !ok {
		return domain.RemoteTaskPreset{}, false
	}
	return preset.meta, true
}

func BuildPresetCommand(id string, args map[string]string) (string, error) {
	preset, ok := presetCatalog[strings.TrimSpace(id)]
	if !ok {
		return "", fmt.Errorf("unknown preset %q", id)
	}
	return preset.build(args)
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
		"sudo -n sh -lc " + collectShellQuote(command) + ";",
		"status=$?;",
		"if [ $status -ne 0 ]; then",
		"echo " + collectShellQuote(failureHint) + " >&2;",
		"fi;",
		"exit $status;",
		"else",
		"echo " + collectShellQuote(failureHint) + " >&2;",
		"exit 1;",
		"fi",
	}, " ")
}

func collectShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

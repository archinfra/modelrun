package dispatch

import (
	"strings"
	"testing"
)

func TestBuildPresetCommand(t *testing.T) {
	command, err := BuildPresetCommand("docker_install_npu_exporter", map[string]string{
		"image":         "repo/exporter:v1",
		"containerName": "npu-exporter",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(command, "run_docker run -d") {
		t.Fatalf("expected docker run command, got %q", command)
	}
	if !strings.Contains(command, "repo/exporter:v1") {
		t.Fatalf("expected image in command, got %q", command)
	}
	if !strings.Contains(command, "sudo -n docker") {
		t.Fatalf("expected sudo fallback in command, got %q", command)
	}
}

func TestBuildDockerRestartPresetCommand(t *testing.T) {
	command, err := BuildPresetCommand("docker_restart_service", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(command, "sudo -n sh -lc") {
		t.Fatalf("expected sudo wrapper in command, got %q", command)
	}
}

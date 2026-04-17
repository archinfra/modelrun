package collect

import (
	"strings"
	"testing"
)

func TestBuildNPUExporterInstallCommandUsesDockerPrivilegeWrapper(t *testing.T) {
	command, endpoint, err := buildNPUExporterInstallCommand(NPUExporterInstallOptions{
		Mode:  "docker",
		Image: "example.com/npu-exporter:latest",
	}, "")
	if err != nil {
		t.Fatalf("buildNPUExporterInstallCommand returned error: %v", err)
	}
	if endpoint != "http://127.0.0.1:8082/metrics" {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
	for _, needle := range []string{
		"run_docker(){",
		"sudo -n docker",
		"run_docker run -d --name modelrun-npu-exporter",
	} {
		if !strings.Contains(command, needle) {
			t.Fatalf("expected command to contain %q, got %q", needle, command)
		}
	}
}

func TestDefaultNPUExporterValues(t *testing.T) {
	if DefaultNPUExporterImage() != "swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v7.3.0" {
		t.Fatalf("unexpected default image: %s", DefaultNPUExporterImage())
	}
	if DefaultNPUExporterEndpoint() != "http://127.0.0.1:8082/metrics" {
		t.Fatalf("unexpected default endpoint: %s", DefaultNPUExporterEndpoint())
	}
}

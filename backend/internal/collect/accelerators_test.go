package collect

import "testing"

func TestParseAscendNPUInfo(t *testing.T) {
	out := `
+------+--------+----------+
| 0 910B | OK | 180 65 |
| 0 0 | 0000 | 28 8192 / 65536 |
| 1 Ascend910B | OK | 190 67 |
| 1 1 | 0000 | 42 16384/65536 |
`

	devices := parseAscendNPUInfo(out)
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
	if devices[0].Type != "npu" || devices[0].Name != "Ascend 910B" {
		t.Fatalf("unexpected first device identity: %#v", devices[0])
	}
	if devices[0].Utilization != 28 || devices[0].MemoryUsed != 8192 || devices[0].MemoryTotal != 65536 {
		t.Fatalf("unexpected first device metrics: %#v", devices[0])
	}
	if devices[1].Name != "Ascend910B" || devices[1].LogicID != 1 || devices[1].MemoryFree != 49152 {
		t.Fatalf("unexpected second device: %#v", devices[1])
	}
}

func TestParseNPUExporterMetrics(t *testing.T) {
	out := `
# HELP npu_chip_info_utilization the ai core utilization
npu_chip_info_name{npuID="0",name="Ascend910B",uuid="uuid0",pcie="0000",namespace="",podName="",containerName=""} 1
npu_chip_info_utilization{npuID="0",modelName="Ascend910B",uuid="uuid0",pcie="0000",namespace="",podName="",containerName=""} 36
npu_chip_info_temperature{npuID="0",modelName="Ascend910B",uuid="uuid0",pcie="0000",namespace="",podName="",containerName=""} 62
npu_chip_info_power{npuID="0",modelName="Ascend910B",uuid="uuid0",pcie="0000",namespace="",podName="",containerName=""} 188
npu_chip_info_hbm_total_memory{npuID="0",modelName="Ascend910B",uuid="uuid0",pcie="0000",namespace="",podName="",containerName=""} 65536
npu_chip_info_hbm_used_memory{npuID="0",modelName="Ascend910B",uuid="uuid0",pcie="0000",namespace="",podName="",containerName=""} 8192
npu_chip_info_health_status{npuID="0",modelName="Ascend910B",uuid="uuid0",pcie="0000",namespace="",podName="",containerName=""} 1
`

	devices := parseNPUExporterMetrics(out)
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	device := devices[0]
	if device.Name != "Ascend910B" || device.Type != "npu" || device.Health != "OK" {
		t.Fatalf("unexpected identity: %#v", device)
	}
	if device.Utilization != 36 || device.Temperature != 62 || device.PowerDraw != 188 {
		t.Fatalf("unexpected live metrics: %#v", device)
	}
	if device.MemoryTotal != 65536 || device.MemoryUsed != 8192 || device.MemoryFree != 57344 {
		t.Fatalf("unexpected memory metrics: %#v", device)
	}
}

func TestParseNPUExporterMetricsPrefersHBMMemoryOverGenericZeroMetrics(t *testing.T) {
	out := `
npu_chip_info_name{id="0",name="910B2n-Ascend-V1"} 1 1776313463623
npu_chip_info_hbm_total_memory{id="0"} 65536 1776313463623
npu_chip_info_hbm_used_memory{id="0"} 3399 1776313463623
npu_chip_info_power{id="0"} 97.4 1776313463623
npu_chip_info_temperature{id="0"} 36 1776313463623
npu_chip_info_total_memory{id="0"} 0 1776313463623
npu_chip_info_used_memory{id="0"} 0 1776313463623
`

	devices := parseNPUExporterMetrics(out)
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	device := devices[0]
	if device.Name != "910B2n-Ascend-V1" {
		t.Fatalf("unexpected name: %#v", device)
	}
	if device.MemoryTotal != 65536 || device.MemoryUsed != 3399 || device.MemoryFree != 62137 {
		t.Fatalf("expected HBM memory to win over zero generic metrics, got %#v", device)
	}
}

func TestParseNPUExporterMetricsSupportsSnakeCaseLabelsAndAlternateMetricNames(t *testing.T) {
	out := `
npu_chip_info_name{id="1",model_name="Ascend 910B4"} 1
npu_chip_info_aicore_utilization{id="1",model_name="Ascend 910B4"} 72
npu_chip_info_chip_temperature{id="1"} 41
npu_chip_info_power_usage{id="1"} 205.5
npu_chip_info_hbm_total_memory{id="1"} 68719476736
npu_chip_info_hbm_used_memory{id="1"} 17179869184
npu_chip_info_health{id="1"} 1
`

	devices := parseNPUExporterMetrics(out)
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	device := devices[0]
	if device.Name != "Ascend 910B4" || device.Index != 1 {
		t.Fatalf("unexpected identity: %#v", device)
	}
	if device.Utilization != 72 || device.Temperature != 41 || device.PowerDraw != 205.5 {
		t.Fatalf("unexpected live metrics: %#v", device)
	}
	if device.MemoryTotal != 65536 || device.MemoryUsed != 16384 || device.MemoryFree != 49152 {
		t.Fatalf("unexpected memory normalization: %#v", device)
	}
	if device.Health != "OK" {
		t.Fatalf("expected health OK, got %#v", device)
	}
}

func TestNormalizeAcceleratorMemory(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  int64
	}{
		{name: "mib", value: 65536, want: 65536},
		{name: "kib", value: 67108864, want: 65536},
		{name: "bytes", value: 68719476736, want: 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeAcceleratorMemory(tt.value); got != tt.want {
				t.Fatalf("normalizeAcceleratorMemory(%v) = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestNPUExporterEndpoints(t *testing.T) {
	endpoints := npuExporterEndpoints("")
	if len(endpoints) < 2 {
		t.Fatalf("expected fallback endpoints, got %#v", endpoints)
	}
	if endpoints[0] != defaultNPUExporterEndpoint {
		t.Fatalf("expected default endpoint first, got %#v", endpoints)
	}
	foundAlternate := false
	for _, endpoint := range endpoints {
		if endpoint == alternateNPUExporterEndpoint {
			foundAlternate = true
			break
		}
	}
	if !foundAlternate {
		t.Fatalf("expected alternate endpoint in candidates, got %#v", endpoints)
	}
}

func TestNPUExporterEndpointsKeepConfiguredEndpointAndFallbacks(t *testing.T) {
	endpoints := npuExporterEndpoints("http://127.0.0.1:9101/metrics")
	if len(endpoints) < 2 {
		t.Fatalf("expected configured endpoint plus fallback candidates, got %#v", endpoints)
	}
	if endpoints[0] != "http://127.0.0.1:9101/metrics" {
		t.Fatalf("expected configured endpoint first, got %#v", endpoints)
	}
	if endpoints[1] != defaultNPUExporterEndpoint {
		t.Fatalf("expected default endpoint as next fallback, got %#v", endpoints)
	}
}

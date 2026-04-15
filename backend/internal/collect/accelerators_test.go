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

package collect

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"modelrun/backend/internal/domain"

	"golang.org/x/crypto/ssh"
)

const defaultNPUExporterEndpoint = "http://127.0.0.1:9101/metrics"

type NPUExporterStatus struct {
	Endpoint         string           `json:"endpoint"`
	Reachable        bool             `json:"reachable"`
	Message          string           `json:"message"`
	AcceleratorCount int              `json:"acceleratorCount"`
	Accelerators     []domain.GPUInfo `json:"accelerators,omitempty"`
}

type NPUExporterInstallOptions struct {
	Mode     string `json:"mode"`
	Image    string `json:"image,omitempty"`
	Command  string `json:"command,omitempty"`
	Endpoint string `json:"endpoint,omitempty"`
	Port     int    `json:"port,omitempty"`
}

type NPUExporterInstallResult struct {
	Endpoint string            `json:"endpoint"`
	Command  string            `json:"command"`
	Output   string            `json:"output,omitempty"`
	Status   NPUExporterStatus `json:"status"`
}

func collectNPUExporter(client sshRunner, configuredEndpoint string) ([]domain.GPUInfo, error) {
	out, err := fetchNPUExporterMetrics(client, npuExporterEndpoint(configuredEndpoint))
	if err != nil {
		return nil, err
	}
	devices := parseNPUExporterMetrics(out)
	if len(devices) == 0 {
		return nil, errors.New("npu exporter metrics did not contain npu devices")
	}
	return devices, nil
}

func (c *Collector) NPUExporterStatus(server domain.ServerConfig, jump *SSHConfig) (NPUExporterStatus, error) {
	endpoint := npuExporterEndpoint(server.NPUExporterEndpoint)
	if IsMockServer(server) {
		accelerators := []domain.GPUInfo{{Index: 0, Type: "npu", Name: "Ascend 910B", MemoryTotal: 65536, MemoryUsed: 8192, MemoryFree: 57344, Utilization: 28, Temperature: 65, PowerDraw: 180, Health: "OK"}}
		return NPUExporterStatus{
			Endpoint:         endpoint,
			Reachable:        true,
			Message:          "mock npu exporter is reachable",
			AcceleratorCount: len(accelerators),
			Accelerators:     accelerators,
		}, nil
	}

	client, closeFn, err := c.dial(FromServer(server), jump)
	if err != nil {
		return NPUExporterStatus{}, err
	}
	defer closeFn()

	return npuExporterStatusFromClient(client, endpoint), nil
}

func (c *Collector) InstallNPUExporter(server domain.ServerConfig, jump *SSHConfig, opts NPUExporterInstallOptions) (NPUExporterInstallResult, error) {
	endpoint := npuExporterEndpoint(firstNonEmpty(opts.Endpoint, server.NPUExporterEndpoint))
	if IsMockServer(server) {
		status, _ := c.NPUExporterStatus(server, jump)
		return NPUExporterInstallResult{
			Endpoint: endpoint,
			Command:  "mock install",
			Output:   "mock npu exporter installed",
			Status:   status,
		}, nil
	}

	client, closeFn, err := c.dial(FromServer(server), jump)
	if err != nil {
		return NPUExporterInstallResult{}, err
	}
	defer closeFn()

	command, endpoint, err := buildNPUExporterInstallCommand(opts, endpoint)
	if err != nil {
		return NPUExporterInstallResult{}, err
	}

	output, err := run(client, command)
	if err != nil {
		return NPUExporterInstallResult{}, err
	}

	status := npuExporterStatusFromClient(client, endpoint)
	return NPUExporterInstallResult{
		Endpoint: endpoint,
		Command:  command,
		Output:   output,
		Status:   status,
	}, nil
}

type sshRunner interface {
	NewSession() (*ssh.Session, error)
}

func npuExporterEndpoint(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	if env := strings.TrimSpace(os.Getenv("MODELRUN_NPU_EXPORTER_ENDPOINT")); env != "" {
		return env
	}
	return defaultNPUExporterEndpoint
}

func fetchNPUExporterMetrics(client sshRunner, endpoint string) (string, error) {
	command := "if command -v curl >/dev/null 2>&1; then " +
		"curl -fsS --max-time 3 " + shellQuote(endpoint) + "; " +
		"elif command -v wget >/dev/null 2>&1; then " +
		"wget -q -T 3 -O - " + shellQuote(endpoint) + "; " +
		"else echo 'curl or wget is required to scrape npu exporter' >&2; exit 127; fi"
	return run(client, command)
}

func npuExporterStatusFromClient(client sshRunner, endpoint string) NPUExporterStatus {
	status := NPUExporterStatus{Endpoint: endpoint}
	out, err := fetchNPUExporterMetrics(client, endpoint)
	if err != nil {
		status.Message = err.Error()
		return status
	}

	accelerators := parseNPUExporterMetrics(out)
	status.Reachable = true
	status.Accelerators = accelerators
	status.AcceleratorCount = len(accelerators)
	status.Message = fmt.Sprintf("npu exporter returned %d accelerator(s)", len(accelerators))
	return status
}

func buildNPUExporterInstallCommand(opts NPUExporterInstallOptions, endpoint string) (string, string, error) {
	mode := strings.ToLower(strings.TrimSpace(opts.Mode))
	if mode == "" {
		if strings.TrimSpace(opts.Command) != "" {
			mode = "command"
		} else {
			mode = "docker"
		}
	}

	switch mode {
	case "command", "custom":
		if strings.TrimSpace(opts.Command) == "" {
			return "", endpoint, errors.New("command is required for custom npu exporter install")
		}
		return opts.Command, endpoint, nil
	case "docker":
		if strings.TrimSpace(opts.Image) == "" {
			return "", endpoint, errors.New("image is required for docker npu exporter install")
		}
		port := opts.Port
		if port == 0 {
			port = 9101
		}
		if strings.TrimSpace(opts.Endpoint) == "" {
			endpoint = fmt.Sprintf("http://127.0.0.1:%d/metrics", port)
		}
		command := "command -v docker >/dev/null 2>&1 && " +
			"(docker rm -f modelrun-npu-exporter >/dev/null 2>&1 || true) && " +
			"docker run -d --name modelrun-npu-exporter --restart unless-stopped --network host --privileged " +
			"-v /dev:/dev " +
			"-v /usr/local/Ascend:/usr/local/Ascend:ro " +
			"-v /etc/localtime:/etc/localtime:ro " +
			shellQuote(opts.Image)
		return command, endpoint, nil
	default:
		return "", endpoint, fmt.Errorf("unsupported npu exporter install mode %q", opts.Mode)
	}
}

type prometheusSample struct {
	name   string
	labels map[string]string
	value  float64
}

func parseNPUExporterMetrics(out string) []domain.GPUInfo {
	devices := map[int]*domain.GPUInfo{}

	for _, line := range strings.Split(out, "\n") {
		sample, ok := parsePrometheusSample(line)
		if !ok || !strings.HasPrefix(sample.name, "npu_chip_info_") {
			continue
		}
		index, ok := metricNPUIndex(sample.labels)
		if !ok {
			continue
		}

		device := devices[index]
		if device == nil {
			device = &domain.GPUInfo{Index: index, Type: "npu", Name: "Ascend NPU"}
			devices[index] = device
		}
		if name := firstNonEmpty(sample.labels["name"], sample.labels["modelName"]); name != "" {
			device.Name = normalizeNPUName(name)
		}

		switch sample.name {
		case "npu_chip_info_name":
			if sample.value > 0 {
				if name := firstNonEmpty(sample.labels["name"], sample.labels["modelName"]); name != "" {
					device.Name = normalizeNPUName(name)
				}
			}
		case "npu_chip_info_utilization", "npu_chip_info_overall_utilization":
			device.Utilization = sample.value
		case "npu_chip_info_temperature":
			device.Temperature = sample.value
		case "npu_chip_info_power":
			device.PowerDraw = sample.value
		case "npu_chip_info_hbm_total_memory", "npu_chip_info_total_memory":
			device.MemoryTotal = int64(sample.value)
		case "npu_chip_info_hbm_used_memory", "npu_chip_info_used_memory":
			device.MemoryUsed = int64(sample.value)
		case "npu_chip_info_health_status":
			if sample.value == 1 {
				device.Health = "OK"
			} else {
				device.Health = "UNHEALTHY"
			}
		}
	}

	result := make([]domain.GPUInfo, 0, len(devices))
	for _, device := range devices {
		if device.MemoryTotal > 0 {
			device.MemoryFree = maxInt64(0, device.MemoryTotal-device.MemoryUsed)
		}
		result = append(result, *device)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Index < result[j].Index })
	return result
}

func parsePrometheusSample(line string) (prometheusSample, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return prometheusSample{}, false
	}

	name := ""
	labelText := ""
	valueText := ""
	if brace := strings.Index(line, "{"); brace >= 0 {
		end := strings.Index(line[brace:], "}")
		if end < 0 {
			return prometheusSample{}, false
		}
		end += brace
		name = strings.TrimSpace(line[:brace])
		labelText = line[brace+1 : end]
		valueText = strings.TrimSpace(line[end+1:])
	} else {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			return prometheusSample{}, false
		}
		name = parts[0]
		valueText = parts[1]
	}

	valueParts := strings.Fields(valueText)
	if len(valueParts) == 0 {
		return prometheusSample{}, false
	}
	value, err := strconv.ParseFloat(valueParts[0], 64)
	if err != nil {
		return prometheusSample{}, false
	}

	return prometheusSample{name: name, labels: parsePrometheusLabels(labelText), value: value}, true
}

func parsePrometheusLabels(labelText string) map[string]string {
	labels := map[string]string{}
	for len(labelText) > 0 {
		labelText = strings.TrimLeft(labelText, " ,")
		if labelText == "" {
			break
		}
		eq := strings.Index(labelText, "=")
		if eq < 0 {
			break
		}
		key := strings.TrimSpace(labelText[:eq])
		labelText = strings.TrimLeft(labelText[eq+1:], " ")
		if !strings.HasPrefix(labelText, "\"") {
			break
		}
		value, rest, ok := readPrometheusLabelValue(labelText[1:])
		if !ok {
			break
		}
		labels[key] = value
		labelText = rest
	}
	return labels
}

func readPrometheusLabelValue(input string) (string, string, bool) {
	var builder strings.Builder
	escaped := false
	for i, r := range input {
		if escaped {
			builder.WriteRune(r)
			escaped = false
			continue
		}
		switch r {
		case '\\':
			escaped = true
		case '"':
			return builder.String(), input[i+1:], true
		default:
			builder.WriteRune(r)
		}
	}
	return "", "", false
}

func metricNPUIndex(labels map[string]string) (int, bool) {
	for _, key := range []string{"npuID", "npu_id", "logicID", "logic_id", "device_id", "id"} {
		if value := labels[key]; value != "" {
			index, err := strconv.Atoi(value)
			return index, err == nil
		}
	}
	return 0, false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

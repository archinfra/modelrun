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

const defaultNPUExporterEndpoint = "http://127.0.0.1:8082/metrics"
const alternateNPUExporterEndpoint = "http://127.0.0.1:9101/metrics"
const defaultNPUExporterImage = "swr.cn-south-1.myhuaweicloud.com/ascendhub/npu-exporter:v7.3.0"

func DefaultNPUExporterEndpoint() string {
	return defaultNPUExporterEndpoint
}

func DefaultNPUExporterImage() string {
	return defaultNPUExporterImage
}

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
	devices, _, err := collectNPUExporterWithEndpoint(client, configuredEndpoint)
	return devices, err
}

func (c *Collector) NPUExporterStatus(server domain.ServerConfig, jump *SSHConfig) (NPUExporterStatus, error) {
	endpoint := firstNPUExporterEndpoint(server.NPUExporterEndpoint)
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

	return npuExporterStatusFromClient(client, server.NPUExporterEndpoint), nil
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
	return firstNPUExporterEndpoint(configured)
}

func firstNPUExporterEndpoint(configured string) string {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimSpace(configured)
	}
	if env := strings.TrimSpace(os.Getenv("MODELRUN_NPU_EXPORTER_ENDPOINT")); env != "" {
		return env
	}
	return defaultNPUExporterEndpoint
}

func npuExporterEndpoints(configured string) []string {
	candidates := []string{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range candidates {
			if existing == value {
				return
			}
		}
		candidates = append(candidates, value)
	}

	if strings.TrimSpace(configured) != "" {
		add(configured)
	}

	if env := strings.TrimSpace(os.Getenv("MODELRUN_NPU_EXPORTER_ENDPOINT")); env != "" {
		add(env)
	}
	add(defaultNPUExporterEndpoint)
	add(alternateNPUExporterEndpoint)
	return candidates
}

func fetchNPUExporterMetrics(client sshRunner, endpoint string) (string, error) {
	command := "if command -v curl >/dev/null 2>&1; then " +
		"curl -fsS --max-time 3 " + shellQuote(endpoint) + "; " +
		"elif command -v wget >/dev/null 2>&1; then " +
		"wget -q -T 3 -O - " + shellQuote(endpoint) + "; " +
		"else echo 'curl or wget is required to scrape npu exporter' >&2; exit 127; fi"
	return run(client, command)
}

func collectNPUExporterWithEndpoint(client sshRunner, configured string) ([]domain.GPUInfo, string, error) {
	var lastErr error
	for _, endpoint := range npuExporterEndpoints(configured) {
		out, err := fetchNPUExporterMetrics(client, endpoint)
		if err != nil {
			lastErr = err
			continue
		}
		accelerators := parseNPUExporterMetrics(out)
		if len(accelerators) == 0 {
			lastErr = errors.New("npu exporter metrics did not contain npu devices")
			continue
		}
		return accelerators, endpoint, nil
	}
	if lastErr == nil {
		lastErr = errors.New("npu exporter metrics did not contain npu devices")
	}
	return nil, firstNPUExporterEndpoint(configured), lastErr
}

func npuExporterStatusFromClient(client sshRunner, configured string) NPUExporterStatus {
	status := NPUExporterStatus{Endpoint: firstNPUExporterEndpoint(configured)}
	accelerators, endpoint, err := collectNPUExporterWithEndpoint(client, configured)
	if err != nil {
		for _, probeEndpoint := range npuExporterEndpoints(configured) {
			out, fetchErr := fetchNPUExporterMetrics(client, probeEndpoint)
			if fetchErr != nil {
				continue
			}
			status.Endpoint = probeEndpoint
			status.Reachable = true
			status.Message = explainNPUExporterParseFailure(out)
			return status
		}
		status.Message = err.Error()
		return status
	}

	status.Endpoint = endpoint
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
			port = 8082
		}
		if strings.TrimSpace(opts.Endpoint) == "" {
			endpoint = fmt.Sprintf("http://127.0.0.1:%d/metrics", port)
		}
		command := withDockerPrivilegesCommand(
			"(run_docker rm -f modelrun-npu-exporter >/dev/null 2>&1 || true) && " +
				"run_docker run -d --name modelrun-npu-exporter --restart unless-stopped --network host --privileged --entrypoint npu-exporter " +
				"-v /dev:/dev " +
				"-v /usr/local/Ascend:/usr/local/Ascend:ro " +
				"-v /usr/local/dcmi:/usr/local/dcmi:ro " +
				"-v /sys:/sys:ro " +
				"-v /tmp:/tmp " +
				"-v /var/run/docker.sock:/var/run/docker.sock " +
				"-v /etc/localtime:/etc/localtime:ro " +
				shellQuote(opts.Image) + " " +
				"-ip=" + shellQuote("0.0.0.0") + " " +
				"-port=" + shellQuote(strconv.Itoa(port)) + " " +
				"-containerMode=docker",
		)
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

type exporterDeviceState struct {
	info        domain.GPUInfo
	hasHBMTotal bool
	hasHBMUsed  bool
	hasHBMFree  bool
}

func parseNPUExporterMetrics(out string) []domain.GPUInfo {
	devices := map[int]*exporterDeviceState{}

	for _, line := range strings.Split(out, "\n") {
		sample, ok := parsePrometheusSample(line)
		if !ok || !looksLikeNPUMetric(sample.name) {
			continue
		}
		index, ok := metricNPUIndex(sample.labels)
		if !ok {
			continue
		}

		device := devices[index]
		if device == nil {
			device = &exporterDeviceState{
				info: domain.GPUInfo{Index: index, Type: "npu", Name: "Ascend NPU"},
			}
			devices[index] = device
		}
		if name := metricNPUName(sample.labels); name != "" {
			device.info.Name = normalizeNPUName(name)
		}

		switch sample.name {
		case "npu_chip_info_name":
			if sample.value > 0 {
				if name := metricNPUName(sample.labels); name != "" {
					device.info.Name = normalizeNPUName(name)
				}
			}
		case "npu_chip_info_utilization", "npu_chip_info_overall_utilization", "npu_chip_info_aicore_utilization", "npu_chip_info_core_utilization":
			device.info.Utilization = sample.value
		case "npu_chip_info_temperature", "npu_chip_info_chip_temperature", "npu_chip_info_temp":
			device.info.Temperature = sample.value
		case "npu_chip_info_power", "npu_chip_info_power_usage", "npu_chip_info_power_draw":
			device.info.PowerDraw = sample.value
		case "npu_chip_info_hbm_total_memory":
			device.info.MemoryTotal = normalizeAcceleratorMemory(sample.value)
			device.hasHBMTotal = true
		case "npu_chip_info_hbm_used_memory":
			device.info.MemoryUsed = normalizeAcceleratorMemory(sample.value)
			device.hasHBMUsed = true
		case "npu_chip_info_hbm_free_memory":
			device.info.MemoryFree = normalizeAcceleratorMemory(sample.value)
			device.hasHBMFree = true
		case "npu_chip_info_total_memory":
			if !device.hasHBMTotal && (device.info.MemoryTotal == 0 || sample.value > 0) {
				device.info.MemoryTotal = normalizeAcceleratorMemory(sample.value)
			}
		case "npu_chip_info_used_memory":
			if !device.hasHBMUsed && (device.info.MemoryUsed == 0 || sample.value > 0) {
				device.info.MemoryUsed = normalizeAcceleratorMemory(sample.value)
			}
		case "npu_chip_info_free_memory":
			if !device.hasHBMFree && (device.info.MemoryFree == 0 || sample.value > 0) {
				device.info.MemoryFree = normalizeAcceleratorMemory(sample.value)
			}
		case "npu_chip_info_health_status", "npu_chip_info_health":
			if sample.value == 1 {
				device.info.Health = "OK"
			} else {
				device.info.Health = "UNHEALTHY"
			}
		}
	}

	result := make([]domain.GPUInfo, 0, len(devices))
	for _, device := range devices {
		if device.info.MemoryTotal > 0 && device.info.MemoryFree == 0 {
			device.info.MemoryFree = maxInt64(0, device.info.MemoryTotal-device.info.MemoryUsed)
		}
		result = append(result, device.info)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Index < result[j].Index })
	return result
}

func normalizeAcceleratorMemory(value float64) int64 {
	const (
		maxReasonableMiB = 1_000_000
		minBytesValue    = 1 << 30
	)

	switch {
	case value >= minBytesValue:
		return int64(value / (1024 * 1024))
	case value >= maxReasonableMiB:
		return int64(value / 1024)
	default:
		return int64(value)
	}
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
	for _, key := range []string{"npuID", "npu_id", "logicID", "logic_id", "device_id", "deviceId", "chip_id", "chipID", "id"} {
		if value := labels[key]; value != "" {
			index, err := strconv.Atoi(value)
			return index, err == nil
		}
	}
	return 0, false
}

func looksLikeNPUMetric(name string) bool {
	return strings.HasPrefix(name, "npu_chip_info_") || strings.HasPrefix(name, "npu_chip_")
}

func metricNPUName(labels map[string]string) string {
	return firstNonEmpty(
		labels["name"],
		labels["modelName"],
		labels["model_name"],
		labels["product_name"],
		labels["chip_type"],
	)
}

func explainNPUExporterParseFailure(out string) string {
	families := collectMetricFamilies(out, 6)
	if len(families) == 0 {
		return "npu exporter endpoint is reachable, but returned no recognizable Prometheus metrics"
	}
	return fmt.Sprintf("npu exporter endpoint is reachable, but ModelRun could not map the current metric format to devices. detected metric families: %s", strings.Join(families, ", "))
}

func collectMetricFamilies(out string, limit int) []string {
	seen := map[string]struct{}{}
	families := make([]string, 0, limit)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		nameEnd := strings.IndexAny(line, "{ ")
		if nameEnd <= 0 {
			continue
		}
		name := line[:nameEnd]
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		families = append(families, name)
		if len(families) >= limit {
			break
		}
	}
	sort.Strings(families)
	return families
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
		"sudo -n true >/dev/null 2>&1 || { echo 'docker command requires sudo privileges for the current SSH user, or the user must be added to the docker group.' >&2; return 1; };",
		"sudo -n docker \"$@\";",
		"return $?;",
		"fi;",
		"echo 'docker command requires sudo privileges because the current SSH user is not root and sudo is unavailable.' >&2;",
		"return 1;",
		"};",
		body,
	}, " ")
}

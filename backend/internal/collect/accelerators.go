package collect

import (
	"strconv"
	"strings"

	"modelrun/backend/internal/domain"

	"golang.org/x/crypto/ssh"
)

func collectAccelerators(client *ssh.Client, npuExporterEndpoint string) ([]domain.GPUInfo, string, string, error) {
	accelerators := []domain.GPUInfo{}
	driverVersion := ""
	cudaVersion := ""

	if hasCommand(client, "nvidia-smi") {
		gpus, driver, cuda, err := collectNVIDIA(client)
		if err != nil {
			return nil, "", "", err
		}
		accelerators = append(accelerators, gpus...)
		driverVersion = driver
		cudaVersion = cuda
	}

	if npus, err := collectNPUExporter(client, npuExporterEndpoint); err == nil && len(npus) > 0 {
		accelerators = append(accelerators, npus...)
	} else if hasCommand(client, "npu-smi") {
		npus, err := collectAscendNPU(client)
		if err != nil {
			return nil, "", "", err
		}
		accelerators = append(accelerators, npus...)
	}

	return accelerators, driverVersion, cudaVersion, nil
}

func collectNVIDIA(client *ssh.Client) ([]domain.GPUInfo, string, string, error) {
	query := "nvidia-smi --query-gpu=index,name,memory.total,memory.used,memory.free,utilization.gpu,temperature.gpu,power.draw,power.limit --format=csv,noheader,nounits"
	out, err := run(client, query)
	if err != nil {
		return nil, "", "", err
	}

	gpus := []domain.GPUInfo{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := splitCSVLine(line)
		if len(parts) < 9 {
			continue
		}
		gpus = append(gpus, domain.GPUInfo{
			Index:       parseInt(parts[0]),
			Type:        "gpu",
			Name:        parts[1],
			MemoryTotal: parseInt64(parts[2]),
			MemoryUsed:  parseInt64(parts[3]),
			MemoryFree:  parseInt64(parts[4]),
			Utilization: parseFloat(parts[5]),
			Temperature: parseFloat(parts[6]),
			PowerDraw:   parseFloat(parts[7]),
			PowerLimit:  parseFloat(parts[8]),
			Health:      "OK",
		})
	}

	driver, _ := run(client, "nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits | head -n 1")
	cuda, _ := run(client, "nvidia-smi | sed -n 's/.*CUDA Version: *\\([^ |]*\\).*/\\1/p' | head -n 1")

	return gpus, strings.TrimSpace(driver), strings.TrimSpace(cuda), nil
}

func collectAscendNPU(client *ssh.Client) ([]domain.GPUInfo, error) {
	out, err := run(client, "npu-smi info")
	if err != nil {
		return nil, err
	}
	return parseAscendNPUInfo(out), nil
}

func parseAscendNPUInfo(out string) []domain.GPUInfo {
	lines := strings.Split(out, "\n")
	devices := []domain.GPUInfo{}

	for i := 0; i < len(lines); i++ {
		fields := tableFields(lines[i])
		if len(fields) < 3 {
			continue
		}
		left := strings.Fields(fields[0])
		if len(left) < 2 || !isInteger(left[0]) {
			continue
		}
		health := firstWord(fields[1])
		if isInteger(left[1]) || !looksLikeNPUHealth(health) {
			continue
		}

		device := domain.GPUInfo{
			Index:  parseInt(left[0]),
			Type:   "npu",
			Name:   normalizeNPUName(left[1]),
			Health: health,
		}

		powerTemp := strings.Fields(fields[2])
		if len(powerTemp) > 0 {
			device.PowerDraw = parseFloat(powerTemp[0])
		}
		if len(powerTemp) > 1 {
			device.Temperature = parseFloat(powerTemp[1])
		}

		if i+1 < len(lines) {
			nextFields := tableFields(lines[i+1])
			if len(nextFields) >= 3 {
				nextLeft := strings.Fields(nextFields[0])
				if len(nextLeft) > 0 && isInteger(nextLeft[0]) {
					device.ChipID = parseInt(nextLeft[0])
				}
				if len(nextLeft) > 1 && isInteger(nextLeft[1]) {
					device.LogicID = parseInt(nextLeft[1])
				}
				parseNPUUtilAndMemory(nextFields[2], &device)
			}
		}

		if device.MemoryTotal > 0 && device.MemoryFree == 0 {
			device.MemoryFree = maxInt64(0, device.MemoryTotal-device.MemoryUsed)
		}

		devices = append(devices, device)
	}

	return devices
}

func parseNPUUtilAndMemory(value string, device *domain.GPUInfo) {
	tokens := strings.Fields(value)
	if len(tokens) > 0 {
		device.Utilization = parseFloat(tokens[0])
	}
	for i, token := range tokens {
		if token == "/" && i > 0 && i+1 < len(tokens) {
			device.MemoryUsed = parseInt64(tokens[i-1])
			device.MemoryTotal = parseInt64(tokens[i+1])
			return
		}
		if strings.Contains(token, "/") {
			pair := strings.Split(token, "/")
			if len(pair) == 2 {
				device.MemoryUsed = parseInt64(pair[0])
				device.MemoryTotal = parseInt64(pair[1])
				return
			}
		}
	}
}

func tableFields(line string) []string {
	raw := strings.Split(line, "|")
	fields := []string{}
	for _, value := range raw {
		value = strings.TrimSpace(value)
		if value != "" {
			fields = append(fields, value)
		}
	}
	return fields
}

func splitCSVLine(line string) []string {
	raw := strings.Split(line, ",")
	parts := make([]string, 0, len(raw))
	for _, value := range raw {
		parts = append(parts, strings.TrimSpace(value))
	}
	return parts
}

func normalizeNPUName(name string) string {
	if strings.Contains(strings.ToLower(name), "ascend") {
		return name
	}
	return "Ascend " + name
}

func firstWord(value string) string {
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func looksLikeNPUHealth(value string) bool {
	switch strings.ToUpper(value) {
	case "OK", "NORMAL", "WARNING", "WARN", "ALARM", "FAULT", "ABNORMAL":
		return true
	default:
		return false
	}
}

func isInteger(value string) bool {
	_, err := strconv.Atoi(value)
	return err == nil
}

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(cleanNumber(value))
	return parsed
}

func parseInt64(value string) int64 {
	parsed, _ := strconv.ParseInt(cleanNumber(value), 10, 64)
	return parsed
}

func parseFloat(value string) float64 {
	parsed, _ := strconv.ParseFloat(cleanFloat(value), 64)
	return parsed
}

func cleanNumber(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "MiB")
	value = strings.TrimSuffix(value, "MB")
	value = strings.TrimSuffix(value, "W")
	value = strings.TrimSuffix(value, "C")
	value = strings.TrimSpace(value)
	if value == "N/A" || value == "-" {
		return "0"
	}
	if idx := strings.Index(value, "."); idx >= 0 {
		value = value[:idx]
	}
	return value
}

func cleanFloat(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimSuffix(value, "MiB")
	value = strings.TrimSuffix(value, "MB")
	value = strings.TrimSuffix(value, "W")
	value = strings.TrimSuffix(value, "C")
	value = strings.TrimSuffix(value, "%")
	value = strings.TrimSpace(value)
	if value == "N/A" || value == "-" {
		return "0"
	}
	return value
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

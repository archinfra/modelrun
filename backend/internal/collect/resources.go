package collect

import (
	"math"
	"strings"

	"modelrun/backend/internal/domain"

	"golang.org/x/crypto/ssh"
)

func collectResources(client *ssh.Client) (domain.ServerResource, error) {
	var resource domain.ServerResource

	cores, _ := run(client, "getconf _NPROCESSORS_ONLN 2>/dev/null || nproc 2>/dev/null || echo 0")
	resource.CPU.Cores = parseInt(strings.TrimSpace(cores))
	resource.CPU.Usage = collectCPUUsage(client)

	memory, err := run(client, "awk '/MemTotal/ {total=int($2/1024)} /MemAvailable/ {free=int($2/1024)} END {print total, total-free, free}' /proc/meminfo")
	if err != nil {
		return resource, err
	}
	memoryParts := strings.Fields(memory)
	if len(memoryParts) >= 3 {
		resource.Memory.Total = parseInt64(memoryParts[0])
		resource.Memory.Used = parseInt64(memoryParts[1])
		resource.Memory.Free = parseInt64(memoryParts[2])
	}

	disk, err := run(client, "df -Pm / | awk 'NR==2 {print $2, $3, $4}'")
	if err != nil {
		return resource, err
	}
	diskParts := strings.Fields(disk)
	if len(diskParts) >= 3 {
		resource.Disk.Total = parseInt64(diskParts[0])
		resource.Disk.Used = parseInt64(diskParts[1])
		resource.Disk.Free = parseInt64(diskParts[2])
	}

	resource.Network.RXSpeed = 0
	resource.Network.TXSpeed = 0

	return resource, nil
}

func collectCPUUsage(client *ssh.Client) float64 {
	command := "awk '/^cpu / {total=0; for (i=2;i<=NF;i++) total+=$i; print total, $5}' /proc/stat; sleep 1; awk '/^cpu / {total=0; for (i=2;i<=NF;i++) total+=$i; print total, $5}' /proc/stat"
	out, err := run(client, command)
	if err != nil {
		return 0
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return 0
	}
	first := strings.Fields(lines[0])
	second := strings.Fields(lines[1])
	if len(first) < 2 || len(second) < 2 {
		return 0
	}

	total1 := parseFloat(first[0])
	idle1 := parseFloat(first[1])
	total2 := parseFloat(second[0])
	idle2 := parseFloat(second[1])
	totalDelta := total2 - total1
	if totalDelta <= 0 {
		return 0
	}

	usage := (1 - ((idle2 - idle1) / totalDelta)) * 100
	return math.Round(usage*100) / 100
}

func collectDockerVersion(client *ssh.Client) string {
	out, err := run(client, "docker --version 2>/dev/null | awk '{print $3}' | tr -d ','")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

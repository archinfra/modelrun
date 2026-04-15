package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/store"
)

func (a *API) handleServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		servers := data.Servers
		if projectID := r.URL.Query().Get("projectId"); projectID != "" {
			servers = make([]domain.ServerConfig, 0, len(data.Servers))
			for _, server := range data.Servers {
				if server.ProjectID == projectID {
					servers = append(servers, server)
				}
			}
		}
		writeJSON(w, http.StatusOK, servers)
	case http.MethodPost:
		var server domain.ServerConfig
		if err := readJSON(r, &server); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		defaultServer(&server, r.URL.Query().Get("projectId"))

		if err := a.store.Update(func(data *domain.Data) error {
			data.Servers = append(data.Servers, server)
			if server.ProjectID != "" {
				idx := findProject(data.Projects, server.ProjectID)
				if idx >= 0 && !contains(data.Projects[idx].ServerIDs, server.ID) {
					data.Projects[idx].ServerIDs = append(data.Projects[idx].ServerIDs, server.ID)
					data.Projects[idx].UpdatedAt = domain.Now()
				}
			}
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusCreated, server)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleServer(w http.ResponseWriter, r *http.Request) {
	id, rest := pathParts(r.URL.Path, "/api/servers/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	if len(rest) == 0 {
		a.handleServerItem(w, r, id)
		return
	}
	if len(rest) != 1 {
		http.NotFound(w, r)
		return
	}

	switch rest[0] {
	case "test":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		a.handleServerTest(w, id)
	case "resources":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		a.handleServerResources(w, id)
	case "gpu":
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		a.handleServerGPU(w, id)
	default:
		http.NotFound(w, r)
	}
}

func (a *API) handleServerItem(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		idx := findServer(data.Servers, id)
		if idx < 0 {
			writeError(w, http.StatusNotFound, store.ErrNotFound)
			return
		}
		writeJSON(w, http.StatusOK, data.Servers[idx])
	case http.MethodPut, http.MethodPatch:
		var patch map[string]json.RawMessage
		if err := readJSON(r, &patch); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		var server domain.ServerConfig
		err := a.store.Update(func(data *domain.Data) error {
			idx := findServer(data.Servers, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			updated, err := mergeJSON(data.Servers[idx], patch)
			if err != nil {
				return err
			}
			updated.ID = data.Servers[idx].ID
			defaultServer(&updated, "")
			data.Servers[idx] = updated
			server = updated
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, server)
	case http.MethodDelete:
		err := a.store.Update(func(data *domain.Data) error {
			idx := findServer(data.Servers, id)
			if idx < 0 {
				return store.ErrNotFound
			}
			projectID := data.Servers[idx].ProjectID
			data.Servers = append(data.Servers[:idx], data.Servers[idx+1:]...)
			if projectID != "" {
				pidx := findProject(data.Projects, projectID)
				if pidx >= 0 {
					data.Projects[pidx].ServerIDs = removeString(data.Projects[pidx].ServerIDs, id)
					data.Projects[pidx].UpdatedAt = domain.Now()
				}
			}
			return nil
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleServerTest(w http.ResponseWriter, id string) {
	data := a.store.Snapshot()
	idx := findServer(data.Servers, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	server := data.Servers[idx]
	success, message := probeSSH(server)
	status := "offline"
	if success {
		status = "online"
	}

	if err := a.store.Update(func(data *domain.Data) error {
		idx := findServer(data.Servers, id)
		if idx < 0 {
			return store.ErrNotFound
		}
		data.Servers[idx].Status = status
		data.Servers[idx].LastCheck = domain.Now()
		if success {
			if len(data.Servers[idx].GPUInfo) == 0 {
				data.Servers[idx].GPUInfo = mockGPUs(id)
			}
			if data.Servers[idx].DriverVersion == "" {
				data.Servers[idx].DriverVersion = "535.104.05"
			}
			if data.Servers[idx].CUDAVersion == "" {
				data.Servers[idx].CUDAVersion = "12.2"
			}
			if data.Servers[idx].DockerVersion == "" {
				data.Servers[idx].DockerVersion = "24.0.7"
			}
			server = data.Servers[idx]
		}
		return nil
	}); err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       success,
		"message":       message,
		"gpuInfo":       server.GPUInfo,
		"driverVersion": server.DriverVersion,
		"cudaVersion":   server.CUDAVersion,
	})
}

func (a *API) handleServerResources(w http.ResponseWriter, id string) {
	data := a.store.Snapshot()
	idx := findServer(data.Servers, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	writeJSON(w, http.StatusOK, makeResource(data.Servers[idx]))
}

func (a *API) handleServerGPU(w http.ResponseWriter, id string) {
	data := a.store.Snapshot()
	idx := findServer(data.Servers, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	gpus := data.Servers[idx].GPUInfo
	if len(gpus) == 0 {
		gpus = mockGPUs(id)
	}
	writeJSON(w, http.StatusOK, gpus)
}

func probeSSH(server domain.ServerConfig) (bool, string) {
	if server.Host == "" {
		return false, "server host is empty"
	}

	port := server.SSHPort
	if port == 0 {
		port = 22
	}

	if os.Getenv("MODELRUN_FAKE_CONNECT") == "1" || strings.HasPrefix(server.Host, "mock") {
		return true, "mock connection succeeded"
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(server.Host, strconv.Itoa(port)), 1500*time.Millisecond)
	if err != nil {
		return false, fmt.Sprintf("tcp connection failed: %v", err)
	}
	_ = conn.Close()

	return true, "ssh port is reachable"
}

func makeResource(server domain.ServerConfig) domain.ServerResource {
	gpus := server.GPUInfo
	if len(gpus) == 0 {
		gpus = mockGPUs(server.ID)
	}

	var resource domain.ServerResource
	resource.CPU.Cores = maxInt(8, len(gpus)*16)
	resource.CPU.Usage = 22.5
	resource.Memory.Total = int64(resource.CPU.Cores) * 4096
	resource.Memory.Used = resource.Memory.Total / 3
	resource.Memory.Free = resource.Memory.Total - resource.Memory.Used
	resource.Disk.Total = 4 * 1024 * 1024
	resource.Disk.Used = resource.Disk.Total / 2
	resource.Disk.Free = resource.Disk.Total - resource.Disk.Used
	resource.Network.RXSpeed = 12.5
	resource.Network.TXSpeed = 8.2
	return resource
}

func mockGPUs(seed string) []domain.GPUInfo {
	if len(seed)%2 == 0 {
		return []domain.GPUInfo{
			{Index: 0, Name: "NVIDIA A100 80GB", MemoryTotal: 81920, MemoryUsed: 24576, MemoryFree: 57344, Utilization: 45, Temperature: 72, PowerDraw: 285, PowerLimit: 400},
			{Index: 1, Name: "NVIDIA A100 80GB", MemoryTotal: 81920, MemoryUsed: 18432, MemoryFree: 63488, Utilization: 32, Temperature: 68, PowerDraw: 240, PowerLimit: 400},
		}
	}
	return []domain.GPUInfo{
		{Index: 0, Name: "NVIDIA A100 40GB", MemoryTotal: 40960, MemoryUsed: 8192, MemoryFree: 32768, Utilization: 28, Temperature: 65, PowerDraw: 180, PowerLimit: 250},
	}
}

func findServer(servers []domain.ServerConfig, id string) int {
	for i, server := range servers {
		if server.ID == id {
			return i
		}
	}
	return -1
}

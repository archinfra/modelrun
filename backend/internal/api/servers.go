package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"modelrun/backend/internal/collect"
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
		writeJSON(w, http.StatusOK, a.state.OverlayServers(servers))
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
	if rest[0] == "npu-exporter" {
		a.handleServerNPUExporter(w, r, id, rest[1:])
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

func (a *API) handleServerNPUExporter(w http.ResponseWriter, r *http.Request, id string, rest []string) {
	if len(rest) == 0 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		a.handleServerNPUExporterStatus(w, id)
		return
	}
	if len(rest) == 1 && rest[0] == "install" {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		a.handleServerNPUExporterInstall(w, r, id)
		return
	}
	http.NotFound(w, r)
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
		writeJSON(w, http.StatusOK, a.state.OverlayServer(data.Servers[idx]))
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
		a.state.SetServerRuntime(id, server)

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
		a.state.DeleteServerRuntime(id)
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
	jump, err := resolveJumpHost(data, server)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	snapshot, err := a.collector.Collect(server, jump)
	success := err == nil
	message := snapshot.Message
	var exporterStatus *collect.NPUExporterStatus
	if err != nil {
		message = err.Error()
	} else if status, statusErr := a.collector.NPUExporterStatus(server, jump); statusErr == nil {
		exporterStatus = &status
	}

	if success {
		server.Status = "online"
		server.GPUInfo = snapshot.Accelerators
		server.DriverVersion = snapshot.DriverVersion
		server.CUDAVersion = snapshot.CUDAVersion
		server.DockerVersion = snapshot.DockerVersion
		if exporterStatus != nil {
			server.NPUExporterEndpoint = exporterStatus.Endpoint
			if exporterStatus.Reachable {
				server.NPUExporterStatus = "online"
			} else {
				server.NPUExporterStatus = "offline"
			}
			server.NPUExporterLastCheck = domain.Now()
		}
	} else {
		server.Status = "offline"
	}
	server.LastCheck = domain.Now()
	a.state.SetServerRuntime(id, server)

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       success,
		"message":       message,
		"gpuInfo":       server.GPUInfo,
		"driverVersion": server.DriverVersion,
		"cudaVersion":   server.CUDAVersion,
		"dockerVersion": server.DockerVersion,
		"resources":     snapshot.Resources,
		"server":        server,
	})
}

func (a *API) handleServerResources(w http.ResponseWriter, id string) {
	data := a.store.Snapshot()
	idx := findServer(data.Servers, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	jump, err := resolveJumpHost(data, data.Servers[idx])
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	resource, err := a.collector.Resources(data.Servers[idx], jump)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, resource)
}

func (a *API) handleServerGPU(w http.ResponseWriter, id string) {
	data := a.store.Snapshot()
	idx := findServer(data.Servers, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	jump, err := resolveJumpHost(data, data.Servers[idx])
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	gpus, err := a.collector.Accelerators(data.Servers[idx], jump)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}
	server := data.Servers[idx]
	server.GPUInfo = gpus
	server.LastCheck = domain.Now()
	a.state.SetServerRuntime(id, server)

	writeJSON(w, http.StatusOK, gpus)
}

func (a *API) handleServerNPUExporterStatus(w http.ResponseWriter, id string) {
	data := a.store.Snapshot()
	idx := findServer(data.Servers, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	server := data.Servers[idx]
	jump, err := resolveJumpHost(data, server)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	status, err := a.collector.NPUExporterStatus(server, jump)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	server.NPUExporterEndpoint = status.Endpoint
	if status.Reachable {
		server.NPUExporterStatus = "online"
	} else {
		server.NPUExporterStatus = "offline"
	}
	server.NPUExporterLastCheck = domain.Now()
	a.state.SetServerRuntime(id, server)

	writeJSON(w, http.StatusOK, status)
}

func (a *API) handleServerNPUExporterInstall(w http.ResponseWriter, r *http.Request, id string) {
	var opts collect.NPUExporterInstallOptions
	if err := readJSON(r, &opts); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	data := a.store.Snapshot()
	idx := findServer(data.Servers, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	server := data.Servers[idx]
	jump, err := resolveJumpHost(data, server)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := a.collector.InstallNPUExporter(server, jump, opts)
	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	server.NPUExporterEndpoint = result.Endpoint
	if result.Status.Reachable {
		server.NPUExporterStatus = "online"
	} else {
		server.NPUExporterStatus = "offline"
	}
	server.NPUExporterLastCheck = domain.Now()
	a.state.SetServerRuntime(id, server)

	writeJSON(w, http.StatusOK, result)
}

func findServer(servers []domain.ServerConfig, id string) int {
	for i, server := range servers {
		if server.ID == id {
			return i
		}
	}
	return -1
}

func resolveJumpHost(data domain.Data, server domain.ServerConfig) (*collect.SSHConfig, error) {
	if collect.IsMockServer(server) || !server.UseJumpHost {
		return nil, nil
	}
	if server.JumpHostID == "" {
		return nil, errors.New("jumpHostId is required when useJumpHost is true")
	}
	if server.JumpHostID == server.ID {
		return nil, errors.New("server cannot use itself as jump host")
	}

	for _, candidate := range data.Servers {
		if candidate.ID == server.JumpHostID {
			config := collect.FromServer(candidate)
			return &config, nil
		}
	}
	for _, candidate := range data.JumpHosts {
		if candidate.ID == server.JumpHostID {
			config := collect.FromJumpHost(candidate)
			return &config, nil
		}
	}

	return nil, errors.New("jump host not found")
}

package api

import (
	"net/http"
	"sort"

	"modelrun/backend/internal/dispatch"
	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/store"
)

func (a *API) handleRemoteTaskPresets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, a.dispatcher.Presets())
}

func (a *API) handleRemoteTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data := a.store.Snapshot()
		filtered := make([]domain.RemoteTask, 0, len(data.RemoteTasks))
		filtered = append(filtered, data.RemoteTasks...)
		if projectID := r.URL.Query().Get("projectId"); projectID != "" {
			filtered = make([]domain.RemoteTask, 0, len(data.RemoteTasks))
			for _, task := range data.RemoteTasks {
				if task.ProjectID == projectID {
					filtered = append(filtered, task)
				}
			}
		}
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].CreatedAt > filtered[j].CreatedAt
		})
		writeJSON(w, http.StatusOK, filtered)
	case http.MethodPost:
		var req dispatch.CreateTaskRequest
		if err := readJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		task, err := a.dispatcher.Create(req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, task)
	default:
		methodNotAllowed(w)
	}
}

func (a *API) handleRemoteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	id, rest := pathParts(r.URL.Path, "/api/remote-tasks/")
	if id == "" || len(rest) != 0 {
		http.NotFound(w, r)
		return
	}

	data := a.store.Snapshot()
	for _, task := range data.RemoteTasks {
		if task.ID == id {
			writeJSON(w, http.StatusOK, task)
			return
		}
	}

	writeError(w, http.StatusNotFound, store.ErrNotFound)
}

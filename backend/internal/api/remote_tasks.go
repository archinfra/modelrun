package api

import (
	"net/http"
	"sort"

	"modelrun/backend/internal/dispatch"
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
		filtered := a.state.RemoteTasks(r.URL.Query().Get("projectId"))
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

	task, ok := a.state.RemoteTask(id)
	if ok {
		writeJSON(w, http.StatusOK, task)
		return
	}

	writeError(w, http.StatusNotFound, store.ErrNotFound)
}

package api

import (
	"net/http"

	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/store"
)

func (a *API) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	tasks := []domain.DeploymentTask{}
	if deploymentID := r.URL.Query().Get("deploymentId"); deploymentID != "" {
		tasks = a.state.Tasks(deploymentID)
	}
	tasks = a.executor.HydrateTasks(tasks)

	writeJSON(w, http.StatusOK, tasks)
}

func (a *API) handleTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	id, rest := pathParts(r.URL.Path, "/api/tasks/")
	if id == "" || len(rest) != 0 {
		http.NotFound(w, r)
		return
	}

	task, ok := a.state.TaskByID(id)
	if !ok {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	writeJSON(w, http.StatusOK, a.executor.HydrateTasks([]domain.DeploymentTask{task})[0])
}

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

	data := a.store.Snapshot()
	tasks := data.Tasks
	if deploymentID := r.URL.Query().Get("deploymentId"); deploymentID != "" {
		tasks = make([]domain.DeploymentTask, 0, len(data.Tasks))
		for _, task := range data.Tasks {
			if task.DeploymentID == deploymentID {
				tasks = append(tasks, task)
			}
		}
	}

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

	data := a.store.Snapshot()
	idx := findTaskByID(data.Tasks, id)
	if idx < 0 {
		writeError(w, http.StatusNotFound, store.ErrNotFound)
		return
	}

	writeJSON(w, http.StatusOK, data.Tasks[idx])
}

func findTaskByID(tasks []domain.DeploymentTask, id string) int {
	for i, task := range tasks {
		if task.ID == id {
			return i
		}
	}
	return -1
}

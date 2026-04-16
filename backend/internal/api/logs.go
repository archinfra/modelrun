package api

import (
	"net/http"
	"strconv"

	"modelrun/backend/internal/domain"
)

func (a *API) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	data := a.store.Snapshot()
	items := make([]domain.DeploymentLog, 0, len(data.Logs))

	deploymentID := r.URL.Query().Get("deploymentId")
	serverID := r.URL.Query().Get("serverId")
	stepID := r.URL.Query().Get("stepId")

	for _, item := range data.Logs {
		if deploymentID != "" && item.DeploymentID != deploymentID {
			continue
		}
		if serverID != "" && item.ServerID != serverID {
			continue
		}
		if stepID != "" && item.StepID != stepID {
			continue
		}
		items = append(items, item)
	}

	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	if len(items) > limit {
		items = items[len(items)-limit:]
	}

	writeJSON(w, http.StatusOK, items)
}

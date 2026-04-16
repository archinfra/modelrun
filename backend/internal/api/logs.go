package api

import (
	"net/http"
	"strconv"
)

func (a *API) handleLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	deploymentID := r.URL.Query().Get("deploymentId")
	serverID := r.URL.Query().Get("serverId")
	stepID := r.URL.Query().Get("stepId")

	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}
	writeJSON(w, http.StatusOK, a.state.Logs(deploymentID, serverID, stepID, limit))
}

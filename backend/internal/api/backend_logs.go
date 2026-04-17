package api

import (
	"net/http"
	"strconv"

	"modelrun/backend/internal/logging"
)

func (a *API) handleBackendLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	limit := 200
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil && value > 0 {
			limit = value
		}
	}

	writeJSON(w, http.StatusOK, logging.Default().Tail(limit))
}

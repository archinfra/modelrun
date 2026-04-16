package api

import (
	"net/http"

	"modelrun/backend/internal/deploy"
)

func (a *API) handlePipelineTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, deploy.PipelineTemplates())
}

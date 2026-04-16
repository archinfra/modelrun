package api

import (
	"net/http"
	"os"
)

func (a *API) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	data := a.store.Snapshot()

	writeJSON(w, http.StatusOK, map[string]any{
		"storage": map[string]any{
			"driver": "sqlite",
			"path":   a.store.Path(),
			"counts": map[string]int{
				"projects":         len(data.Projects),
				"servers":          len(data.Servers),
				"jumpHosts":        len(data.JumpHosts),
				"models":           len(data.Models),
				"deployments":      len(data.Deployments),
				"tasks":            len(data.Tasks),
				"remoteTasks":      len(data.RemoteTasks),
				"actionTemplates":  len(data.ActionTemplates),
				"bootstrapConfigs": len(data.BootstrapConfigs),
				"pipelineSteps":    len(data.PipelineSteps),
				"logs":             len(data.Logs),
			},
			"persistedCollections": []string{
				"projects",
				"servers",
				"jumpHosts",
				"models",
				"deployments",
				"tasks",
				"remoteTasks",
				"actionTemplates",
				"bootstrapConfigs",
				"pipeline_steps",
				"logs",
			},
		},
		"mock": map[string]any{
			"fakeConnectEnabled":   os.Getenv("MODELRUN_FAKE_CONNECT") == "1",
			"toggleEnv":            "MODELRUN_FAKE_CONNECT=1",
			"legacyHostPrefixMock": false,
			"description":          "SSH and metrics collection will only return mock data when MODELRUN_FAKE_CONNECT=1 is explicitly enabled.",
		},
		"demoFeatures": []map[string]any{
			{
				"key":         "model_search_catalog",
				"name":        "Model search catalog",
				"enabled":     true,
				"persisted":   false,
				"description": "GET /api/models/search still returns a built-in demo catalog. Created models are persisted in SQLite.",
			},
		},
	})
}

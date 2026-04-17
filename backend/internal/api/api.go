package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"modelrun/backend/internal/collect"
	"modelrun/backend/internal/deploy"
	"modelrun/backend/internal/dispatch"
	"modelrun/backend/internal/domain"
	"modelrun/backend/internal/logging"
	"modelrun/backend/internal/realtime"
	"modelrun/backend/internal/runstate"
	"modelrun/backend/internal/store"
)

type API struct {
	store      *store.Store
	state      *runstate.State
	executor   *deploy.Executor
	hub        *realtime.Hub
	collector  *collect.Collector
	dispatcher *dispatch.Executor
	staticDir  string
	startedAt  time.Time
}

func New(st *store.Store, state *runstate.State, executor *deploy.Executor, hub *realtime.Hub, staticDir string) *API {
	collector := collect.New()
	return &API{
		store:      st,
		state:      state,
		executor:   executor,
		hub:        hub,
		collector:  collector,
		dispatcher: dispatch.New(st, state, collector),
		staticDir:  staticDir,
		startedAt:  time.Now(),
	}
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/system/status", a.handleSystemStatus)

	mux.HandleFunc("/api/projects", a.handleProjects)
	mux.HandleFunc("/api/projects/", a.handleProject)

	mux.HandleFunc("/api/servers", a.handleServers)
	mux.HandleFunc("/api/servers/", a.handleServer)

	mux.HandleFunc("/api/models/search", a.handleModelSearch)
	mux.HandleFunc("/api/models/scan", a.handleModelScan)
	mux.HandleFunc("/api/models", a.handleModels)
	mux.HandleFunc("/api/models/", a.handleModel)

	mux.HandleFunc("/api/deployments", a.handleDeployments)
	mux.HandleFunc("/api/deployments/", a.handleDeployment)

	mux.HandleFunc("/api/tasks", a.handleTasks)
	mux.HandleFunc("/api/tasks/", a.handleTask)
	mux.HandleFunc("/api/remote-task-presets", a.handleRemoteTaskPresets)
	mux.HandleFunc("/api/remote-tasks", a.handleRemoteTasks)
	mux.HandleFunc("/api/remote-tasks/", a.handleRemoteTask)
	mux.HandleFunc("/api/action-templates", a.handleActionTemplates)
	mux.HandleFunc("/api/action-templates/", a.handleActionTemplate)
	mux.HandleFunc("/api/bootstrap-configs", a.handleBootstrapConfigs)
	mux.HandleFunc("/api/bootstrap-configs/", a.handleBootstrapConfig)
	mux.HandleFunc("/api/pipeline-step-templates", a.handlePipelineStepTemplates)
	mux.HandleFunc("/api/pipeline-step-templates/", a.handlePipelineStepTemplate)
	mux.HandleFunc("/api/pipeline-templates", a.handlePipelineTemplates)
	mux.HandleFunc("/api/logs", a.handleLogs)
	mux.HandleFunc("/api/backend-logs", a.handleBackendLogs)

	mux.HandleFunc("/ws", a.hub.ServeHTTP)

	if a.staticDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(a.staticDir)))
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]any{
				"name":    "modelrun-backend",
				"message": "backend is running",
			})
		})
	}

	return withRequestLogging(withCORS(mux))
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":   "ok",
		"uptime":   time.Since(a.startedAt).String(),
		"datetime": domain.Now(),
	})
}

func readJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(target)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeError(w, http.StatusBadRequest, err)
}

func writeError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		err = errors.New(http.StatusText(status))
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func withRequestLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/health" {
			next.ServeHTTP(w, r)
			return
		}
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		startedAt := time.Now()
		next.ServeHTTP(recorder, r)
		logging.Infof(
			"http",
			"%s %s -> %s (%dms)",
			r.Method,
			r.URL.RequestURI(),
			strconv.Itoa(recorder.status),
			time.Since(startedAt).Milliseconds(),
		)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

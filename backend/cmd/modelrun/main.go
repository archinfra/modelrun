package main

import (
	"net/http"
	"time"

	"modelrun/backend/internal/api"
	"modelrun/backend/internal/config"
	"modelrun/backend/internal/deploy"
	"modelrun/backend/internal/logging"
	"modelrun/backend/internal/realtime"
	"modelrun/backend/internal/runstate"
	"modelrun/backend/internal/runtimefiles"
	"modelrun/backend/internal/store"
)

func main() {
	cfg := config.Load()

	logger, err := logging.Setup(cfg.LogDir)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Close()
	}()

	st, err := store.New(cfg.DataPath)
	if err != nil {
		logging.Errorf("main", "open data store: %v", err)
		return
	}
	defer func() {
		_ = st.Close()
	}()

	hub := realtime.NewHub()
	state := runstate.New()
	runLogs, err := runtimefiles.New(cfg.RunLogDir)
	if err != nil {
		logging.Errorf("main", "open runtime log dir: %v", err)
		return
	}
	executor := deploy.NewExecutor(st, state, hub, runLogs)
	handler := api.New(st, state, executor, hub, cfg.StaticDir)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logging.Infof("main", "modelrun backend listening on %s", cfg.Addr)
	logging.Infof("main", "data file: %s", cfg.DataPath)
	logging.Infof("main", "backend log file: %s", logger.Path())
	logging.Infof("main", "runtime log dir: %s", runLogs.BaseDir())
	if cfg.StaticDir != "" {
		logging.Infof("main", "static dir: %s", cfg.StaticDir)
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logging.Errorf("main", "serve: %v", err)
	}
}

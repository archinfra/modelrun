package main

import (
	"log"
	"net/http"
	"time"

	"modelrun/backend/internal/api"
	"modelrun/backend/internal/config"
	"modelrun/backend/internal/deploy"
	"modelrun/backend/internal/realtime"
	"modelrun/backend/internal/runstate"
	"modelrun/backend/internal/store"
)

func main() {
	cfg := config.Load()

	st, err := store.New(cfg.DataPath)
	if err != nil {
		log.Fatalf("open data store: %v", err)
	}

	hub := realtime.NewHub()
	state := runstate.New()
	executor := deploy.NewExecutor(st, state, hub)
	handler := api.New(st, state, executor, hub, cfg.StaticDir)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("modelrun backend listening on %s", cfg.Addr)
	log.Printf("data file: %s", cfg.DataPath)
	if cfg.StaticDir != "" {
		log.Printf("static dir: %s", cfg.StaticDir)
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}

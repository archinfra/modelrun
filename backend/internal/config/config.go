package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Addr      string
	DataPath  string
	StaticDir string
	LogDir    string
	RunLogDir string
}

func Load() Config {
	cfg := Config{
		Addr:     ":8080",
		DataPath: "data/modelrun.db",
	}

	if v := os.Getenv("MODELRUN_ADDR"); v != "" {
		cfg.Addr = v
	}
	if v := os.Getenv("MODELRUN_DATA"); v != "" {
		cfg.DataPath = v
	}
	if v := os.Getenv("MODELRUN_STATIC_DIR"); v != "" {
		cfg.StaticDir = v
	}
	cfg.LogDir = filepath.Join(filepath.Dir(cfg.DataPath), "logs")
	cfg.RunLogDir = filepath.Join(filepath.Dir(cfg.DataPath), "runs")
	if v := os.Getenv("MODELRUN_LOG_DIR"); v != "" {
		cfg.LogDir = v
	}
	if v := os.Getenv("MODELRUN_RUN_LOG_DIR"); v != "" {
		cfg.RunLogDir = v
	}

	return cfg
}

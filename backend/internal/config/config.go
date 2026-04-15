package config

import "os"

type Config struct {
	Addr      string
	DataPath  string
	StaticDir string
}

func Load() Config {
	cfg := Config{
		Addr:     ":8080",
		DataPath: "data/modelrun.json",
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

	return cfg
}

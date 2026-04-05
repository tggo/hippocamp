package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all server configuration.
type Config struct {
	Store    StoreConfig       `yaml:"store"`
	Prefixes map[string]string `yaml:"prefixes"`
}

// StoreConfig holds RDF store settings.
type StoreConfig struct {
	DefaultFile string `yaml:"default_file"`
	AutoLoad    bool   `yaml:"auto_load"`
	Format      string `yaml:"format"`
}

// Load reads config from path (if it exists) and applies ENV overrides.
// Missing file is not an error — defaults are used instead.
func Load(path string) (*Config, error) {
	cfg := &Config{
		Store: StoreConfig{
			DefaultFile: "./data/default.trig",
			AutoLoad:    true,
			Format:      "trig",
		},
		Prefixes: make(map[string]string),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnv(cfg)
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	if cfg.Prefixes == nil {
		cfg.Prefixes = make(map[string]string)
	}

	applyEnv(cfg)
	return cfg, nil
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("HIPPOCAMP_STORE_DEFAULT_FILE"); v != "" {
		cfg.Store.DefaultFile = v
	}
	if v := os.Getenv("HIPPOCAMP_STORE_AUTO_LOAD"); v != "" {
		cfg.Store.AutoLoad = v == "true" || v == "1"
	}
	if v := os.Getenv("HIPPOCAMP_STORE_FORMAT"); v != "" {
		cfg.Store.Format = v
	}
}

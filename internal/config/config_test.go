package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ruslanmv/hippocamp/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := config.Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Store.DefaultFile == "" {
		t.Error("DefaultFile should have a default value")
	}
	if cfg.Store.Format != "trig" {
		t.Errorf("expected format trig, got %q", cfg.Store.Format)
	}
	if !cfg.Store.AutoLoad {
		t.Error("AutoLoad should default to true")
	}
}

func TestLoad_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
store:
  default_file: "./testdata/my.trig"
  auto_load: false
  format: "nt"
prefixes:
  ex: "http://example.org/"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Store.DefaultFile != "./testdata/my.trig" {
		t.Errorf("unexpected DefaultFile: %q", cfg.Store.DefaultFile)
	}
	if cfg.Store.AutoLoad {
		t.Error("AutoLoad should be false")
	}
	if cfg.Store.Format != "nt" {
		t.Errorf("expected format nt, got %q", cfg.Store.Format)
	}
	if cfg.Prefixes["ex"] != "http://example.org/" {
		t.Errorf("unexpected prefix ex: %q", cfg.Prefixes["ex"])
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	t.Setenv("HIPPOCAMP_STORE_DEFAULT_FILE", "/tmp/override.trig")
	t.Setenv("HIPPOCAMP_STORE_AUTO_LOAD", "false")
	t.Setenv("HIPPOCAMP_STORE_FORMAT", "nq")

	cfg, err := config.Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Store.DefaultFile != "/tmp/override.trig" {
		t.Errorf("ENV override failed: %q", cfg.Store.DefaultFile)
	}
	if cfg.Store.AutoLoad {
		t.Error("ENV override for AutoLoad failed")
	}
	if cfg.Store.Format != "nq" {
		t.Errorf("ENV override for Format failed: %q", cfg.Store.Format)
	}
}

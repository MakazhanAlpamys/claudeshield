package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Sandbox.Mount == "" {
		t.Error("default mount should not be empty")
	}
	if cfg.Sandbox.Network {
		t.Error("network should be disabled by default")
	}
	if cfg.Sandbox.UseGVisor {
		t.Error("gVisor should be disabled by default")
	}
	if len(cfg.Rules.Block) == 0 {
		t.Error("should have default block rules")
	}
	if len(cfg.Rules.Allow) == 0 {
		t.Error("should have default allow rules")
	}
	if cfg.Secrets.Provider != "env" {
		t.Errorf("default provider should be env, got %s", cfg.Secrets.Provider)
	}
	if !cfg.Audit.Enabled {
		t.Error("audit should be enabled by default")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := DefaultConfig()
	cfg.Sandbox.Network = true
	cfg.Sandbox.MemoryLimit = "4g"

	if err := Save(tmpDir, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Check file exists
	configPath := filepath.Join(tmpDir, ConfigFileName)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}

	// Load it back
	loaded, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !loaded.Sandbox.Network {
		t.Error("Network should be true after load")
	}
	if loaded.Sandbox.MemoryLimit != "4g" {
		t.Errorf("MemoryLimit expected 4g, got %s", loaded.Sandbox.MemoryLimit)
	}
}

func TestLoadMissing(t *testing.T) {
	tmpDir := t.TempDir()

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load should not error for missing config: %v", err)
	}

	// Should return defaults
	if cfg.Sandbox.Mount == "" {
		t.Error("should return default config when file is missing")
	}
}

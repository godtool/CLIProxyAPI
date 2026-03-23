package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigOptional_CodexSync(t *testing.T) {
	tmp := t.TempDir()
	configPath := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(configPath, []byte(`
auth-dir: "~/.cli-proxy-api"
codex-sync:
  enable: true
  source: "~/.codex/auth.json"
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfigOptional(configPath, false)
	if err != nil {
		t.Fatalf("LoadConfigOptional() error = %v", err)
	}

	if !cfg.CodexSync.Enable {
		t.Fatal("expected codex sync to be enabled")
	}
	if got, want := cfg.CodexSync.Source, "~/.codex/auth.json"; got != want {
		t.Fatalf("CodexSync.Source = %q, want %q", got, want)
	}
}

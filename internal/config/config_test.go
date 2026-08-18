package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveNeverPersistsLegacyPrivateKey(t *testing.T) {
	repoPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoPath, ".bit"), 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(repoPath, ".bit", "config.json")
	if err := os.WriteFile(configPath, []byte(`{"privateKey":"old"}`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{PrivateKey: "secret", Remotes: map[string]Remote{}}
	if err := Save(repoPath, cfg); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret") || strings.Contains(string(data), "privateKey") {
		t.Fatalf("private key was persisted: %s", data)
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("config mode = %o, want 600", info.Mode().Perm())
	}
}

func TestEnsureLocalExcludeIsIdempotent(t *testing.T) {
	repoPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoPath, ".git", "info"), 0755); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := EnsureLocalExclude(repoPath); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(filepath.Join(repoPath, ".git", "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), ".bit/") != 1 {
		t.Fatalf("exclude entry is not idempotent: %q", data)
	}
}

// Package config reads and writes .bit/config.json.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const configDir = ".bit"
const configFile = "config.json"

type Remote struct {
	URL             string `json:"url"`
	Network         string `json:"network,omitempty"`
	ContractAddress string `json:"contractAddress,omitempty"`
	RepoID          uint64 `json:"repoId"`
}

type Config struct {
	RPCURL          string            `json:"rpcURL"`
	ContractAddress string            `json:"contractAddress"`
	PrivateKey      string            `json:"privateKey,omitempty"` // Legacy only; Save clears it.
	IPFSURL         string            `json:"ipfsURL"`
	RepoID          uint64            `json:"repoId"`
	Remotes         map[string]Remote `json:"remotes"`
}

func configPath(repoPath string) string {
	return filepath.Join(repoPath, configDir, configFile)
}

func Load(repoPath string) (*Config, error) {
	data, err := os.ReadFile(configPath(repoPath))
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Remotes == nil {
		cfg.Remotes = make(map[string]Remote)
	}
	return &cfg, nil
}

// Save never persists the legacy PrivateKey field.
func Save(repoPath string, cfg *Config) error {
	dir := filepath.Join(repoPath, configDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	safeConfig := *cfg
	safeConfig.PrivateKey = ""
	data, err := json.MarshalIndent(&safeConfig, "", "  ")
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, configPath(repoPath))
}

// EnsureLocalExclude avoids modifying the user's tracked .gitignore.
func EnsureLocalExclude(repoPath string) error {
	excludePath := filepath.Join(repoPath, ".git", "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(excludePath), 0755); err != nil {
		return err
	}
	data, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == ".bit/" {
			return nil
		}
	}
	prefix := ""
	if len(data) > 0 && data[len(data)-1] != '\n' {
		prefix = "\n"
	}
	file, err := os.OpenFile(excludePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(prefix + ".bit/\n")
	return err
}

func AddRemote(repoPath, name string, remote Remote) error {
	cfg, err := Load(repoPath)
	if err != nil {
		return err
	}
	cfg.Remotes[name] = remote
	return Save(repoPath, cfg)
}

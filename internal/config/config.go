// Package config reads and writes the per-repository settings stored at
// .bit/config.json, including RPC/IPFS endpoints and registered remotes.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const configDir = ".bit"
const configFile = "config.json"

// Remote holds the information for a remote registered via `bit remote add`.
type Remote struct {
	URL             string `json:"url"`                       // bit://<network>/<registry>/<repoId>
	Network         string `json:"network,omitempty"`         // remote network label
	ContractAddress string `json:"contractAddress,omitempty"` // remote registry address
	RepoID          uint64 `json:"repoId"`                    // repo ID assigned by the chain
}

// Config is the per-project settings persisted at .bit/config.json.
type Config struct {
	RPCURL          string            `json:"rpcURL"`               // Ethereum node address
	ContractAddress string            `json:"contractAddress"`      // BitRegistry contract address
	PrivateKey      string            `json:"privateKey,omitempty"` // legacy only; never written by Save
	IPFSURL         string            `json:"ipfsURL"`              // IPFS node address
	RepoID          uint64            `json:"repoId"`               // repo ID assigned by the chain
	Remotes         map[string]Remote `json:"remotes"`              // remote name -> Remote info
}

// configPath returns the path to .bit/config.json under repoPath.
func configPath(repoPath string) string {
	return filepath.Join(repoPath, configDir, configFile)
}

// Load reads .bit/config.json and returns the resulting Config.
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

// Save atomically writes Config to .bit/config.json, stripping any
// legacy PrivateKey field so it is never persisted to disk.
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

// EnsureLocalExclude keeps Bit's repository-local state out of Git without
// modifying the user's tracked .gitignore file.
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

// AddRemote adds a remote to the config and saves it.
func AddRemote(repoPath, name string, remote Remote) error {
	cfg, err := Load(repoPath)
	if err != nil {
		return err
	}
	cfg.Remotes[name] = remote
	return Save(repoPath, cfg)
}

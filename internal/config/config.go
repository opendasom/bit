package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const configDir = ".bit"
const configFile = "config.json"

// Remote는 bit remote add로 등록한 원격 저장소 정보를 담는다.
type Remote struct {
	URL             string `json:"url"`                       // bit://<network>/<registry>/<repoId>
	Network         string `json:"network,omitempty"`         // remote network label
	ContractAddress string `json:"contractAddress,omitempty"` // remote registry address
	RepoID          uint64 `json:"repoId"`                    // 체인에서 발급된 저장소 ID
}

// Config는 .bit/config.json에 저장되는 프로젝트 설정이다.
type Config struct {
	RPCURL          string            `json:"rpcURL"`               // 이더리움 노드 주소
	ContractAddress string            `json:"contractAddress"`      // BitRegistry 컨트랙트 주소
	PrivateKey      string            `json:"privateKey,omitempty"` // legacy only; never written by Save
	IPFSURL         string            `json:"ipfsURL"`              // IPFS 노드 주소
	RepoID          uint64            `json:"repoId"`               // 체인에서 발급된 저장소 ID
	Remotes         map[string]Remote `json:"remotes"`              // remote 이름 → Remote 정보
}

// configPath는 .bit/config.json 경로를 반환한다.
func configPath(repoPath string) string {
	return filepath.Join(repoPath, configDir, configFile)
}

// Load는 .bit/config.json을 읽어 Config를 반환한다.
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

// Save는 Config를 .bit/config.json에 저장한다.
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

// AddRemote는 config에 remote를 추가하고 저장한다.
func AddRemote(repoPath, name string, remote Remote) error {
	cfg, err := Load(repoPath)
	if err != nil {
		return err
	}
	cfg.Remotes[name] = remote
	return Save(repoPath, cfg)
}

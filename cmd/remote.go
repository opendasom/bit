package cmd

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/opendasom/bit/internal/config"
	"github.com/spf13/cobra"
)

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remotes",
}

var remoteAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add a new remote (예: bit remote add origin bit://sepolia/0xABC.../1)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, url := args[0], args[1]

		parsed, err := parseRemoteURL(url)
		if err != nil {
			return fmt.Errorf("URL 형식 오류: %w", err)
		}
		cfg, err := config.Load(".")
		if err != nil {
			return fmt.Errorf("config 읽기 실패 (.bit/config.json): %w", err)
		}
		if !strings.EqualFold(parsed.ContractAddress, cfg.ContractAddress) {
			return fmt.Errorf("remote contract %s does not match configured contract %s", parsed.ContractAddress, cfg.ContractAddress)
		}

		remote := config.Remote{
			URL:             url,
			Network:         parsed.Network,
			ContractAddress: parsed.ContractAddress,
			RepoID:          parsed.RepoID,
		}
		cfg.Remotes[name] = remote
		if err := config.Save(".", cfg); err != nil {
			return fmt.Errorf("remote 저장 실패: %w", err)
		}

		fmt.Printf("remote '%s' 추가 완료 (repoId: %d)\n", name, parsed.RepoID)
		return nil
	},
}

// parseRepoID는 bit://<network>/<registry>/<repoId> 에서 repoId를 추출한다.
type parsedRemote struct {
	Network         string
	ContractAddress string
	RepoID          uint64
}

func parseRemoteURL(rawURL string) (*parsedRemote, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme != "bit" || parsedURL.Host == "" || parsedURL.User != nil ||
		parsedURL.RawQuery != "" || parsedURL.ForceQuery || parsedURL.Fragment != "" || strings.Contains(parsedURL.Host, ":") {
		return nil, fmt.Errorf("올바른 형식: bit://<network>/<registry>/<repoId>")
	}
	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(parts) != 2 || !common.IsHexAddress(parts[0]) {
		return nil, fmt.Errorf("올바른 형식: bit://<network>/<registry>/<repoId>")
	}
	repoIDStr := parts[1]
	repoID, err := strconv.ParseUint(repoIDStr, 10, 64)
	if err != nil || repoID == 0 {
		return nil, fmt.Errorf("repoId는 1 이상의 숫자여야 합니다: %s", repoIDStr)
	}
	return &parsedRemote{Network: parsedURL.Host, ContractAddress: common.HexToAddress(parts[0]).Hex(), RepoID: repoID}, nil
}

func validateRemoteForConfig(remote config.Remote, cfg *config.Config) error {
	parsed, err := parseRemoteURL(remote.URL)
	if err != nil {
		return fmt.Errorf("remote URL 형식 오류: %w", err)
	}
	if parsed.RepoID != remote.RepoID {
		return fmt.Errorf("remote repoId mismatch: URL=%d config=%d", parsed.RepoID, remote.RepoID)
	}
	if !strings.EqualFold(parsed.ContractAddress, cfg.ContractAddress) {
		return fmt.Errorf("remote contract %s does not match configured contract %s", parsed.ContractAddress, cfg.ContractAddress)
	}
	if remote.ContractAddress != "" && !strings.EqualFold(remote.ContractAddress, parsed.ContractAddress) {
		return fmt.Errorf("stored remote contract %s does not match URL contract %s", remote.ContractAddress, parsed.ContractAddress)
	}
	if remote.Network != "" && !strings.EqualFold(remote.Network, parsed.Network) {
		return fmt.Errorf("stored remote network %s does not match URL network %s", remote.Network, parsed.Network)
	}
	return nil
}

func init() {
	remoteCmd.AddCommand(remoteAddCmd)
}

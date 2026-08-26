package cli

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
	Short: "Add a new remote (e.g. bit remote add origin bit://sepolia/0xABC.../1)",
	Args:  argsWithUsage(cobra.ExactArgs(2)),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, url := args[0], args[1]

		parsed, err := parseRemoteURL(url)
		if err != nil {
			return fmt.Errorf("invalid URL format: %w", err)
		}
		cfg, err := config.Load(".")
		if err != nil {
			return fmt.Errorf("config read failed (.bit/config.json): %w", err)
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
			return fmt.Errorf("failed to save remote: %w", err)
		}

		fmt.Printf("added remote '%s' (repoId: %d)\n", name, parsed.RepoID)
		return nil
	},
}

type parsedRemote struct {
	Network         string
	ContractAddress string
	RepoID          uint64
}

func parseRemoteURL(rawURL string) (*parsedRemote, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme != "bit" || parsedURL.Host == "" || parsedURL.User != nil ||
		parsedURL.RawQuery != "" || parsedURL.ForceQuery || parsedURL.Fragment != "" || strings.Contains(parsedURL.Host, ":") {
		return nil, fmt.Errorf("expected format: bit://<network>/<registry>/<repoId>")
	}
	parts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(parts) != 2 || !common.IsHexAddress(parts[0]) {
		return nil, fmt.Errorf("expected format: bit://<network>/<registry>/<repoId>")
	}
	repoIDStr := parts[1]
	repoID, err := strconv.ParseUint(repoIDStr, 10, 64)
	if err != nil || repoID == 0 {
		return nil, fmt.Errorf("repoId must be a number >= 1: %s", repoIDStr)
	}
	return &parsedRemote{Network: parsedURL.Host, ContractAddress: common.HexToAddress(parts[0]).Hex(), RepoID: repoID}, nil
}

func validateRemoteForConfig(remote config.Remote, cfg *config.Config) error {
	parsed, err := parseRemoteURL(remote.URL)
	if err != nil {
		return fmt.Errorf("invalid remote URL format: %w", err)
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

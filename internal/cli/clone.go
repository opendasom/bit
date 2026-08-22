package cli

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"github.com/opendasom/bit/internal/app"
	"github.com/opendasom/bit/internal/chain"
	"github.com/opendasom/bit/internal/config"
	bitgit "github.com/opendasom/bit/internal/git"
	"github.com/opendasom/bit/internal/ipfs"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone <url> [directory]",
	Short: "Clone an existing Bit repository without a signing key",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remote, err := parseRemoteURL(args[0])
		if err != nil {
			return fmt.Errorf("remote URL: %w", err)
		}
		rpcURL, _ := cmd.Flags().GetString("rpc")
		ipfsURL, _ := cmd.Flags().GetString("ipfs")
		branch, _ := cmd.Flags().GetString("branch")
		target := fmt.Sprintf("bit-repo-%d", remote.RepoID)
		if len(args) == 2 {
			target = args[1]
		}
		target, err = filepath.Abs(target)
		if err != nil {
			return err
		}
		if _, err := os.Stat(target); err == nil {
			return fmt.Errorf("clone target already exists: %s", target)
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(target, 0755); err != nil {
			return err
		}
		if err := bitgit.InitRepository(target); err != nil {
			return fmt.Errorf("git init failed: %w", err)
		}
		if err := config.EnsureLocalExclude(target); err != nil {
			return fmt.Errorf("failed to exclude .bit: %w", err)
		}
		cfg := &config.Config{
			RPCURL: rpcURL, ContractAddress: remote.ContractAddress, IPFSURL: ipfsURL,
			Remotes: map[string]config.Remote{
				"origin": {
					URL: args[0], Network: remote.Network, ContractAddress: remote.ContractAddress, RepoID: remote.RepoID,
				},
			},
		}
		if err := config.Save(target, cfg); err != nil {
			return fmt.Errorf("config save failed: %w", err)
		}
		chainClient, err := chain.NewReadOnlyClient(rpcURL, remote.ContractAddress)
		if err != nil {
			return fmt.Errorf("chain connection failed: %w", err)
		}
		defer chainClient.Close()
		if err := app.Pull(chainClient, ipfs.NewClient(ipfsURL), target, new(big.Int).SetUint64(remote.RepoID), branch); err != nil {
			return fmt.Errorf("clone left an initialized directory at %s: %w", target, err)
		}
		fmt.Printf("cloned %s into %s\n", args[0], target)
		return nil
	},
}

func init() {
	cloneCmd.Flags().String("rpc", "", "Ethereum RPC URL")
	cloneCmd.Flags().String("ipfs", "http://localhost:5001", "IPFS node API URL")
	cloneCmd.Flags().String("branch", "main", "branch to clone")
	cloneCmd.MarkFlagRequired("rpc")
	rootCmd.AddCommand(cloneCmd)
}

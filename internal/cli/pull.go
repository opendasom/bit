package cli

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/app"
	"github.com/opendasom/bit/internal/chain"
	"github.com/opendasom/bit/internal/config"
	"github.com/opendasom/bit/internal/ipfs"
	"github.com/spf13/cobra"
)

// pullCmd implements `bit pull <remote> <branch>`: it resolves the named
// remote from .bit/config.json and delegates to app.Pull to fetch and
// verify any commits missing from the local repository.
var pullCmd = &cobra.Command{
	Use:   "pull <remote> <branch>",
	Short: "Pull branch from remote",
	Args:  argsWithUsage(cobra.ExactArgs(2)),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteName, branch := args[0], args[1]

		cfg, err := config.Load(".")
		if err != nil {
			return fmt.Errorf("config read failed (.bit/config.json): %w", err)
		}

		remote, ok := cfg.Remotes[remoteName]
		if !ok {
			return fmt.Errorf("remote '%s' not found", remoteName)
		}
		if err := validateRemoteForConfig(remote, cfg); err != nil {
			return err
		}
		repoID := new(big.Int).SetUint64(remote.RepoID)

		chainClient, err := chain.NewReadOnlyClient(cfg.RPCURL, cfg.ContractAddress)
		if err != nil {
			return fmt.Errorf("chain connection failed: %w", err)
		}
		defer chainClient.Close()
		ipfsClient := ipfs.NewClient(cfg.IPFSURL)

		return app.Pull(chainClient, ipfsClient, ".", repoID, branch)
	},
}

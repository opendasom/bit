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

// pushCmd implements `bit push <remote>`: it resolves the named remote and
// signing key from .bit/config.json / BIT_PRIVATE_KEY, then delegates to
// app.Push to upload and record any local commits missing on-chain.
var pushCmd = &cobra.Command{
	Use:   "push <remote>",
	Short: "Push current branch to remote",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteName := args[0]

		cfg, err := config.Load(".")
		if err != nil {
			return fmt.Errorf("config 읽기 실패 (.bit/config.json): %w", err)
		}

		remote, ok := cfg.Remotes[remoteName]
		if !ok {
			return fmt.Errorf("remote '%s' 를 찾을 수 없습니다", remoteName)
		}
		if err := validateRemoteForConfig(remote, cfg); err != nil {
			return err
		}
		repoID := new(big.Int).SetUint64(remote.RepoID)
		privateKey, err := resolveSigningKey("", cfg)
		if err != nil {
			return err
		}

		chainClient, err := chain.NewClient(cfg.RPCURL, cfg.ContractAddress, privateKey)
		if err != nil {
			return fmt.Errorf("체인 연결 실패: %w", err)
		}
		defer chainClient.Close()
		ipfsClient := ipfs.NewClient(cfg.IPFSURL)

		return app.Push(chainClient, ipfsClient, ".", repoID)
	},
}

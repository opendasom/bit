package cmd

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/app"
	"github.com/opendasom/bit/internal/chain"
	"github.com/opendasom/bit/internal/config"
	"github.com/opendasom/bit/internal/ipfs"
	"github.com/spf13/cobra"
)

var pullCmd = &cobra.Command{
	Use:   "pull <remote> <branch>",
	Short: "Pull branch from remote",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteName, branch := args[0], args[1]

		cfg, err := config.Load(".")
		if err != nil {
			return fmt.Errorf("config 읽기 실패 (.bit/config.json): %w", err)
		}

		remote, ok := cfg.Remotes[remoteName]
		if !ok {
			return fmt.Errorf("remote '%s' 를 찾을 수 없습니다", remoteName)
		}
		repoID := new(big.Int).SetUint64(remote.RepoID)

		chainClient, err := chain.NewClient(cfg.RPCURL, cfg.ContractAddress, cfg.PrivateKey)
		if err != nil {
			return fmt.Errorf("체인 연결 실패: %w", err)
		}
		ipfsClient := ipfs.NewClient(cfg.IPFSURL)

		return app.Pull(chainClient, ipfsClient, ".", repoID, branch)
	},
}

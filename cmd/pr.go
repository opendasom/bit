package cmd

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/app"
	"github.com/opendasom/bit/internal/chain"
	"github.com/opendasom/bit/internal/config"
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
}

var prCreateCmd = &cobra.Command{
	Use:   "create <remote> <branch>",
	Short: "Create a pull request from the current branch to a remote branch",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		remoteName, targetBranch := args[0], args[1]

		cfg, err := config.Load(".")
		if err != nil {
			return fmt.Errorf("config 읽기 실패 (.bit/config.json): %w", err)
		}
		if cfg.RepoID == 0 {
			return fmt.Errorf("현재 저장소 repoId가 설정되지 않았습니다. 먼저 bit init 또는 bit fork를 실행하세요")
		}

		remote, ok := cfg.Remotes[remoteName]
		if !ok {
			return fmt.Errorf("remote '%s' 를 찾을 수 없습니다", remoteName)
		}

		chainClient, err := chain.NewClient(cfg.RPCURL, cfg.ContractAddress, cfg.PrivateKey)
		if err != nil {
			return fmt.Errorf("체인 연결 실패: %w", err)
		}

		sourceRepoID := new(big.Int).SetUint64(cfg.RepoID)
		targetRepoID := new(big.Int).SetUint64(remote.RepoID)

		prID, sourceBranch, err := app.PRCreate(chainClient, ".", sourceRepoID, targetRepoID, targetBranch)
		if err != nil {
			return err
		}

		fmt.Printf("PR 생성 완료: #%s\n", prID.String())
		fmt.Printf("  source: current repo #%d (%s)\n", cfg.RepoID, sourceBranch)
		fmt.Printf("  target: %s #%d (%s)\n", remoteName, remote.RepoID, targetBranch)
		return nil
	},
}

func init() {
	prCmd.AddCommand(prCreateCmd)
}

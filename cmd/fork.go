package cmd

import (
	"fmt"
	"math/big"
	"os"
	"os/exec"

	"github.com/opendasom/bit/internal/app"
	"github.com/opendasom/bit/internal/chain"
	"github.com/opendasom/bit/internal/config"
	"github.com/opendasom/bit/internal/ipfs"
	"github.com/spf13/cobra"
)

var forkCmd = &cobra.Command{
	Use:   "fork <bitURL>",
	Short: "Fork a remote repository into a new local repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceURL := args[0]

		rpcURL, _ := cmd.Flags().GetString("rpc")
		contractAddress, _ := cmd.Flags().GetString("contract")
		privateKey, _ := cmd.Flags().GetString("key")
		ipfsURL, _ := cmd.Flags().GetString("ipfs")
		branch, _ := cmd.Flags().GetString("branch")

		// 1. .git 없으면 자동으로 git init
		if _, err := os.Stat(".git"); os.IsNotExist(err) {
			fmt.Println("git 저장소가 없습니다. git init을 실행합니다...")
			initCmd := exec.Command("git", "init")
			initCmd.Stdout = os.Stdout
			initCmd.Stderr = os.Stderr
			if err := initCmd.Run(); err != nil {
				return fmt.Errorf("git init 실패: %w", err)
			}
		}

		// 2. 플래그 미입력 시 기존 .bit/config.json에서 읽기
		if rpcURL == "" || contractAddress == "" || privateKey == "" {
			cfg, err := config.Load(".")
			if err != nil {
				return fmt.Errorf("--rpc, --contract, --key 플래그가 필요합니다 (또는 .bit/config.json이 있어야 합니다): %w", err)
			}
			if rpcURL == "" {
				rpcURL = cfg.RPCURL
			}
			if contractAddress == "" {
				contractAddress = cfg.ContractAddress
			}
			if privateKey == "" {
				privateKey = cfg.PrivateKey
			}
			if ipfsURL == "" {
				ipfsURL = cfg.IPFSURL
			}
		}
		if ipfsURL == "" {
			ipfsURL = "http://localhost:5001"
		}

		// 3. source repoId 파싱
		sourceRepoID, err := parseRepoID(sourceURL)
		if err != nil {
			return fmt.Errorf("bitURL 파싱 실패: %w", err)
		}

		// 4. 체인 연결
		chainClient, err := chain.NewClient(rpcURL, contractAddress, privateKey)
		if err != nil {
			return fmt.Errorf("체인 연결 실패: %w", err)
		}
		ipfsClient := ipfs.NewClient(ipfsURL)

		// 5~7. source 히스토리 조회 → fork 저장소 생성 → 커밋 복원 및 기록
		forkRepoID, err := app.Fork(chainClient, ipfsClient, ".", new(big.Int).SetUint64(sourceRepoID), branch)
		if err != nil {
			return err
		}

		// 8. .bit/config.json 저장
		cfg := &config.Config{
			RPCURL:          rpcURL,
			ContractAddress: contractAddress,
			PrivateKey:      privateKey,
			IPFSURL:         ipfsURL,
			RepoID:          forkRepoID.Uint64(),
			Remotes:         make(map[string]config.Remote),
		}
		if err := config.Save(".", cfg); err != nil {
			return fmt.Errorf("config 저장 실패: %w", err)
		}

		// 9. remote 설정
		forkURL := fmt.Sprintf("bit://local/%s/%s", contractAddress, forkRepoID.String())
		if err := config.AddRemote(".", "origin", config.Remote{URL: forkURL, RepoID: forkRepoID.Uint64()}); err != nil {
			return fmt.Errorf("origin remote 저장 실패: %w", err)
		}
		if err := config.AddRemote(".", "upstream", config.Remote{URL: sourceURL, RepoID: sourceRepoID}); err != nil {
			return fmt.Errorf("upstream remote 저장 실패: %w", err)
		}

		fmt.Printf("\nfork 완료!\n")
		fmt.Printf("  origin   → %s (repoId: %s)\n", forkURL, forkRepoID.String())
		fmt.Printf("  upstream → %s (repoId: %d)\n", sourceURL, sourceRepoID)
		fmt.Printf("\n새 커밋 후 push: bit push origin\n")
		fmt.Printf("PR 생성:         bit pr create upstream %s\n", branch)

		return nil
	},
}

func init() {
	forkCmd.Flags().String("rpc", "", "이더리움 RPC URL")
	forkCmd.Flags().String("contract", "", "BitRegistry 컨트랙트 주소")
	forkCmd.Flags().String("key", "", "지갑 개인키 (0x 제외)")
	forkCmd.Flags().String("ipfs", "", "IPFS 노드 주소 (기본값: http://localhost:5001)")
	forkCmd.Flags().String("branch", "main", "fork할 브랜치")
}

package cmd

import (
	"fmt"
	"strconv"
	"strings"

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

		// bit://<network>/<registry>/<repoId> 파싱
		repoID, err := parseRepoID(url)
		if err != nil {
			return fmt.Errorf("URL 형식 오류: %w", err)
		}

		remote := config.Remote{
			URL:    url,
			RepoID: repoID,
		}

		if err := config.AddRemote(".", name, remote); err != nil {
			return fmt.Errorf("remote 저장 실패: %w", err)
		}

		fmt.Printf("remote '%s' 추가 완료 (repoId: %d)\n", name, repoID)
		return nil
	},
}

// parseRepoID는 bit://<network>/<registry>/<repoId> 에서 repoId를 추출한다.
func parseRepoID(url string) (uint64, error) {
	// "bit://" 제거
	trimmed := strings.TrimPrefix(url, "bit://")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 3 {
		return 0, fmt.Errorf("올바른 형식: bit://<network>/<registry>/<repoId>")
	}

	// 마지막 부분이 repoId, #branch 있으면 제거
	repoIDStr := strings.Split(parts[len(parts)-1], "#")[0]
	repoID, err := strconv.ParseUint(repoIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("repoId는 숫자여야 합니다: %s", repoIDStr)
	}
	return repoID, nil
}

func init() {
	remoteCmd.AddCommand(remoteAddCmd)
}

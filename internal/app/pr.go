package app

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/chain"
	"github.com/opendasom/bit/internal/git"
)

// PRCreate opens a pull request from the current branch of the repo at
// repoPath (sourceRepoID) targeting targetBranch on targetRepoID. It
// verifies that the source branch actually has commits beyond whatever the
// target branch has already seen before recording the PR on-chain.
// It returns the new PR's ID and the resolved source branch name.
func PRCreate(chainClient ChainClient, repoPath string, sourceRepoID, targetRepoID *big.Int, targetBranch string) (*big.Int, string, error) {
	sourceBranch, err := git.CurrentBranch(repoPath)
	if err != nil {
		return nil, "", fmt.Errorf("현재 브랜치 조회 실패: %w", err)
	}

	sourceHistoryLen, err := chainClient.GetBranchHistoryLength(sourceRepoID, sourceBranch)
	if err != nil {
		return nil, "", fmt.Errorf("source 브랜치 히스토리 조회 실패: %w", err)
	}
	if sourceHistoryLen.Sign() == 0 {
		return nil, "", fmt.Errorf("현재 브랜치 '%s' 에 커밋이 없습니다", sourceBranch)
	}

	targetHead, err := chainClient.GetBranchCommit(targetRepoID, targetBranch)
	if err != nil {
		return nil, "", fmt.Errorf("target 브랜치 헤드 조회 실패: %w", err)
	}

	sourceRecords, err := loadBranchRecords(chainClient, sourceRepoID, sourceBranch, sourceHistoryLen.Int64())
	if err != nil {
		return nil, "", err
	}

	if !chain.IsZeroBytes20(targetHead) {
		targetHeadHex := chain.Bytes20ToGitHash(targetHead)
		targetIndex := -1
		for i, record := range sourceRecords {
			if chain.Bytes20ToGitHash(record.CommitHash) == targetHeadHex {
				targetIndex = i
				break
			}
		}
		if targetIndex == -1 {
			return nil, "", fmt.Errorf("target 브랜치 '%s' 가 source 히스토리에 없습니다. upstream이 먼저 앞서 나갔습니다", targetBranch)
		}
		if targetIndex >= len(sourceRecords)-1 {
			return nil, "", fmt.Errorf("새로 추가된 커밋이 없습니다. '%s' 는 이미 최신입니다", sourceBranch)
		}
	}

	prID, err := chainClient.CreatePullRequest(targetRepoID, targetBranch, sourceRepoID, sourceBranch)
	if err != nil {
		return nil, "", fmt.Errorf("PR 생성 실패: %w", err)
	}

	return prID, sourceBranch, nil
}

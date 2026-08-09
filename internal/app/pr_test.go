package app

import (
	"math/big"
	"strings"
	"testing"

	"github.com/opendasom/bit/internal/chain"
)

func TestPRCreateOpensFromCurrentBranch(t *testing.T) {
	repoDir := newTestRepo(t)

	sourceRepoID := big.NewInt(1)
	targetRepoID := big.NewInt(2)
	wantPRID := big.NewInt(7)

	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(1), nil
		},
		getBranchCommit: func(repoID *big.Int, branch string) ([20]byte, error) {
			return [20]byte{}, nil // target has no commits yet
		},
		getBranchCommitsWithMetadata: func(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error) {
			return []chain.BranchCommitRecord{{}}, nil
		},
		createPullRequest: func(targetRepoID *big.Int, targetBranch string, sourceRepoID *big.Int, sourceBranch string) (*big.Int, error) {
			if sourceBranch != "main" {
				t.Fatalf("expected source branch main, got %s", sourceBranch)
			}
			return wantPRID, nil
		},
	}

	prID, sourceBranch, err := PRCreate(chainClient, repoDir, sourceRepoID, targetRepoID, "main")
	if err != nil {
		t.Fatalf("PRCreate failed: %v", err)
	}
	if prID.Cmp(wantPRID) != 0 {
		t.Fatalf("expected PR ID %s, got %s", wantPRID, prID)
	}
	if sourceBranch != "main" {
		t.Fatalf("expected source branch main, got %s", sourceBranch)
	}
}

func TestPRCreateRejectsEmptySourceBranch(t *testing.T) {
	repoDir := newTestRepo(t)
	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(0), nil
		},
	}

	_, _, err := PRCreate(chainClient, repoDir, big.NewInt(1), big.NewInt(2), "main")
	if err == nil || !strings.Contains(err.Error(), "커밋이 없습니다") {
		t.Fatalf("expected empty-source-branch error, got: %v", err)
	}
}

func TestPRCreateRejectsTargetAheadOfSource(t *testing.T) {
	repoDir := newTestRepo(t)
	onlyRecord := chain.BranchCommitRecord{CommitHash: [20]byte{1, 2, 3}}
	targetHead := [20]byte{9, 9, 9} // not present in source history at all

	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(1), nil
		},
		getBranchCommit: func(repoID *big.Int, branch string) ([20]byte, error) {
			return targetHead, nil
		},
		getBranchCommitsWithMetadata: func(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error) {
			return []chain.BranchCommitRecord{onlyRecord}, nil
		},
	}

	_, _, err := PRCreate(chainClient, repoDir, big.NewInt(1), big.NewInt(2), "main")
	if err == nil || !strings.Contains(err.Error(), "앞서 나갔습니다") {
		t.Fatalf("expected target-ahead error, got: %v", err)
	}
}

func TestPRCreateRejectsWhenNoNewCommitsSinceTarget(t *testing.T) {
	repoDir := newTestRepo(t)
	targetHead := [20]byte{1, 2, 3}
	onlyRecord := chain.BranchCommitRecord{CommitHash: targetHead} // target is already the latest

	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(1), nil
		},
		getBranchCommit: func(repoID *big.Int, branch string) ([20]byte, error) {
			return targetHead, nil
		},
		getBranchCommitsWithMetadata: func(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error) {
			return []chain.BranchCommitRecord{onlyRecord}, nil
		},
	}

	_, _, err := PRCreate(chainClient, repoDir, big.NewInt(1), big.NewInt(2), "main")
	if err == nil || !strings.Contains(err.Error(), "이미 최신입니다") {
		t.Fatalf("expected already-up-to-date error, got: %v", err)
	}
}

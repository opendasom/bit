package app

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/opendasom/bit/internal/chain"
)

func TestPushRecordsNewCommitOnChain(t *testing.T) {
	repoDir := newTestRepo(t)

	var recorded bool
	var recordedBranch string
	chainClient := &fakeChainClient{
		getBranchCommit: func(repoID *big.Int, branch string) ([20]byte, error) {
			return [20]byte{}, nil // no prior push
		},
		recordCommit: func(repoID *big.Int, branch string, expectedOldCommit, commitHash, treeHash [20]byte, parentHashes [][20]byte, manifestDigest, diffDigest [32]byte) error {
			recorded = true
			recordedBranch = branch
			return nil
		},
	}
	ipfsClient := newFakeIPFSClient()

	if err := Push(chainClient, ipfsClient, repoDir, big.NewInt(1)); err != nil {
		t.Fatalf("Push failed: %v", err)
	}
	if !recorded {
		t.Fatal("expected RecordCommit to be called")
	}
	if recordedBranch != "main" {
		t.Fatalf("expected branch main, got %s", recordedBranch)
	}
}

func TestPushIsNoopWhenUpToDate(t *testing.T) {
	repoDir := newTestRepo(t)
	headHash := strings.TrimSpace(runGit(t, repoDir, "rev-parse", "HEAD"))
	headBytes, err := chain.GitHashToBytes20(headHash)
	if err != nil {
		t.Fatalf("GitHashToBytes20 failed: %v", err)
	}

	chainClient := &fakeChainClient{
		getBranchCommit: func(repoID *big.Int, branch string) ([20]byte, error) {
			return headBytes, nil
		},
		recordCommit: func(repoID *big.Int, branch string, expectedOldCommit, commitHash, treeHash [20]byte, parentHashes [][20]byte, manifestDigest, diffDigest [32]byte) error {
			t.Fatal("RecordCommit should not be called when already up to date")
			return nil
		},
	}

	if err := Push(chainClient, newFakeIPFSClient(), repoDir, big.NewInt(1)); err != nil {
		t.Fatalf("Push failed: %v", err)
	}
}

func TestPushRejectsMergeCommit(t *testing.T) {
	repoDir := newTestRepo(t) // commit A on main

	runGit(t, repoDir, "checkout", "-b", "feature")
	writeFile(t, repoDir, "feature.txt", "feature work\n")
	runGit(t, repoDir, "add", "feature.txt")
	runGit(t, repoDir, "commit", "-m", "feature commit") // commit B

	runGit(t, repoDir, "checkout", "main")
	writeFile(t, repoDir, "main.txt", "main work\n")
	runGit(t, repoDir, "add", "main.txt")
	runGit(t, repoDir, "commit", "-m", "main commit") // commit C

	runGit(t, repoDir, "merge", "--no-ff", "-m", "merge feature", "feature") // commit M, 2 parents

	var recordCount int
	chainClient := &fakeChainClient{
		getBranchCommit: func(repoID *big.Int, branch string) ([20]byte, error) {
			return [20]byte{}, nil
		},
		recordCommit: func(repoID *big.Int, branch string, expectedOldCommit, commitHash, treeHash [20]byte, parentHashes [][20]byte, manifestDigest, diffDigest [32]byte) error {
			recordCount++
			return nil
		},
	}

	err := Push(chainClient, newFakeIPFSClient(), repoDir, big.NewInt(1))
	if err == nil {
		t.Fatal("expected Push to reject a merge commit")
	}
	if !strings.Contains(err.Error(), "merge commit") {
		t.Fatalf("expected merge commit error, got: %v", err)
	}
	// A, C, and B precede the merge commit M in rev-list order; only they
	// should have been recorded before Push stopped at M.
	if recordCount != 3 {
		t.Fatalf("expected 3 commits recorded before the merge commit halted push, got %d", recordCount)
	}
}

func TestPushPropagatesChainLookupError(t *testing.T) {
	repoDir := newTestRepo(t)
	wantErr := errors.New("rpc unavailable")

	chainClient := &fakeChainClient{
		getBranchCommit: func(repoID *big.Int, branch string) ([20]byte, error) {
			return [20]byte{}, wantErr
		},
	}

	err := Push(chainClient, newFakeIPFSClient(), repoDir, big.NewInt(1))
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestPushStopsAfterRecordCommitFailure(t *testing.T) {
	repoDir := newTestRepo(t)
	writeFile(t, repoDir, "second.txt", "second\n")
	runGit(t, repoDir, "add", "second.txt")
	runGit(t, repoDir, "commit", "-m", "second commit")

	wantErr := errors.New("someone else pushed first")
	var recordCount int
	chainClient := &fakeChainClient{
		getBranchCommit: func(repoID *big.Int, branch string) ([20]byte, error) {
			return [20]byte{}, nil
		},
		recordCommit: func(repoID *big.Int, branch string, expectedOldCommit, commitHash, treeHash [20]byte, parentHashes [][20]byte, manifestDigest, diffDigest [32]byte) error {
			recordCount++
			return wantErr
		},
	}

	err := Push(chainClient, newFakeIPFSClient(), repoDir, big.NewInt(1))
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
	if recordCount != 1 {
		t.Fatalf("expected push to stop after the first RecordCommit failure, got %d calls", recordCount)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(fmt.Errorf("write %s: %w", name, err))
	}
}

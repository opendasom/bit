package app

import (
	"math/big"
	"strings"
	"testing"

	"github.com/opendasom/bit/internal/chain"
)

func TestPullReconstructsRemoteCommit(t *testing.T) {
	src := newTestRepo(t)
	commitHash := strings.TrimSpace(runGit(t, src, "rev-parse", "HEAD"))

	ipfsClient := newFakeIPFSClient()
	record := uploadCommitRecord(t, ipfsClient, src, "main", commitHash)

	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(1), nil
		},
		getBranchCommitsWithMetadata: func(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error) {
			return []chain.BranchCommitRecord{record}, nil
		},
	}

	dst := newEmptyRepoDir(t)
	if err := Pull(chainClient, ipfsClient, dst, big.NewInt(1), "main"); err != nil {
		t.Fatalf("Pull failed: %v", err)
	}

	got := strings.TrimSpace(runGit(t, dst, "rev-parse", "HEAD"))
	if got != commitHash {
		t.Fatalf("expected HEAD %s, got %s", commitHash, got)
	}
}

func TestPullRejectsBranchNeverPushed(t *testing.T) {
	dst := newEmptyRepoDir(t)
	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(0), nil
		},
	}

	err := Pull(chainClient, newFakeIPFSClient(), dst, big.NewInt(1), "main")
	if err == nil || !strings.Contains(err.Error(), "push된 적 없습니다") {
		t.Fatalf("expected 'never pushed' error, got: %v", err)
	}
}

func TestPullRejectsDivergedLocalHead(t *testing.T) {
	src := newTestRepo(t)
	remoteCommitHash := strings.TrimSpace(runGit(t, src, "rev-parse", "HEAD"))

	ipfsClient := newFakeIPFSClient()
	record := uploadCommitRecord(t, ipfsClient, src, "main", remoteCommitHash)

	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(1), nil
		},
		getBranchCommitsWithMetadata: func(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error) {
			return []chain.BranchCommitRecord{record}, nil
		},
	}

	// dst has its own unrelated history (a second commit on top of the same
	// base), so its HEAD can't be found in the remote's recorded history.
	dst := newTestRepo(t)
	writeFile(t, dst, "unrelated.txt", "local-only work\n")
	runGit(t, dst, "add", "unrelated.txt")
	runGit(t, dst, "commit", "-m", "unrelated local commit")

	err := Pull(chainClient, ipfsClient, dst, big.NewInt(1), "main")
	if err == nil || !strings.Contains(err.Error(), "히스토리에 없습니다") {
		t.Fatalf("expected diverged-history error, got: %v", err)
	}
}

func TestPullRejectsManifestCommitMismatch(t *testing.T) {
	src := newTestRepo(t)
	commitHash := strings.TrimSpace(runGit(t, src, "rev-parse", "HEAD"))

	ipfsClient := newFakeIPFSClient()
	record := uploadCommitRecord(t, ipfsClient, src, "main", commitHash)
	// Corrupt the on-chain record so it no longer matches the manifest
	// that was actually uploaded (simulates a tampered/incorrect record).
	record.CommitHash[0] ^= 0xFF

	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(1), nil
		},
		getBranchCommitsWithMetadata: func(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error) {
			return []chain.BranchCommitRecord{record}, nil
		},
	}

	dst := newEmptyRepoDir(t)
	err := Pull(chainClient, ipfsClient, dst, big.NewInt(1), "main")
	if err == nil || !strings.Contains(err.Error(), "commit mismatch") {
		t.Fatalf("expected commit mismatch error, got: %v", err)
	}
}

func TestPullIsNoopWhenLocalHeadIsLatest(t *testing.T) {
	src := newTestRepo(t)
	commitHash := strings.TrimSpace(runGit(t, src, "rev-parse", "HEAD"))

	ipfsClient := newFakeIPFSClient()
	record := uploadCommitRecord(t, ipfsClient, src, "main", commitHash)

	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(1), nil
		},
		getBranchCommitsWithMetadata: func(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error) {
			return []chain.BranchCommitRecord{record}, nil
		},
	}

	// dst already has exactly the commit the remote knows about.
	dst := src
	if err := Pull(chainClient, ipfsClient, dst, big.NewInt(1), "main"); err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
}

package app

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/opendasom/bit/internal/chain"
)

func TestForkRestoresCommitUnderNewRepoID(t *testing.T) {
	src := newTestRepo(t)
	commitHash := strings.TrimSpace(runGit(t, src, "rev-parse", "HEAD"))

	ipfsClient := newFakeIPFSClient()
	record := uploadCommitRecord(t, ipfsClient, src, "main", commitHash)

	newRepoID := big.NewInt(2)
	var recordedUnderRepoID *big.Int
	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(1), nil
		},
		getBranchCommitsWithMetadata: func(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error) {
			return []chain.BranchCommitRecord{record}, nil
		},
		createRepo: func(metadataCID string) (*big.Int, error) {
			return newRepoID, nil
		},
		recordCommit: func(repoID *big.Int, branch string, expectedOldCommit, commitHash, treeHash [20]byte, parentHashes [][20]byte, manifestDigest, diffDigest [32]byte) error {
			recordedUnderRepoID = repoID
			return nil
		},
	}

	dst := newEmptyRepoDir(t)
	forkRepoID, err := Fork(chainClient, ipfsClient, dst, big.NewInt(1), "main")
	if err != nil {
		t.Fatalf("Fork failed: %v", err)
	}
	if forkRepoID.Cmp(newRepoID) != 0 {
		t.Fatalf("expected fork repo ID %s, got %s", newRepoID, forkRepoID)
	}
	if recordedUnderRepoID == nil || recordedUnderRepoID.Cmp(newRepoID) != 0 {
		t.Fatalf("expected RecordCommit to use new repo ID %s, got %v", newRepoID, recordedUnderRepoID)
	}

	got := strings.TrimSpace(runGit(t, dst, "rev-parse", "HEAD"))
	if got != commitHash {
		t.Fatalf("expected HEAD %s, got %s", commitHash, got)
	}
}

func TestForkRejectsEmptySourceBranch(t *testing.T) {
	dst := newEmptyRepoDir(t)
	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(0), nil
		},
	}

	_, err := Fork(chainClient, newFakeIPFSClient(), dst, big.NewInt(1), "main")
	if err == nil || !strings.Contains(err.Error(), "커밋이 없습니다") {
		t.Fatalf("expected empty-source-branch error, got: %v", err)
	}
}

func TestForkPropagatesCreateRepoFailure(t *testing.T) {
	src := newTestRepo(t)
	commitHash := strings.TrimSpace(runGit(t, src, "rev-parse", "HEAD"))

	ipfsClient := newFakeIPFSClient()
	record := uploadCommitRecord(t, ipfsClient, src, "main", commitHash)
	wantErr := errors.New("chain unavailable")

	chainClient := &fakeChainClient{
		getBranchHistoryLength: func(repoID *big.Int, branch string) (*big.Int, error) {
			return big.NewInt(1), nil
		},
		getBranchCommitsWithMetadata: func(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error) {
			return []chain.BranchCommitRecord{record}, nil
		},
		createRepo: func(metadataCID string) (*big.Int, error) {
			return nil, wantErr
		},
	}

	dst := newEmptyRepoDir(t)
	_, err := Fork(chainClient, ipfsClient, dst, big.NewInt(1), "main")
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
}

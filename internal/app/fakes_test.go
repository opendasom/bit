package app

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/opendasom/bit/internal/chain"
	compactcid "github.com/opendasom/bit/internal/cid"
	"github.com/opendasom/bit/internal/git"
	"github.com/opendasom/bit/internal/manifest"
)

// fakeChainClient implements ChainClient with per-method overrides so each
// test only wires up the calls it actually exercises.
type fakeChainClient struct {
	getBranchCommit              func(repoID *big.Int, branch string) ([20]byte, error)
	getBranchHistoryLength       func(repoID *big.Int, branch string) (*big.Int, error)
	getBranchCommitsWithMetadata func(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error)
	recordCommit                 func(repoID *big.Int, branch string, expectedOldCommit, commitHash, treeHash [20]byte, parentHashes [][20]byte, manifestDigest, diffDigest [32]byte) error
	createRepo                   func(metadataCID string) (*big.Int, error)
	createPullRequest            func(targetRepoID *big.Int, targetBranch string, sourceRepoID *big.Int, sourceBranch string) (*big.Int, error)
}

func (f *fakeChainClient) GetBranchCommit(repoID *big.Int, branch string) ([20]byte, error) {
	if f.getBranchCommit == nil {
		return [20]byte{}, nil
	}
	return f.getBranchCommit(repoID, branch)
}

func (f *fakeChainClient) GetBranchHistoryLength(repoID *big.Int, branch string) (*big.Int, error) {
	if f.getBranchHistoryLength == nil {
		return big.NewInt(0), nil
	}
	return f.getBranchHistoryLength(repoID, branch)
}

func (f *fakeChainClient) GetBranchCommitsWithMetadata(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error) {
	if f.getBranchCommitsWithMetadata == nil {
		return nil, fmt.Errorf("getBranchCommitsWithMetadata not stubbed")
	}
	return f.getBranchCommitsWithMetadata(repoID, branch, start, limit)
}

func (f *fakeChainClient) RecordCommit(
	repoID *big.Int,
	branch string,
	expectedOldCommit [20]byte,
	commitHash [20]byte,
	treeHash [20]byte,
	parentHashes [][20]byte,
	manifestDigest [32]byte,
	diffDigest [32]byte,
) error {
	if f.recordCommit == nil {
		return nil
	}
	return f.recordCommit(repoID, branch, expectedOldCommit, commitHash, treeHash, parentHashes, manifestDigest, diffDigest)
}

func (f *fakeChainClient) CreateRepo(metadataCID string) (*big.Int, error) {
	if f.createRepo == nil {
		return nil, fmt.Errorf("createRepo not stubbed")
	}
	return f.createRepo(metadataCID)
}

func (f *fakeChainClient) CreatePullRequest(targetRepoID *big.Int, targetBranch string, sourceRepoID *big.Int, sourceBranch string) (*big.Int, error) {
	if f.createPullRequest == nil {
		return nil, fmt.Errorf("createPullRequest not stubbed")
	}
	return f.createPullRequest(targetRepoID, targetBranch, sourceRepoID, sourceBranch)
}

// fakeIPFSClient is an in-memory content store keyed by a real CIDv0
// (sha2-256 of the uploaded bytes), so compactcid round-trips exactly like
// they would against a real Kubo node.
type fakeIPFSClient struct {
	store     map[string][]byte
	uploadErr error
}

func newFakeIPFSClient() *fakeIPFSClient {
	return &fakeIPFSClient{store: make(map[string][]byte)}
}

func (f *fakeIPFSClient) Upload(data []byte) (string, error) {
	if f.uploadErr != nil {
		return "", f.uploadErr
	}
	digest := sha256.Sum256(data)
	cidStr := compactcid.CIDV0FromDigest(digest)
	f.store[cidStr] = append([]byte(nil), data...)
	return cidStr, nil
}

func (f *fakeIPFSClient) Download(cidStr string) ([]byte, error) {
	data, ok := f.store[cidStr]
	if !ok {
		return nil, fmt.Errorf("fakeIPFSClient: no such CID %s", cidStr)
	}
	return data, nil
}

// newTestRepo creates a temp git repo on branch "main" with a single commit.
func newTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "checkout", "-b", "main")
	runGit(t, dir, "config", "user.name", "Tester")
	runGit(t, dir, "config", "user.email", "tester@example.test")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial commit")
	return dir
}

// newEmptyRepoDir creates a bare-uninitialized-HEAD git repo directory
// suitable as a Pull/Fork destination.
func newEmptyRepoDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	if err := os.MkdirAll(filepath.Join(dir, ".bit"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".bit", "config.json"), []byte("{}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return dir
}

// uploadCommitRecord reads commitHash out of the repo at repoPath, uploads
// its diff+manifest to ipfsClient, and returns the chain.BranchCommitRecord
// a real push would have recorded for it.
func uploadCommitRecord(t *testing.T, ipfsClient *fakeIPFSClient, repoPath, branch, commitHash string) chain.BranchCommitRecord {
	t.Helper()
	info, err := git.ReadCommit(repoPath, commitHash)
	if err != nil {
		t.Fatalf("ReadCommit failed: %v", err)
	}
	diff, err := git.ExtractCommitDiff(repoPath, info)
	if err != nil {
		t.Fatalf("ExtractCommitDiff failed: %v", err)
	}
	diffCID, err := ipfsClient.Upload(diff)
	if err != nil {
		t.Fatalf("upload diff failed: %v", err)
	}
	m := git.ManifestForCommit(branch, info, diffCID)
	manifestData, err := manifest.Encode(m)
	if err != nil {
		t.Fatalf("encode manifest failed: %v", err)
	}
	manifestCID, err := ipfsClient.Upload(manifestData)
	if err != nil {
		t.Fatalf("upload manifest failed: %v", err)
	}

	commitHash20, err := chain.GitHashToBytes20(info.Hash)
	if err != nil {
		t.Fatalf("GitHashToBytes20(commit) failed: %v", err)
	}
	treeHash20, err := chain.GitHashToBytes20(info.TreeHash)
	if err != nil {
		t.Fatalf("GitHashToBytes20(tree) failed: %v", err)
	}
	manifestDigest, err := compactcid.DigestFromCIDV0(manifestCID)
	if err != nil {
		t.Fatalf("DigestFromCIDV0(manifest) failed: %v", err)
	}
	diffDigest, err := compactcid.DigestFromCIDV0(diffCID)
	if err != nil {
		t.Fatalf("DigestFromCIDV0(diff) failed: %v", err)
	}

	return chain.BranchCommitRecord{
		CommitHash:     commitHash20,
		TreeHash:       treeHash20,
		ManifestDigest: manifestDigest,
		DiffDigest:     diffDigest,
	}
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
	return string(out)
}

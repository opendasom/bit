package app

import (
	"errors"
	"math/big"
	"os/exec"
	"strings"
	"testing"

	"github.com/opendasom/bit/internal/chain"
	compactcid "github.com/opendasom/bit/internal/cid"
	"github.com/opendasom/bit/internal/git"
	"github.com/opendasom/bit/internal/manifest"
)

type pullChainFake struct {
	record chain.BranchCommitRecord
}

func (f pullChainFake) GetBranchCommit(*big.Int, string) ([20]byte, error) {
	return f.record.CommitHash, nil
}
func (f pullChainFake) GetBranchHistoryLength(*big.Int, string) (*big.Int, error) {
	return big.NewInt(1), nil
}
func (f pullChainFake) GetBranchCommitsWithMetadata(*big.Int, string, *big.Int, *big.Int) ([]chain.BranchCommitRecord, error) {
	return []chain.BranchCommitRecord{f.record}, nil
}
func (pullChainFake) RecordCommit(*big.Int, string, [20]byte, [20]byte, [20]byte, [][20]byte, [32]byte, [32]byte) error {
	return errors.New("not implemented")
}
func (pullChainFake) CreateRepo(string) (*big.Int, error) {
	return nil, errors.New("not implemented")
}

type pullIPFSFake map[string][]byte

func (pullIPFSFake) Upload([]byte) (string, error) { return "", errors.New("not implemented") }
func (f pullIPFSFake) Download(cid string) ([]byte, error) {
	data, ok := f[cid]
	if !ok {
		return nil, errors.New("missing CID")
	}
	return data, nil
}

func TestPullRejectsManifestBranchBeforeMutatingRepository(t *testing.T) {
	repoPath := t.TempDir()
	command := exec.Command("git", "init")
	command.Dir = repoPath
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}

	var commitHash [20]byte
	commitHash[0] = 1
	var treeHash [20]byte
	treeHash[0] = 2
	var manifestDigest [32]byte
	manifestDigest[0] = 3
	var diffDigest [32]byte
	diffDigest[0] = 4
	record := chain.BranchCommitRecord{
		CommitHash: commitHash, TreeHash: treeHash, ManifestDigest: manifestDigest, DiffDigest: diffDigest,
	}
	m := &manifest.Manifest{
		Version: manifest.VersionCommitDiff, Storage: manifest.StorageGitDiff, DiffAlgorithm: manifest.DiffBinaryPatch,
		Branch: "unexpected", DiffCID: compactcid.CIDV0FromDigest(diffDigest), GitCommit: chain.Bytes20ToGitHash(commitHash),
		TreeHash: chain.Bytes20ToGitHash(treeHash),
	}
	manifestData, err := manifest.Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	ipfs := pullIPFSFake{compactcid.CIDV0FromDigest(manifestDigest): manifestData}
	err = Pull(pullChainFake{record: record}, ipfs, repoPath, big.NewInt(1), "main")
	if err == nil || !strings.Contains(err.Error(), "manifest branch mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
	if git.HasHead(repoPath) {
		t.Fatal("pull mutated HEAD before validation completed")
	}
}

type oversizedHistoryChainFake struct{ pullChainFake }

func (oversizedHistoryChainFake) GetBranchHistoryLength(*big.Int, string) (*big.Int, error) {
	return new(big.Int).Lsh(big.NewInt(1), 255), nil
}

func TestPullRejectsOversizedHistoryLength(t *testing.T) {
	err := Pull(oversizedHistoryChainFake{}, pullIPFSFake{}, t.TempDir(), big.NewInt(1), "main")
	if err == nil || !strings.Contains(err.Error(), "out of the supported range") {
		t.Fatalf("unexpected error: %v", err)
	}
}

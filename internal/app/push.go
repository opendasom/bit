package app

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/chain"
	compactcid "github.com/opendasom/bit/internal/cid"
	"github.com/opendasom/bit/internal/git"
	"github.com/opendasom/bit/internal/manifest"
)

// Push uploads and records commits missing from the remote branch.
func Push(chainClient ChainClient, ipfsClient IPFSClient, repoPath string, repoID *big.Int) error {
	branch, err := git.CurrentBranch(repoPath)
	if err != nil {
		return fmt.Errorf("failed to read git info: %w", err)
	}

	expectedOldCommit, err := chainClient.GetBranchCommit(repoID, branch)
	if err != nil {
		return fmt.Errorf("branch head lookup failed: %w", err)
	}
	baseCommit := ""
	if !chain.IsZeroBytes20(expectedOldCommit) {
		baseCommit = chain.Bytes20ToGitHash(expectedOldCommit)
	}

	commits, err := git.CommitsAfter(repoPath, baseCommit)
	if err != nil {
		return fmt.Errorf("failed to compute commits to push: %w", err)
	}
	if len(commits) == 0 {
		fmt.Printf("%s is already up to date\n", branch)
		return nil
	}

	fmt.Printf("branch: %s, commits to upload: %d\n", branch, len(commits))
	fmt.Println("pre-validation complete, uploading diffs...")

	for _, commit := range commits {
		info, err := git.ReadCommit(repoPath, commit)
		if err != nil {
			return fmt.Errorf("failed to read commit metadata (%s): %w", commit, err)
		}
		if len(info.ParentHashes) > 1 {
			return fmt.Errorf("merge commit diff push is not supported yet: %s", info.Hash)
		}

		diff, err := git.ExtractCommitDiff(repoPath, info)
		if err != nil {
			return fmt.Errorf("failed to extract commit diff (%s): %w", info.Hash, err)
		}
		diffCID, err := ipfsClient.Upload(diff)
		if err != nil {
			return fmt.Errorf("diff IPFS upload failed (%s): %w", info.Hash, err)
		}
		diffDigest, err := compactcid.DigestFromCIDV0(diffCID)
		if err != nil {
			return fmt.Errorf("failed to compress diff CID (%s): %w", diffCID, err)
		}

		m := git.ManifestForCommit(branch, info, diffCID)
		manifestData, err := manifest.Encode(m)
		if err != nil {
			return fmt.Errorf("failed to build manifest (%s): %w", info.Hash, err)
		}
		manifestCID, err := ipfsClient.Upload(manifestData)
		if err != nil {
			return fmt.Errorf("manifest IPFS upload failed (%s): %w", info.Hash, err)
		}
		manifestDigest, err := compactcid.DigestFromCIDV0(manifestCID)
		if err != nil {
			return fmt.Errorf("failed to compress manifest CID (%s): %w", manifestCID, err)
		}

		commitHash, err := chain.GitHashToBytes20(info.Hash)
		if err != nil {
			return fmt.Errorf("failed to convert commit hash (%s): %w", info.Hash, err)
		}
		treeHash, err := chain.GitHashToBytes20(info.TreeHash)
		if err != nil {
			return fmt.Errorf("failed to convert tree hash (%s): %w", info.TreeHash, err)
		}
		parentHashes := make([][20]byte, 0, len(info.ParentHashes))
		for _, parent := range info.ParentHashes {
			parentHash, err := chain.GitHashToBytes20(parent)
			if err != nil {
				return fmt.Errorf("failed to convert parent commit hash (%s): %w", parent, err)
			}
			parentHashes = append(parentHashes, parentHash)
		}

		if err := chainClient.RecordCommit(
			repoID,
			branch,
			expectedOldCommit,
			commitHash,
			treeHash,
			parentHashes,
			manifestDigest,
			diffDigest,
		); err != nil {
			return fmt.Errorf("failed to record commit on-chain (someone else may have pushed first, commit %s): %w", info.Hash, err)
		}

		expectedOldCommit = commitHash
		fmt.Printf("pushed commit: %s diff=%s manifest=%s\n", info.Hash[:8], diffCID, manifestCID)
	}

	fmt.Printf("push complete: %s -> %s\n", branch, commits[len(commits)-1][:8])
	return nil
}

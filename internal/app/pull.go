package app

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/chain"
	compactcid "github.com/opendasom/bit/internal/cid"
	"github.com/opendasom/bit/internal/git"
	"github.com/opendasom/bit/internal/manifest"
)

type pendingCommit struct {
	record   chain.BranchCommitRecord
	manifest *manifest.Manifest
	diff     []byte
}

// Pull verifies and applies commits missing from the local branch.
func Pull(chainClient ChainClient, ipfsClient IPFSClient, repoPath string, repoID *big.Int, branch string) error {
	historyLen, err := chainClient.GetBranchHistoryLength(repoID, branch)
	if err != nil {
		return fmt.Errorf("branch history lookup failed: %w", err)
	}
	if historyLen.Sign() == 0 {
		return fmt.Errorf("branch '%s' has not been pushed yet", branch)
	}
	if historyLen.Sign() < 0 || !historyLen.IsInt64() || historyLen.BitLen() > 63 {
		return fmt.Errorf("branch history length is out of the supported range: %s", historyLen.String())
	}
	total := historyLen.Int64()
	if int64(int(total)) != total {
		return fmt.Errorf("branch history length exceeds this platform's int range: %s", historyLen.String())
	}
	records, err := loadBranchRecords(chainClient, repoID, branch, int(total))
	if err != nil {
		return err
	}

	start := 0
	if git.HasHead(repoPath) {
		localHead, err := git.CurrentHead(repoPath)
		if err != nil {
			return fmt.Errorf("local HEAD lookup failed: %w", err)
		}
		found := false
		for i, record := range records {
			if chain.Bytes20ToGitHash(record.CommitHash) == localHead {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("local HEAD %s is not in remote branch '%s' history", localHead[:8], branch)
		}
	}
	if start >= len(records) {
		fmt.Printf("%s is already up to date\n", branch)
		return nil
	}
	if err := git.EnsureCleanWorktree(repoPath); err != nil {
		return err
	}

	pending := make([]pendingCommit, 0, len(records)-start)
	for index := start; index < len(records); index++ {
		record := records[index]
		manifestCID := compactcid.CIDV0FromDigest(record.ManifestDigest)
		diffCID := compactcid.CIDV0FromDigest(record.DiffDigest)

		manifestData, err := ipfsClient.Download(manifestCID)
		if err != nil {
			return fmt.Errorf("manifest download failed (%s): %w", manifestCID, err)
		}
		m, err := manifest.Decode(manifestData)
		if err != nil {
			return fmt.Errorf("manifest parse failed (%s): %w", manifestCID, err)
		}

		expectedCommit := chain.Bytes20ToGitHash(record.CommitHash)
		if m.GitCommit != expectedCommit {
			return fmt.Errorf("manifest commit mismatch: got %s, want %s", m.GitCommit, expectedCommit)
		}
		if m.DiffCID != diffCID {
			return fmt.Errorf("manifest diff CID mismatch for %s", expectedCommit)
		}
		if m.TreeHash != chain.Bytes20ToGitHash(record.TreeHash) {
			return fmt.Errorf("manifest tree mismatch for %s", expectedCommit)
		}
		if m.Branch != branch {
			return fmt.Errorf("manifest branch mismatch for %s: got %q, want %q", expectedCommit, m.Branch, branch)
		}
		if index == 0 {
			if len(m.ParentCommits) != 0 {
				return fmt.Errorf("root commit %s unexpectedly has parents", expectedCommit)
			}
		} else {
			expectedParent := chain.Bytes20ToGitHash(records[index-1].CommitHash)
			if len(m.ParentCommits) != 1 || m.ParentCommits[0] != expectedParent {
				return fmt.Errorf("manifest parent mismatch for %s: want %s", expectedCommit, expectedParent)
			}
		}

		diff, err := ipfsClient.Download(diffCID)
		if err != nil {
			return fmt.Errorf("diff download failed (%s): %w", m.DiffCID, err)
		}
		pending = append(pending, pendingCommit{record: record, manifest: m, diff: diff})
	}

	fmt.Printf("branch: %s, verified commits: %d\n", branch, len(pending))
	for _, item := range pending {
		expectedCommit := chain.Bytes20ToGitHash(item.record.CommitHash)
		if _, err := git.BuildCommitDiff(repoPath, item.manifest, item.diff); err != nil {
			return fmt.Errorf("commit diff reconstruction failed (%s): %w", expectedCommit, err)
		}
		fmt.Printf("verified commit: %s diff=%s\n", expectedCommit[:8], item.manifest.DiffCID)
	}
	finalCommit := chain.Bytes20ToGitHash(pending[len(pending)-1].record.CommitHash)
	if err := git.CheckoutBranch(repoPath, branch, finalCommit); err != nil {
		return fmt.Errorf("branch checkout failed (%s): %w", branch, err)
	}

	fmt.Printf("pull complete: %s\n", branch)
	return nil
}

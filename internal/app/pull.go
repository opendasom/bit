package app

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/chain"
	compactcid "github.com/opendasom/bit/internal/cid"
	"github.com/opendasom/bit/internal/git"
	"github.com/opendasom/bit/internal/manifest"
)

// pendingCommit is a commit that has been downloaded and verified against
// on-chain metadata but not yet applied to the local git repository.
type pendingCommit struct {
	record   chain.BranchCommitRecord
	manifest *manifest.Manifest
	diff     []byte
}

// Pull fetches commits recorded on-chain for the given branch that are
// missing locally, verifies them against IPFS, and reconstructs them in the
// local git repository.
func Pull(chainClient ChainClient, ipfsClient IPFSClient, repoPath string, repoID *big.Int, branch string) error {
	historyLen, err := chainClient.GetBranchHistoryLength(repoID, branch)
	if err != nil {
		return fmt.Errorf("브랜치 히스토리 조회 실패: %w", err)
	}
	if historyLen.Sign() == 0 {
		return fmt.Errorf("브랜치 '%s' 가 아직 push된 적 없습니다", branch)
	}
	if historyLen.Sign() < 0 || !historyLen.IsInt64() || historyLen.BitLen() > 63 {
		return fmt.Errorf("브랜치 히스토리 길이가 지원 범위를 벗어났습니다: %s", historyLen.String())
	}
	total := historyLen.Int64()
	if int64(int(total)) != total {
		return fmt.Errorf("브랜치 히스토리 길이가 로컬 플랫폼 범위를 벗어났습니다: %s", historyLen.String())
	}
	records, err := loadBranchRecords(chainClient, repoID, branch, int(total))
	if err != nil {
		return err
	}

	start := 0
	if git.HasHead(repoPath) {
		localHead, err := git.CurrentHead(repoPath)
		if err != nil {
			return fmt.Errorf("로컬 HEAD 조회 실패: %w", err)
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
			return fmt.Errorf("로컬 HEAD %s 는 원격 브랜치 '%s' 히스토리에 없습니다", localHead[:8], branch)
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
			return fmt.Errorf("manifest 다운로드 실패 (%s): %w", manifestCID, err)
		}
		m, err := manifest.Decode(manifestData)
		if err != nil {
			return fmt.Errorf("manifest 파싱 실패 (%s): %w", manifestCID, err)
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
			return fmt.Errorf("diff 다운로드 실패 (%s): %w", m.DiffCID, err)
		}
		pending = append(pending, pendingCommit{record: record, manifest: m, diff: diff})
	}

	fmt.Printf("브랜치: %s, 검증된 커밋: %d개\n", branch, len(pending))
	for _, item := range pending {
		expectedCommit := chain.Bytes20ToGitHash(item.record.CommitHash)
		if _, err := git.BuildCommitDiff(repoPath, item.manifest, item.diff); err != nil {
			return fmt.Errorf("commit diff 재구성 실패 (%s): %w", expectedCommit, err)
		}
		fmt.Printf("commit 검증 완료: %s diff=%s\n", expectedCommit[:8], item.manifest.DiffCID)
	}
	finalCommit := chain.Bytes20ToGitHash(pending[len(pending)-1].record.CommitHash)
	if err := git.CheckoutBranch(repoPath, branch, finalCommit); err != nil {
		return fmt.Errorf("브랜치 전환 실패 (%s): %w", branch, err)
	}

	fmt.Printf("pull 완료: %s\n", branch)
	return nil
}

package app

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/chain"
	compactcid "github.com/opendasom/bit/internal/cid"
	"github.com/opendasom/bit/internal/git"
	"github.com/opendasom/bit/internal/manifest"
)

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
	records, err := loadBranchRecords(chainClient, repoID, branch, historyLen.Int64())
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

	fmt.Printf("브랜치: %s, 적용 커밋: %d개\n", branch, len(records)-start)
	for _, record := range records[start:] {
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

		diff, err := ipfsClient.Download(diffCID)
		if err != nil {
			return fmt.Errorf("diff 다운로드 실패 (%s): %w", m.DiffCID, err)
		}
		if err := git.ApplyCommitDiff(repoPath, m, diff); err != nil {
			return fmt.Errorf("commit diff 적용 실패 (%s): %w", expectedCommit, err)
		}
		fmt.Printf("commit pull 완료: %s diff=%s\n", expectedCommit[:8], m.DiffCID)
	}

	fmt.Printf("pull 완료: %s\n", branch)
	return nil
}

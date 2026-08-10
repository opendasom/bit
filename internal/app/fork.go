package app

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/chain"
	compactcid "github.com/opendasom/bit/internal/cid"
	"github.com/opendasom/bit/internal/git"
	"github.com/opendasom/bit/internal/manifest"
)

// Fork duplicates a source repository's branch history onto a freshly
// created on-chain repo: it downloads and verifies every commit from IPFS,
// restores it into the local git repository at repoPath, and re-records it
// under the new repo ID (reusing the existing IPFS digests, no re-upload).
// It returns the newly created fork's repo ID.
func Fork(chainClient ChainClient, ipfsClient IPFSClient, repoPath string, sourceRepoID *big.Int, branch string) (*big.Int, error) {
	historyLen, err := chainClient.GetBranchHistoryLength(sourceRepoID, branch)
	if err != nil {
		return nil, fmt.Errorf("source 브랜치 히스토리 조회 실패: %w", err)
	}
	if historyLen.Sign() == 0 {
		return nil, fmt.Errorf("source 브랜치 '%s'에 커밋이 없습니다", branch)
	}
	records, err := loadBranchRecords(chainClient, sourceRepoID, branch, historyLen.Int64())
	if err != nil {
		return nil, err
	}
	fmt.Printf("source 커밋 %d개 발견\n", len(records))

	fmt.Println("fork 저장소를 체인에 생성 중...")
	forkRepoID, err := chainClient.CreateRepo("")
	if err != nil {
		return nil, fmt.Errorf("fork 저장소 생성 실패: %w", err)
	}
	fmt.Printf("fork 저장소 생성 완료 (repoId: %s)\n", forkRepoID.String())

	expectedOldCommit := [20]byte{}

	for i, record := range records {
		manifestCID := compactcid.CIDV0FromDigest(record.ManifestDigest)
		diffCID := compactcid.CIDV0FromDigest(record.DiffDigest)

		manifestData, err := ipfsClient.Download(manifestCID)
		if err != nil {
			return nil, fmt.Errorf("manifest 다운로드 실패 (커밋 %d, %s): %w", i+1, manifestCID, err)
		}
		m, err := manifest.Decode(manifestData)
		if err != nil {
			return nil, fmt.Errorf("manifest 파싱 실패 (%s): %w", manifestCID, err)
		}

		expectedCommit := chain.Bytes20ToGitHash(record.CommitHash)
		if m.GitCommit != expectedCommit {
			return nil, fmt.Errorf("manifest commit mismatch: got %s, want %s", m.GitCommit, expectedCommit)
		}
		if m.DiffCID != diffCID {
			return nil, fmt.Errorf("manifest diff CID mismatch for %s", expectedCommit)
		}

		diff, err := ipfsClient.Download(m.DiffCID)
		if err != nil {
			return nil, fmt.Errorf("diff 다운로드 실패 (%s): %w", m.DiffCID, err)
		}

		if err := git.ApplyCommitDiff(repoPath, m, diff); err != nil {
			return nil, fmt.Errorf("commit diff 적용 실패 (%s): %w", expectedCommit, err)
		}

		parentHashes := make([][20]byte, 0, len(m.ParentCommits))
		for _, parent := range m.ParentCommits {
			h, err := chain.GitHashToBytes20(parent)
			if err != nil {
				return nil, fmt.Errorf("parent 해시 변환 실패 (%s): %w", parent, err)
			}
			parentHashes = append(parentHashes, h)
		}

		if err := chainClient.RecordCommit(
			forkRepoID,
			branch,
			expectedOldCommit,
			record.CommitHash,
			record.TreeHash,
			parentHashes,
			record.ManifestDigest,
			record.DiffDigest,
		); err != nil {
			return nil, fmt.Errorf("체인 커밋 기록 실패 (커밋 %d, %s): %w", i+1, expectedCommit, err)
		}

		expectedOldCommit = record.CommitHash
		fmt.Printf("커밋 복원 완료 (%d/%d): %s\n", i+1, len(records), expectedCommit[:8])
	}

	return forkRepoID, nil
}

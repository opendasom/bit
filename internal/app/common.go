package app

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/chain"
)

func loadBranchRecords(chainClient ChainClient, repoID *big.Int, branch string, total int64) ([]chain.BranchCommitRecord, error) {
	const pageSize int64 = 100
	records := make([]chain.BranchCommitRecord, 0, total)
	for start := int64(0); start < total; start += pageSize {
		limit := pageSize
		if remaining := total - start; remaining < limit {
			limit = remaining
		}
		page, err := chainClient.GetBranchCommitsWithMetadata(repoID, branch, big.NewInt(start), big.NewInt(limit))
		if err != nil {
			return nil, fmt.Errorf("브랜치 히스토리 페이지 조회 실패 (%d..%d): %w", start, start+limit, err)
		}
		records = append(records, page...)
	}
	return records, nil
}

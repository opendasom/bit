package app

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/chain"
)

// loadBranchRecords fetches every commit record for a branch from the chain,
// paging through GetBranchCommitsWithMetadata since contracts typically cap
// the number of records returned per call.
func loadBranchRecords(chainClient ChainClient, repoID *big.Int, branch string, total int) ([]chain.BranchCommitRecord, error) {
	const pageSize = 100
	records := make([]chain.BranchCommitRecord, 0, total)
	for start := 0; start < total; start += pageSize {
		limit := pageSize
		if remaining := total - start; remaining < limit {
			limit = remaining
		}
		page, err := chainClient.GetBranchCommitsWithMetadata(repoID, branch, big.NewInt(int64(start)), big.NewInt(int64(limit)))
		if err != nil {
			return nil, fmt.Errorf("branch history page lookup failed (%d..%d): %w", start, start+limit, err)
		}
		if len(page) != limit {
			return nil, fmt.Errorf("branch history page length mismatch (%d..%d): got %d, want %d", start, start+limit, len(page), limit)
		}
		records = append(records, page...)
	}
	return records, nil
}

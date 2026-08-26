// Package app implements Bit's init, push, and pull workflows.
package app

import (
	"math/big"

	"github.com/opendasom/bit/internal/chain"
)

// ChainClient defines the contract operations used by workflows.
type ChainClient interface {
	GetBranchCommit(repoID *big.Int, branch string) ([20]byte, error)
	GetBranchHistoryLength(repoID *big.Int, branch string) (*big.Int, error)
	GetBranchCommitsWithMetadata(repoID *big.Int, branch string, start, limit *big.Int) ([]chain.BranchCommitRecord, error)
	RecordCommit(
		repoID *big.Int,
		branch string,
		expectedOldCommit [20]byte,
		commitHash [20]byte,
		treeHash [20]byte,
		parentHashes [][20]byte,
		manifestDigest [32]byte,
		diffDigest [32]byte,
	) error
	CreateRepo(metadataCID string) (*big.Int, error)
}

// IPFSClient provides workflow content storage.
type IPFSClient interface {
	Upload(data []byte) (string, error)
	Download(cid string) ([]byte, error)
}

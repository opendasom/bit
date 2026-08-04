package app

import (
	"math/big"

	"github.com/opendasom/bit/internal/chain"
)

// ChainClient is the subset of chain.Client's methods that app-level
// commands depend on. Defining it here instead of depending on the concrete
// *chain.Client type lets tests inject a fake instead of dialing a real chain.
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
	CreatePullRequest(targetRepoID *big.Int, targetBranch string, sourceRepoID *big.Int, sourceBranch string) (*big.Int, error)
}

// IPFSClient is the subset of ipfs.Client's methods that app-level commands
// depend on.
type IPFSClient interface {
	Upload(data []byte) (string, error)
	Download(cid string) ([]byte, error)
}

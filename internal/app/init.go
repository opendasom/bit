package app

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/repo"
)

// Init uploads metadata and registers the repository.
func Init(chainClient ChainClient, ipfsClient IPFSClient, metadata *repo.Metadata) (*big.Int, string, error) {
	metadataData, err := repo.EncodeMetadata(metadata)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build repo metadata: %w", err)
	}
	metadataCID, err := ipfsClient.Upload(metadataData)
	if err != nil {
		return nil, "", fmt.Errorf("repo metadata IPFS upload failed: %w", err)
	}

	repoID, err := chainClient.CreateRepo(metadataCID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create repo: %w", err)
	}
	if repoID == nil {
		return nil, "", fmt.Errorf("failed to create repo: could not determine repoId")
	}

	return repoID, metadataCID, nil
}

package app

import (
	"fmt"
	"math/big"

	"github.com/opendasom/bit/internal/repo"
)

// Init uploads repo metadata to IPFS and registers a new repo on-chain,
// returning the newly assigned repo ID and the metadata's CID.
func Init(chainClient ChainClient, ipfsClient IPFSClient, metadata *repo.Metadata) (*big.Int, string, error) {
	metadataData, err := repo.EncodeMetadata(metadata)
	if err != nil {
		return nil, "", fmt.Errorf("repo metadata 생성 실패: %w", err)
	}
	metadataCID, err := ipfsClient.Upload(metadataData)
	if err != nil {
		return nil, "", fmt.Errorf("repo metadata IPFS 업로드 실패: %w", err)
	}

	repoID, err := chainClient.CreateRepo(metadataCID)
	if err != nil {
		return nil, "", fmt.Errorf("저장소 생성 실패: %w", err)
	}
	if repoID == nil {
		return nil, "", fmt.Errorf("저장소 생성 실패: repoId를 확인할 수 없습니다")
	}

	return repoID, metadataCID, nil
}

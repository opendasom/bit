package app

import (
	"errors"
	"math/big"
	"testing"

	"github.com/opendasom/bit/internal/repo"
)

func TestInitUploadsMetadataAndCreatesRepo(t *testing.T) {
	wantRepoID := big.NewInt(3)
	var createRepoCalledWith string

	chainClient := &fakeChainClient{
		createRepo: func(metadataCID string) (*big.Int, error) {
			createRepoCalledWith = metadataCID
			return wantRepoID, nil
		},
	}
	ipfsClient := newFakeIPFSClient()

	metadata := &repo.Metadata{
		Version:       repo.MetadataVersion,
		Name:          "my-project",
		DefaultBranch: "main",
	}

	repoID, metadataCID, err := Init(chainClient, ipfsClient, metadata)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if repoID.Cmp(wantRepoID) != 0 {
		t.Fatalf("expected repo ID %s, got %s", wantRepoID, repoID)
	}
	if metadataCID == "" {
		t.Fatal("expected non-empty metadata CID")
	}
	if createRepoCalledWith != metadataCID {
		t.Fatalf("expected CreateRepo to receive the uploaded metadata CID %s, got %s", metadataCID, createRepoCalledWith)
	}
	if _, err := ipfsClient.Download(metadataCID); err != nil {
		t.Fatalf("expected metadata to be uploaded to IPFS: %v", err)
	}
}

func TestInitPropagatesIPFSUploadFailure(t *testing.T) {
	wantErr := errors.New("ipfs daemon unreachable")
	ipfsClient := &fakeIPFSClient{store: make(map[string][]byte), uploadErr: wantErr}
	chainClient := &fakeChainClient{
		createRepo: func(metadataCID string) (*big.Int, error) {
			t.Fatal("CreateRepo should not be called when metadata upload fails")
			return nil, nil
		},
	}

	metadata := &repo.Metadata{Version: repo.MetadataVersion, Name: "my-project", DefaultBranch: "main"}
	_, _, err := Init(chainClient, ipfsClient, metadata)
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestInitPropagatesCreateRepoFailure(t *testing.T) {
	wantErr := errors.New("chain unavailable")
	chainClient := &fakeChainClient{
		createRepo: func(metadataCID string) (*big.Int, error) {
			return nil, wantErr
		},
	}

	metadata := &repo.Metadata{Version: repo.MetadataVersion, Name: "my-project", DefaultBranch: "main"}
	_, _, err := Init(chainClient, newFakeIPFSClient(), metadata)
	if err == nil || !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped %v, got %v", wantErr, err)
	}
}

func TestInitRejectsNilRepoIDWithoutError(t *testing.T) {
	chainClient := &fakeChainClient{
		createRepo: func(metadataCID string) (*big.Int, error) {
			return nil, nil // defensive case: chain call "succeeded" but returned nothing usable
		},
	}

	metadata := &repo.Metadata{Version: repo.MetadataVersion, Name: "my-project", DefaultBranch: "main"}
	_, _, err := Init(chainClient, newFakeIPFSClient(), metadata)
	if err == nil {
		t.Fatal("expected an error when CreateRepo returns a nil repo ID")
	}
}

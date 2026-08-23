// Package repo defines the repository-level metadata document uploaded to
// IPFS on `bit init`, whose CID is recorded on-chain alongside the new repo.
package repo

import "encoding/json"

const MetadataVersion = 1

// Metadata is the repository-wide (not per-commit) information shown by
// front ends browsing a repo, such as its display name and default branch.
type Metadata struct {
	Version       int    `json:"version"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	DefaultBranch string `json:"defaultBranch"`
}

func EncodeMetadata(m *Metadata) ([]byte, error) {
	return json.Marshal(m)
}

func DecodeMetadata(data []byte) (*Metadata, error) {
	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

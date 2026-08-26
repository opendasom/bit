// Package repo defines repository metadata.
package repo

import "encoding/json"

const MetadataVersion = 1

// Metadata describes a repository for clients browsing it.
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

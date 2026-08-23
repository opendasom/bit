// Package manifest defines the JSON document that accompanies each commit
// diff uploaded to IPFS. It carries the metadata (author, committer,
// parents, tree hash) needed to deterministically rebuild the original Git
// commit on pull, independent of any single git remote.
package manifest

import "encoding/json"

const (
	VersionCommitDiff = 1
	StorageGitDiff    = "git-diff"
	DiffBinaryPatch   = "git diff --binary --full-index"
)

// Identity captures the metadata needed to rebuild the original Git commit.
type Identity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

// Manifest is uploaded to IPFS and its CID is recorded on-chain as the branch
// head. Version 1 stores exactly one commit diff per manifest.
type Manifest struct {
	Version       int      `json:"version"`
	Storage       string   `json:"storage"`
	DiffAlgorithm string   `json:"diffAlgorithm"`
	Branch        string   `json:"branch"`
	DiffCID       string   `json:"diffCid"`
	GitCommit     string   `json:"gitCommit"`
	TreeHash      string   `json:"treeHash"`
	ParentCommits []string `json:"parentCommits"`
	Author        Identity `json:"author"`
	Committer     Identity `json:"committer"`
	Message       string   `json:"message"`

	// BundleCID is kept for backwards-compatible decoding of older manifests.
	BundleCID string `json:"bundleCid,omitempty"`
}

func Encode(m *Manifest) ([]byte, error) {
	return json.Marshal(m)
}

func Decode(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// Package manifest defines the commit metadata stored alongside diffs.
package manifest

import "encoding/json"

const (
	VersionCommitDiff = 1
	StorageGitDiff    = "git-diff"
	DiffBinaryPatch   = "git diff --binary --full-index"
)

type Identity struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Date  string `json:"date"`
}

// Manifest is uploaded to IPFS. The chain stores its CID digest in the commit
// record; branch heads store Git commit hashes.
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

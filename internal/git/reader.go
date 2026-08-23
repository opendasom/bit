// Package git wraps the local `git` CLI (and go-git for read-only HEAD
// inspection) to read commit history, extract diffs for push, and
// reconstruct commits from downloaded diffs for pull. It never talks to
// IPFS or the chain directly; those concerns live in internal/ipfs,
// internal/chain, and internal/app.
package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/opendasom/bit/internal/manifest"
)

// HeadInfo holds the current branch and commit information read from
// the local .git directory.
type HeadInfo struct {
	Branch     string // current branch name (e.g. "main")
	CommitHash string // current commit hash
	ParentHash string // parent commit hash (empty for the initial commit)
}

// CommitInfo is the parsed metadata of a single git commit object, as
// produced by ReadCommit and consumed when building manifests for push.
type CommitInfo struct {
	Hash           string
	TreeHash       string
	ParentHashes   []string
	AuthorName     string
	AuthorEmail    string
	AuthorDate     string
	CommitterName  string
	CommitterEmail string
	CommitterDate  string
	Message        string
}

// InitRepository runs `git init` at repoPath, used by `bit clone` to create
// the local repository before pulling history into it.
func InitRepository(repoPath string) error {
	_, err := runGit(repoPath, "init")
	return err
}

func runGit(repoPath string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out.Bytes(), nil
}

func runGitInput(repoPath string, input []byte, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	cmd.Stdin = bytes.NewReader(input)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out.Bytes(), nil
}

func runGitEnv(repoPath string, env []string, args ...string) ([]byte, error) {
	return runGitEnvInput(repoPath, env, nil, args...)
}

func runGitEnvInput(repoPath string, env []string, input []byte, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdin = bytes.NewReader(input)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return out.Bytes(), nil
}

// ReadHead reads the current HEAD information from the local git
// repository at repoPath (the repo root, e.g. "."). Used by push to read
// the current commit hash and branch name.
func ReadHead(repoPath string) (*HeadInfo, error) {
	repo, err := gogit.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}

	commitHash := head.Hash().String()
	branch := head.Name().Short()

	// Read the parent commit hash; ParentHashes is empty for the initial commit.
	parentHash := ""
	commit, err := repo.CommitObject(head.Hash())
	if err == nil && len(commit.ParentHashes) > 0 {
		parentHash = commit.ParentHashes[0].String()
	}

	return &HeadInfo{
		Branch:     branch,
		CommitHash: commitHash,
		ParentHash: parentHash,
	}, nil
}

// CurrentBranch returns the name of the currently checked-out branch. It
// fails if HEAD is detached, since bit tracks history per named branch.
func CurrentBranch(repoPath string) (string, error) {
	out, err := runGit(repoPath, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", errors.New("failed to determine current branch")
	}
	return branch, nil
}

// CurrentHead returns the full hash of the current HEAD commit.
func CurrentHead(repoPath string) (string, error) {
	out, err := runGit(repoPath, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// HasHead reports whether repoPath has any commits yet, i.e. whether HEAD
// resolves to a valid commit.
func HasHead(repoPath string) bool {
	_, err := runGit(repoPath, "rev-parse", "--verify", "HEAD")
	return err == nil
}

// CommitsAfter lists commit hashes on the current branch after baseCommit,
// oldest first. If baseCommit is empty, it returns the entire history up to
// HEAD. It errors if baseCommit is not an ancestor of HEAD, since that means
// the local branch has diverged from what was last pushed.
func CommitsAfter(repoPath, baseCommit string) ([]string, error) {
	var args []string
	if baseCommit == "" {
		args = []string{"rev-list", "--reverse", "HEAD"}
	} else {
		if _, err := runGit(repoPath, "merge-base", "--is-ancestor", baseCommit, "HEAD"); err != nil {
			return nil, fmt.Errorf("remote branch head %s is not an ancestor of local HEAD", baseCommit)
		}
		args = []string{"rev-list", "--reverse", baseCommit + "..HEAD"}
	}
	out, err := runGit(repoPath, args...)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil, nil
	}
	return strings.Split(text, "\n"), nil
}

// ReadCommit reads and parses a single commit object with `git cat-file`,
// returning its tree, parents, author/committer identities, and message.
func ReadCommit(repoPath, commitHash string) (*CommitInfo, error) {
	resolved, err := runGit(repoPath, "rev-parse", "--verify", commitHash+"^{commit}")
	if err != nil {
		return nil, err
	}
	raw, err := runGit(repoPath, "cat-file", "commit", strings.TrimSpace(string(resolved)))
	if err != nil {
		return nil, err
	}

	header, message, ok := strings.Cut(string(raw), "\n\n")
	if !ok {
		return nil, fmt.Errorf("invalid commit object: %s", commitHash)
	}

	info := &CommitInfo{Hash: strings.TrimSpace(string(resolved)), Message: message}
	for _, line := range strings.Split(header, "\n") {
		switch {
		case strings.HasPrefix(line, "tree "):
			info.TreeHash = strings.TrimPrefix(line, "tree ")
		case strings.HasPrefix(line, "parent "):
			info.ParentHashes = append(info.ParentHashes, strings.TrimPrefix(line, "parent "))
		case strings.HasPrefix(line, "author "):
			name, email, date, err := parseCommitIdentity(strings.TrimPrefix(line, "author "))
			if err != nil {
				return nil, err
			}
			info.AuthorName = name
			info.AuthorEmail = email
			info.AuthorDate = date
		case strings.HasPrefix(line, "committer "):
			name, email, date, err := parseCommitIdentity(strings.TrimPrefix(line, "committer "))
			if err != nil {
				return nil, err
			}
			info.CommitterName = name
			info.CommitterEmail = email
			info.CommitterDate = date
		}
	}

	if info.TreeHash == "" || info.AuthorEmail == "" || info.CommitterEmail == "" {
		return nil, fmt.Errorf("incomplete commit metadata: %s", commitHash)
	}
	return info, nil
}

// parseCommitIdentity parses a raw "author"/"committer" header line body of
// the form "Name <email> <unix-timestamp> <timezone>" into its parts. The
// returned date is in git's "@<timestamp> <timezone>" format, ready to pass
// as a GIT_AUTHOR_DATE/GIT_COMMITTER_DATE value.
func parseCommitIdentity(raw string) (name, email, date string, err error) {
	emailEnd := strings.LastIndex(raw, ">")
	emailStart := strings.LastIndex(raw[:emailEnd+1], " <")
	if emailEnd < 0 || emailStart < 0 || emailEnd+2 >= len(raw) {
		return "", "", "", fmt.Errorf("invalid commit identity: %q", raw)
	}
	name = raw[:emailStart]
	email = raw[emailStart+2 : emailEnd]
	dateParts := strings.Fields(raw[emailEnd+1:])
	if len(dateParts) < 2 {
		return "", "", "", fmt.Errorf("invalid commit date: %q", raw)
	}
	date = "@" + dateParts[0] + " " + dateParts[1]
	return name, email, date, nil
}

// ExtractCommitDiff returns the binary patch for a single commit: a full
// diff against an empty tree for a root commit, or a diff against its first
// parent otherwise. This is the payload uploaded to IPFS on push.
func ExtractCommitDiff(repoPath string, info *CommitInfo) ([]byte, error) {
	if len(info.ParentHashes) == 0 {
		return runGit(repoPath, "diff-tree", "--root", "--binary", "--full-index", "--patch", "--no-commit-id", info.Hash)
	}
	return runGit(repoPath, "diff", "--binary", "--full-index", info.ParentHashes[0], info.Hash)
}

// ManifestForCommit builds the manifest that gets uploaded to IPFS alongside
// a commit's diff, capturing everything needed to reconstruct the commit
// (tree hash, parents, author/committer identity, and message) on pull.
func ManifestForCommit(branch string, info *CommitInfo, diffCID string) *manifest.Manifest {
	return &manifest.Manifest{
		Version:       manifest.VersionCommitDiff,
		Storage:       manifest.StorageGitDiff,
		DiffAlgorithm: manifest.DiffBinaryPatch,
		Branch:        branch,
		DiffCID:       diffCID,
		GitCommit:     info.Hash,
		TreeHash:      info.TreeHash,
		ParentCommits: append([]string(nil), info.ParentHashes...),
		Author: manifest.Identity{
			Name:  info.AuthorName,
			Email: info.AuthorEmail,
			Date:  info.AuthorDate,
		},
		Committer: manifest.Identity{
			Name:  info.CommitterName,
			Email: info.CommitterEmail,
			Date:  info.CommitterDate,
		},
		Message: info.Message,
	}
}

// ApplyCommitDiff builds a commit from a manifest and diff via
// BuildCommitDiff, then checks out the branch at the resulting commit. The
// worktree must be clean beforehand.
func ApplyCommitDiff(repoPath string, m *manifest.Manifest, diff []byte) error {
	if err := ensureCleanWorktree(repoPath); err != nil {
		return err
	}
	newHash, err := BuildCommitDiff(repoPath, m, diff)
	if err != nil {
		return err
	}
	return CheckoutBranch(repoPath, m.Branch, newHash)
}

// BuildCommitDiff verifies and constructs a commit with an isolated temporary
// index. It writes Git objects but does not change refs, the index, or the worktree.
func BuildCommitDiff(repoPath string, m *manifest.Manifest, diff []byte) (string, error) {
	if m.Version != manifest.VersionCommitDiff || m.Storage != manifest.StorageGitDiff || m.DiffAlgorithm != manifest.DiffBinaryPatch {
		return "", fmt.Errorf("unsupported manifest storage: version=%d storage=%s", m.Version, m.Storage)
	}
	if m.Branch == "" {
		return "", errors.New("manifest branch is empty")
	}
	if len(m.ParentCommits) > 1 {
		return "", errors.New("merge commit diff is not supported")
	}
	if _, err := runGit(repoPath, "check-ref-format", "--branch", m.Branch); err != nil {
		return "", fmt.Errorf("invalid manifest branch %q: %w", m.Branch, err)
	}

	indexFile, err := os.CreateTemp("", "bit-index-*")
	if err != nil {
		return "", err
	}
	indexPath := indexFile.Name()
	if err := indexFile.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(indexPath); err != nil {
		return "", err
	}
	defer os.Remove(indexPath)
	indexEnv := []string{"GIT_INDEX_FILE=" + indexPath}

	if len(m.ParentCommits) == 0 {
		if _, err := runGitEnv(repoPath, indexEnv, "read-tree", "--empty"); err != nil {
			return "", err
		}
	} else {
		for _, parent := range m.ParentCommits {
			if _, err := runGit(repoPath, "cat-file", "-e", parent+"^{commit}"); err != nil {
				return "", fmt.Errorf("missing parent commit %s: %w", parent, err)
			}
		}
		if _, err := runGitEnv(repoPath, indexEnv, "read-tree", m.ParentCommits[0]+"^{tree}"); err != nil {
			return "", err
		}
	}

	if len(bytes.TrimSpace(diff)) > 0 {
		if _, err := runGitEnvInput(repoPath, indexEnv, diff, "apply", "--cached", "--binary", "-"); err != nil {
			return "", err
		}
	}

	treeOut, err := runGitEnv(repoPath, indexEnv, "write-tree")
	if err != nil {
		return "", err
	}
	treeHash := strings.TrimSpace(string(treeOut))
	if treeHash != m.TreeHash {
		return "", fmt.Errorf("rebuilt tree mismatch: got %s, want %s", treeHash, m.TreeHash)
	}

	args := []string{"commit-tree", treeHash}
	for _, parent := range m.ParentCommits {
		args = append(args, "-p", parent)
	}
	env := append(indexEnv, []string{
		"GIT_AUTHOR_NAME=" + m.Author.Name,
		"GIT_AUTHOR_EMAIL=" + m.Author.Email,
		"GIT_AUTHOR_DATE=" + m.Author.Date,
		"GIT_COMMITTER_NAME=" + m.Committer.Name,
		"GIT_COMMITTER_EMAIL=" + m.Committer.Email,
		"GIT_COMMITTER_DATE=" + m.Committer.Date,
	}...)
	commitOut, err := runGitEnvInput(repoPath, env, []byte(m.Message), args...)
	if err != nil {
		return "", err
	}
	newHash := strings.TrimSpace(string(commitOut))
	if newHash != m.GitCommit {
		return "", fmt.Errorf("rebuilt commit mismatch: got %s, want %s", newHash, m.GitCommit)
	}
	return newHash, nil
}

// CheckoutBranch force-creates (or resets) branch to point at commitHash and
// checks it out, requiring a clean worktree first.
func CheckoutBranch(repoPath, branch, commitHash string) error {
	if branch == "" {
		return errors.New("branch is empty")
	}
	if err := ensureCleanWorktree(repoPath); err != nil {
		return err
	}
	_, err := runGit(repoPath, "checkout", "-B", branch, commitHash)
	return err
}

// EnsureCleanWorktree returns an error unless the worktree has no
// uncommitted changes (untracked .bit/ state is ignored). Callers use this
// to avoid clobbering local work before checking out a different commit.
func EnsureCleanWorktree(repoPath string) error {
	return ensureCleanWorktree(repoPath)
}

func ensureCleanWorktree(repoPath string) error {
	out, err := runGit(repoPath, "status", "--porcelain")
	if err != nil {
		return err
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "?? .bit/" || line == "?? .bit" || strings.HasPrefix(line, "?? .bit/") {
			continue
		}
		return errors.New("working tree has uncommitted changes")
	}
	return nil
}

// ExtractBundle extracts every Git object in the current repository as a
// bundle. Used by push to upload the full repository history to IPFS.
// Internally runs "git bundle create - --all".
func ExtractBundle(repoPath string) ([]byte, error) {
	cmd := exec.Command("git", "bundle", "create", "-", "--all")
	cmd.Dir = repoPath
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// ApplyBundle applies bundle data downloaded from IPFS to the local git
// repository. Used by pull to apply downloaded history locally.
// Internally runs "git bundle unbundle -" followed by
// "git checkout -b <branch> <commit>".
func ApplyBundle(repoPath string, data []byte) error {
	// Unpack the bundle into the local .git directory.
	unbundle := exec.Command("git", "bundle", "unbundle", "-")
	unbundle.Dir = repoPath
	unbundle.Stdin = bytes.NewReader(data)
	var out bytes.Buffer
	unbundle.Stdout = &out
	if err := unbundle.Run(); err != nil {
		return err
	}

	// Extract the commit hash and branch name from the unbundle output.
	// Output format: "<commitHash> refs/heads/<branch>"
	var commitHash, refName string
	if _, err := fmt.Sscanf(out.String(), "%s %s", &commitHash, &refName); err != nil {
		return fmt.Errorf("invalid git bundle output: %w", err)
	}
	if !strings.HasPrefix(refName, "refs/heads/") {
		return fmt.Errorf("unsupported git bundle ref: %s", refName)
	}
	branch := refName[len("refs/heads/"):]

	// Check out the resulting branch.
	checkout := exec.Command("git", "checkout", "-b", branch, commitHash)
	checkout.Dir = repoPath
	return checkout.Run()
}

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

// HeadInfo는 로컬 .git에서 읽은 현재 브랜치와 커밋 정보를 담는다.
type HeadInfo struct {
	Branch     string // 현재 브랜치명 (예: "main")
	CommitHash string // 현재 커밋 해시
	ParentHash string // 부모 커밋 해시 (초기 커밋이면 빈 문자열)
}

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

// ReadHead는 로컬 git 저장소에서 현재 HEAD 정보를 읽어 반환한다.
// repoPath: git 저장소 루트 경로 (예: ".")
// push 시 커밋 해시와 브랜치명을 읽을 때 사용한다.
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

	// 부모 커밋 해시 읽기 (초기 커밋이면 ParentHashes가 비어있음)
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

func CurrentBranch(repoPath string) (string, error) {
	out, err := runGit(repoPath, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return "", errors.New("현재 브랜치를 확인할 수 없습니다")
	}
	return branch, nil
}

func CurrentHead(repoPath string) (string, error) {
	out, err := runGit(repoPath, "rev-parse", "--verify", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func HasHead(repoPath string) bool {
	_, err := runGit(repoPath, "rev-parse", "--verify", "HEAD")
	return err == nil
}

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

func ExtractCommitDiff(repoPath string, info *CommitInfo) ([]byte, error) {
	if len(info.ParentHashes) == 0 {
		return runGit(repoPath, "diff-tree", "--root", "--binary", "--full-index", "--patch", "--no-commit-id", info.Hash)
	}
	return runGit(repoPath, "diff", "--binary", "--full-index", info.ParentHashes[0], info.Hash)
}

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

func ApplyCommitDiff(repoPath string, m *manifest.Manifest, diff []byte) error {
	if err := ensureCleanWorktree(repoPath); err != nil {
		return err
	}
	if m.Version != manifest.VersionCommitDiff || m.Storage != manifest.StorageGitDiff {
		return fmt.Errorf("unsupported manifest storage: version=%d storage=%s", m.Version, m.Storage)
	}
	if m.Branch == "" {
		return errors.New("manifest branch is empty")
	}

	if len(m.ParentCommits) == 0 {
		if _, err := runGit(repoPath, "symbolic-ref", "HEAD", "refs/heads/"+m.Branch); err != nil {
			return err
		}
	} else {
		for _, parent := range m.ParentCommits {
			if _, err := runGit(repoPath, "cat-file", "-e", parent+"^{commit}"); err != nil {
				return fmt.Errorf("missing parent commit %s: %w", parent, err)
			}
		}
		if _, err := runGit(repoPath, "checkout", "-B", m.Branch, m.ParentCommits[0]); err != nil {
			return err
		}
	}

	if len(bytes.TrimSpace(diff)) > 0 {
		if _, err := runGitInput(repoPath, diff, "apply", "--index", "--binary", "-"); err != nil {
			return err
		}
	}

	treeOut, err := runGit(repoPath, "write-tree")
	if err != nil {
		return err
	}
	treeHash := strings.TrimSpace(string(treeOut))
	if treeHash != m.TreeHash {
		return fmt.Errorf("rebuilt tree mismatch: got %s, want %s", treeHash, m.TreeHash)
	}

	args := []string{"commit-tree", treeHash}
	for _, parent := range m.ParentCommits {
		args = append(args, "-p", parent)
	}
	env := []string{
		"GIT_AUTHOR_NAME=" + m.Author.Name,
		"GIT_AUTHOR_EMAIL=" + m.Author.Email,
		"GIT_AUTHOR_DATE=" + m.Author.Date,
		"GIT_COMMITTER_NAME=" + m.Committer.Name,
		"GIT_COMMITTER_EMAIL=" + m.Committer.Email,
		"GIT_COMMITTER_DATE=" + m.Committer.Date,
	}
	commitOut, err := runGitEnvInput(repoPath, env, []byte(m.Message), args...)
	if err != nil {
		return err
	}
	newHash := strings.TrimSpace(string(commitOut))
	if newHash != m.GitCommit {
		return fmt.Errorf("rebuilt commit mismatch: got %s, want %s", newHash, m.GitCommit)
	}

	if _, err := runGit(repoPath, "reset", "--hard", newHash); err != nil {
		return err
	}
	return nil
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

// ExtractBundle은 현재 저장소의 모든 Git 객체를 bundle 형태로 추출해 반환한다.
// push 시 코드 전체를 IPFS에 업로드하기 위해 사용한다.
// 내부적으로 "git bundle create - --all"을 실행한다.
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

// ApplyBundle은 IPFS에서 받은 bundle 데이터를 로컬 git 저장소에 반영한다.
// pull 시 다운로드한 코드를 로컬에 적용할 때 사용한다.
// 내부적으로 "git bundle unbundle -" 후 "git checkout -b <branch> <commit>"을 실행한다.
func ApplyBundle(repoPath string, data []byte) error {
	// bundle을 로컬 .git에 풀기
	unbundle := exec.Command("git", "bundle", "unbundle", "-")
	unbundle.Dir = repoPath
	unbundle.Stdin = bytes.NewReader(data)
	var out bytes.Buffer
	unbundle.Stdout = &out
	if err := unbundle.Run(); err != nil {
		return err
	}

	// unbundle 출력에서 커밋 해시와 브랜치명 추출
	// 출력 형식: "<commitHash> refs/heads/<branch>"
	var commitHash, refName string
	fmt.Sscanf(out.String(), "%s %s", &commitHash, &refName)
	branch := refName[len("refs/heads/"):]

	// 해당 브랜치로 checkout
	checkout := exec.Command("git", "checkout", "-b", branch, commitHash)
	checkout.Dir = repoPath
	return checkout.Run()
}

#!/usr/bin/env bash
# Test cases for `bit pull`.

# PULL-1: pulling into a brand-new (HEAD-less) repo reconstructs the exact
# same commit hash git originally computed, not just "similar" content.
#
# `git.ApplyCommitDiff` (internal/git/reader.go) rebuilds each commit via
# `git apply` -> `git write-tree` (checked against the manifest's TreeHash)
# -> `git commit-tree` with the original author/committer identity and
# dates replayed through GIT_AUTHOR_DATE / GIT_COMMITTER_DATE -> `git
# reset --hard`. If any of that byte-reproduction is off, the resulting
# commit hash will differ from what's on-chain and pull will simply never
# succeed for that commit - so a passing pull is itself the strongest
# evidence this machinery works.
case_pull_fresh_clone_reconstructs_hash() {
  log_case "PULL-1: fresh clone reconstructs the exact original commit hash"
  local src dst repo_id
  src=$(new_dir pull1_src)
  git_init_repo "$src"
  commit_file "$src" f.txt hello root
  commit_file "$src" g.txt world second
  bit_init "$src" "$KEY_OWNER"
  repo_id=$(repo_id_of "$src")
  bit_remote_add "$src" origin "$repo_id"
  bit_push "$src" >/dev/null

  dst=$(new_dir pull1_dst)
  git_init_repo "$dst"
  bit_init "$dst" "$KEY_OWNER" >/dev/null
  bit_remote_add "$dst" origin "$repo_id"
  bit_pull "$dst" main >/dev/null
  assert_rc "pull succeeds" 0 "$BIT_LAST_RC"

  assert_eq "reconstructed HEAD hash matches source HEAD hash" \
    "$(git_head "$src")" "$(git_head "$dst")"
  assert_eq "file content matches" "$(cat "$src/g.txt")" "$(cat "$dst/g.txt")"

  PULL_SRC="$src"; PULL_DST="$dst"; PULL_REPO_ID="$repo_id"
}

# PULL-2: pulling again after the remote gained more commits applies only
# the new ones and picks up exactly where the last pull left off.
case_pull_incremental() {
  log_case "PULL-2: a second pull only applies newly-pushed commits"
  local src dst repo_id
  src=$(new_dir pull2_src)
  git_init_repo "$src"; commit_file "$src" f.txt hello root
  bit_init "$src" "$KEY_OWNER"
  repo_id=$(repo_id_of "$src")
  bit_remote_add "$src" origin "$repo_id"
  bit_push "$src" >/dev/null

  dst=$(new_dir pull2_dst)
  git_init_repo "$dst"; bit_init "$dst" "$KEY_OWNER" >/dev/null
  bit_remote_add "$dst" origin "$repo_id"
  bit_pull "$dst" main >/dev/null
  assert_rc "first pull succeeds" 0 "$BIT_LAST_RC"

  commit_file "$src" g.txt more second
  bit_push "$src" >/dev/null

  bit_pull "$dst" main >/dev/null
  assert_rc "second pull succeeds" 0 "$BIT_LAST_RC"
  assert_contains "second pull reports exactly 1 applied commit" "$BIT_LAST_OUT" "적용 커밋: 1개"
  assert_eq "dst HEAD now matches src HEAD" "$(git_head "$src")" "$(git_head "$dst")"
}

# PULL-3: a local commit the remote never saw makes the repo "diverged",
# and pull must refuse rather than silently rewriting local history.
#
# This is the single most safety-critical check in `bit pull`: if it were
# missing, pull could discard a user's uncommitted-to-chain work with no
# warning. Verified live: pull.go walks the whole remote history looking
# for local HEAD and, if not found, exits 1 without touching the worktree.
case_pull_diverged_head_rejected() {
  log_case "PULL-3: diverged local HEAD blocks pull instead of overwriting it"
  local src dst repo_id
  src=$(new_dir pull3_src)
  git_init_repo "$src"; commit_file "$src" f.txt hello root
  bit_init "$src" "$KEY_OWNER"
  repo_id=$(repo_id_of "$src")
  bit_remote_add "$src" origin "$repo_id"
  bit_push "$src" >/dev/null

  dst=$(new_dir pull3_dst)
  git_init_repo "$dst"; bit_init "$dst" "$KEY_OWNER" >/dev/null
  bit_remote_add "$dst" origin "$repo_id"
  bit_pull "$dst" main >/dev/null

  local head_before
  head_before=$(git_head "$dst")
  commit_file "$dst" local_only.txt x "local commit never pushed"
  local head_diverged
  head_diverged=$(git_head "$dst")

  bit_pull "$dst" main >/dev/null
  assert_rc "pull exits non-zero" 1 "$BIT_LAST_RC"
  assert_contains "error explains local HEAD isn't in remote history" "$BIT_LAST_OUT" "원격 브랜치 'main' 히스토리에 없습니다"
  assert_eq "local HEAD is untouched by the rejected pull" "$head_diverged" "$(git_head "$dst")"
}

# PULL-4: a dirty worktree blocks pull instead of clobbering uncommitted
# local edits.
case_pull_dirty_worktree_rejected() {
  log_case "PULL-4: uncommitted local changes block pull"
  local src dst repo_id
  src=$(new_dir pull4_src)
  git_init_repo "$src"; commit_file "$src" f.txt hello root
  bit_init "$src" "$KEY_OWNER"
  repo_id=$(repo_id_of "$src")
  bit_remote_add "$src" origin "$repo_id"
  bit_push "$src" >/dev/null

  dst=$(new_dir pull4_dst)
  git_init_repo "$dst"; bit_init "$dst" "$KEY_OWNER" >/dev/null
  bit_remote_add "$dst" origin "$repo_id"
  bit_pull "$dst" main >/dev/null

  commit_file "$src" g.txt more second
  bit_push "$src" >/dev/null

  echo "unstaged edit" >>"$dst/f.txt"
  bit_pull "$dst" main >/dev/null
  assert_rc "pull exits non-zero" 1 "$BIT_LAST_RC"
  assert_contains "error mentions the dirty worktree" "$BIT_LAST_OUT" "uncommitted changes"
  (cd "$dst" && git checkout -- f.txt)
}

# PULL-5: a tampered / inconsistent on-chain record is caught by the
# 3-way manifest verification (commit hash, diff CID, tree hash), not
# blindly applied.
#
# This bypasses `bit push` entirely and calls `recordCommit` directly via
# `cast send`, reusing a REAL, already-uploaded manifestDigest/diffDigest
# but attaching them to a fabricated commitHash/treeHash. This simulates
# either outright tampering, or (more realistically) a bug elsewhere that
# records mismatched metadata for a commit. `internal/app/pull.go` checks
# `m.GitCommit == expectedCommit`, `m.DiffCID == diffCID` and
# `m.TreeHash == expectedTreeHash` before ever calling `git apply` - this
# proves that verification actually stops a bad record instead of just
# looking like it would.
case_pull_tampered_chain_record_rejected() {
  log_case "PULL-5: pull rejects an on-chain record whose manifest doesn't match its claimed commit hash"
  local dir repo_id
  dir=$(new_dir pull5)
  git_init_repo "$dir"; commit_file "$dir" f.txt base root
  bit_init "$dir" "$KEY_OWNER"
  repo_id=$(repo_id_of "$dir")
  bit_remote_add "$dir" origin "$repo_id"
  bit_push "$dir" >/dev/null

  local head branch_h manifest_digest diff_digest
  head=$(chain_branch_commit "$repo_id")
  branch_h=$(branch_hash main)
  read -r _ manifest_digest diff_digest _ _ < <(
    cast call "$CONTRACT" "getCommit(uint256,bytes20)(bytes20,bytes32,bytes32,address,uint256)" \
      "$repo_id" "$head" --rpc-url "$RPC_URL" | tr '\n' ' '
  )

  # Any syntactically valid 20-byte hash works here - it just needs to be
  # different from the real commit hash whose manifest we're reusing.
  local fake_commit fake_tree
  fake_commit="0x$(echo fake-commit | git hash-object --stdin)"
  fake_tree="0x$(echo fake-tree | git hash-object --stdin)"

  cast send "$CONTRACT" \
    "recordCommit(uint256,bytes32,bytes20,bytes20,bytes20,bytes20[],bytes32,bytes32)" \
    "$repo_id" "$branch_h" "$head" "$fake_commit" "$fake_tree" "[$head]" \
    "$manifest_digest" "$diff_digest" \
    --rpc-url "$RPC_URL" --private-key "$KEY_OWNER" >/dev/null

  local puller
  puller=$(new_dir pull5_puller)
  git_init_repo "$puller"; bit_init "$puller" "$KEY_OWNER" >/dev/null
  bit_remote_add "$puller" origin "$repo_id"

  bit_pull "$puller" main >/dev/null
  assert_rc "pull exits non-zero on the tampered record" 1 "$BIT_LAST_RC"
  assert_contains "error identifies a manifest commit mismatch" "$BIT_LAST_OUT" "manifest commit mismatch"
  assert_eq "the real root commit was still applied before the bad one was hit" \
    "$head" "0x$(git_head "$puller")"
}

# PULL-6: pageSize=100 pagination boundary in loadBranchRecords
# (internal/app/common.go). Slow (100+ pushed commits) - opt in with
# RUN_SLOW=1.
case_pull_pagination_boundary() {
  log_case "PULL-6: pull correctly pages through >100 commits (RUN_SLOW=1 only)"
  if [ "$RUN_SLOW" != "1" ]; then
    log_skip "set RUN_SLOW=1 to run this (pushes 101 commits, slow)"
    return
  fi
  local src dst repo_id
  src=$(new_dir pull6_src)
  git_init_repo "$src"; commit_file "$src" f.txt base root
  bit_init "$src" "$KEY_OWNER"
  repo_id=$(repo_id_of "$src")
  bit_remote_add "$src" origin "$repo_id"
  local i
  for i in $(seq 1 100); do
    commit_file "$src" "n$i.txt" "$i" "commit $i"
  done
  bit_push "$src" >/dev/null
  assert_rc "pushing 101 commits succeeds" 0 "$BIT_LAST_RC"
  assert_eq "chain history length is 101" "101" "$(chain_history_length "$repo_id")"

  dst=$(new_dir pull6_dst)
  git_init_repo "$dst"; bit_init "$dst" "$KEY_OWNER" >/dev/null
  bit_remote_add "$dst" origin "$repo_id"
  bit_pull "$dst" main >/dev/null
  assert_rc "pull across the pagination boundary succeeds" 0 "$BIT_LAST_RC"
  assert_eq "final HEAD matches after paginated pull" "$(git_head "$src")" "$(git_head "$dst")"
}

# PULL-7: reproducing commit-tree hashes depends on replaying
# GIT_AUTHOR_DATE / GIT_COMMITTER_DATE exactly (see internal/git/reader.go
# ApplyCommitDiff). This is a real risk if push and pull happen on machines
# with different $TZ, but isn't practically automatable in a single-host
# script (both anvil and this suite run in one timezone) - left as a
# manual check: run `bit push` on one machine, `bit pull` on another with a
# different $TZ, and confirm the resulting commit hash still matches.
case_pull_timezone_reproducibility_manual_note() {
  log_case "PULL-7: cross-timezone commit hash reproducibility (manual only)"
  log_skip "requires two machines/containers with different \$TZ - not automated here"
}

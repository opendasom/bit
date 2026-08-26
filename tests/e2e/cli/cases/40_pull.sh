#!/usr/bin/env bash
# End-to-end cases for `bit pull`.
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

# Reject a divergent HEAD rather than overwrite local history.
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

# Verify metadata before applying a record injected outside the CLI.
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

case_pull_timezone_reproducibility_manual_note() {
  log_case "PULL-7: cross-timezone commit hash reproducibility (manual only)"
  log_skip "requires two machines/containers with different \$TZ - not automated here"
}

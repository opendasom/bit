#!/usr/bin/env bash
# End-to-end cases for `bit remote add`.

case_remote_add_ok() {
  log_case "REMOTE-1: well-formed bit:// URL is parsed and saved"
  local dir
  dir=$(new_dir remote1); git_init_repo "$dir"; commit_file "$dir" f.txt x root
  bit_init "$dir" "$KEY_OWNER" >/dev/null
  bit_remote_add "$dir" origin 42
  assert_rc "remote add succeeds" 0 "$BIT_LAST_RC"
  local stored
  stored=$(jq -r '.remotes.origin.repoId' "$dir/.bit/config.json")
  assert_eq "repoId 42 stored in config.json" "42" "$stored"
}

case_remote_add_branch_fragment() {
  log_case "REMOTE-2: #branch fragment is stripped from repoId"
  local dir
  dir=$(new_dir remote2); git_init_repo "$dir"; commit_file "$dir" f.txt x root
  bit_init "$dir" "$KEY_OWNER" >/dev/null
  (cd "$dir" && "$BIT_BIN" remote add withfrag "bit://local/0xABCDEF0123456789abcdef0123456789ABCDEF01/7#main")
  local stored
  stored=$(jq -r '.remotes.withfrag.repoId' "$dir/.bit/config.json")
  assert_eq "repoId 7 extracted, #main dropped" "7" "$stored"
}

case_remote_add_too_few_segments() {
  log_case "REMOTE-3: malformed URL (too few segments) is rejected cleanly"
  local dir
  dir=$(new_dir remote3); git_init_repo "$dir"; commit_file "$dir" f.txt x root
  bit_init "$dir" "$KEY_OWNER" >/dev/null
  local out rc
  out=$(cd "$dir" && "$BIT_BIN" remote add bad "bit://local/1" 2>&1)
  rc=$?
  assert_rc "exits non-zero, not a panic" 1 "$rc"
  assert_contains "explains the expected URL shape" "$out" "bit://<network>/<registry>/<repoId>"
}

case_remote_add_non_numeric_repo_id() {
  log_case "REMOTE-4: non-numeric repoId is rejected cleanly"
  local dir
  dir=$(new_dir remote4); git_init_repo "$dir"; commit_file "$dir" f.txt x root
  bit_init "$dir" "$KEY_OWNER" >/dev/null
  local out rc
  out=$(cd "$dir" && "$BIT_BIN" remote add bad "bit://local/0xABC/not-a-number" 2>&1)
  rc=$?
  assert_rc "exits non-zero" 1 "$rc"
  assert_contains "explains repoId must be numeric" "$out" "repoId는 숫자여야 합니다"
}

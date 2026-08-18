#!/usr/bin/env bash
# Test cases for `bit init`.

# INIT-1: `bit init` must refuse to run outside a git repository.
#
# Why this matters: init.go checks for `.git` before doing anything else
# (chain connect, IPFS upload, config write). If this check were missing or
# broken, `bit init` would happily create an on-chain repo and write
# .bit/config.json into a directory that isn't actually a git repo yet,
# and the failure would only surface later - much harder to debug - the
# first time `bit push` tries to run `git` commands there.
case_init_missing_git() {
  log_case "INIT-1: bit init fails cleanly when .git is missing"
  local dir
  dir=$(new_dir init1_no_git)
  bit_init "$dir" "$KEY_OWNER" >/dev/null
  assert_rc "exits non-zero" 1 "$BIT_LAST_RC"
  assert_contains "explains that .git is missing" "$BIT_LAST_OUT" ".git 디렉토리가 없습니다"
  # Must not have created a config, since nothing should have happened.
  if [ -f "$dir/.bit/config.json" ]; then
    log_fail "no .bit/config.json should have been written"
  else
    log_pass "no .bit/config.json was written"
  fi
}

# INIT-2: repoIds are assigned sequentially per contract, not per wallet.
#
# Why this matters: `nextRepoId` in RepoRegistry.sol is a single global
# counter for the whole contract. Two independent `bit init` calls (even
# from the same wallet, in different directories) must get two different,
# increasing repoIds - if this ever collided, two unrelated local projects
# would end up writing commits into the same on-chain branch history.
case_init_sequential_repo_ids() {
  log_case "INIT-2: sequential bit init calls get distinct, increasing repoIds"
  local dirA dirB idA idB
  dirA=$(new_dir init2_a); git_init_repo "$dirA"; commit_file "$dirA" f.txt hello root
  dirB=$(new_dir init2_b); git_init_repo "$dirB"; commit_file "$dirB" f.txt hello root

  bit_init "$dirA" "$KEY_OWNER"
  assert_rc "first init succeeds" 0 "$BIT_LAST_RC"
  idA=$(repo_id_of "$dirA")
  bit_init "$dirB" "$KEY_OWNER"
  assert_rc "second init succeeds" 0 "$BIT_LAST_RC"
  idB=$(repo_id_of "$dirB")

  if [ -n "$idA" ] && [ -n "$idB" ] && [ "$idB" -eq $((idA + 1)) ]; then
    log_pass "repoId incremented by exactly 1 ($idA -> $idB)"
  else
    log_fail "expected repoId to increment by 1, got idA=$idA idB=$idB"
  fi

  # Export for later case files that need a ready-made repo + first commit.
  INIT2_DIR_A="$dirA"
  INIT2_REPO_A="$idA"
}

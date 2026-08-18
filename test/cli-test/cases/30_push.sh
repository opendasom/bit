#!/usr/bin/env bash
# Test cases for `bit push`.
#
# Setup shared by most cases here: a repo with a single root commit already
# pushed to chain. Building this fresh per-case (instead of reusing one repo
# across cases) keeps failures isolated - a broken case can't leave stray
# on-chain state that makes a later, unrelated case fail for the wrong
# reason.
_push_setup_pushed_repo() {
  local dir="$1"
  git_init_repo "$dir"
  commit_file "$dir" f.txt base root
  bit_init "$dir" "$KEY_OWNER"
  local repo_id
  repo_id=$(repo_id_of "$dir")
  bit_remote_add "$dir" origin "$repo_id"
  bit_push "$dir" >/dev/null
  echo "$repo_id"
}

# PUSH-1: concurrency / compare-and-swap (CAS) on the branch head.
#
# Two independent working copies of the SAME on-chain repo both start from
# the same pushed commit, each add one new (different) commit on top, and
# then push at (nearly) the same time. `recordCommit` takes an
# `expectedOldCommit` and reverts with StaleBranchHead if the branch head
# moved since the pusher last read it (CommitRegistry.sol). Exactly one of
# the two pushes must win; the other must fail loudly, not silently lose
# data or silently overwrite the winner's commit.
#
# Verified live: the loser's `bit push` exits 1 with the Go-level message
# "다른 사람이 먼저 push했을 수 있습니다" wrapping a raw chain revert.
case_push_cas_race() {
  log_case "PUSH-1: two peers racing to push diverge - CAS rejects the loser"
  local dirD dirE repo_id
  dirD=$(new_dir push1_d)
  repo_id=$(_push_setup_pushed_repo "$dirD")
  dirE=$(new_dir push1_e)
  clone_as_peer "$dirD" "$dirE"

  commit_file "$dirD" from_d.txt d "commit from D"
  commit_file "$dirE" from_e.txt e "commit from E"

  local out_d out_e rc_d rc_e
  ( bit_push "$dirD"; echo "$BIT_LAST_RC" >"$WORKDIR/push1_rc_d"; echo "$BIT_LAST_OUT" >"$WORKDIR/push1_out_d" ) &
  ( bit_push "$dirE"; echo "$BIT_LAST_RC" >"$WORKDIR/push1_rc_e"; echo "$BIT_LAST_OUT" >"$WORKDIR/push1_out_e" ) &
  wait
  rc_d=$(<"$WORKDIR/push1_rc_d"); rc_e=$(<"$WORKDIR/push1_rc_e")
  out_d=$(<"$WORKDIR/push1_out_d"); out_e=$(<"$WORKDIR/push1_out_e")

  if { [ "$rc_d" -eq 0 ] && [ "$rc_e" -ne 0 ]; } || { [ "$rc_e" -eq 0 ] && [ "$rc_d" -ne 0 ]; }; then
    log_pass "exactly one of the two racing pushes succeeded (rc_d=$rc_d rc_e=$rc_e)"
  else
    log_fail "expected exactly one winner (rc_d=$rc_d rc_e=$rc_e)"
  fi
  local loser_out
  if [ "$rc_d" -ne 0 ]; then loser_out="$out_d"; else loser_out="$out_e"; fi
  assert_contains "loser's error explains someone else pushed first" "$loser_out" "다른 사람이 먼저 push했을 수 있습니다"

  local length
  length=$(chain_history_length "$repo_id")
  assert_eq "chain history has exactly 2 commits (root + one winner), not 3" "2" "$length"
}

# PUSH-2: a merge commit aborts the push, but is NOT rolled back atomically.
#
# `internal/app/push.go` walks commits oldest-first and checks
# `len(ParentHashes) > 1` per commit as it goes, not upfront for the whole
# batch. So if a merge commit is preceded by other new (non-merge) commits
# in the same `bit push` invocation, those earlier commits get permanently
# recorded on-chain, and ONLY the merge commit itself is rejected.
#
# This is a real gotcha for testing: a failed push due to a merge commit
# does not mean "nothing happened" - re-running `bit push` after removing
# the merge is the correct recovery, not assuming a clean slate.
#
# Verified live: history length goes from 1 -> 2 (the single-parent commit
# lands), then the command exits 1 with a client-side Go error - no chain
# transaction is even attempted for the merge commit itself.
case_push_merge_commit_partial_then_abort() {
  log_case "PUSH-2: merge commit blocks the push, but earlier commits in the same batch already landed"
  local dir repo_id
  dir=$(new_dir push2)
  repo_id=$(_push_setup_pushed_repo "$dir")

  (cd "$dir" && git checkout -b feature -q)
  commit_file "$dir" feat.txt feat "feature work"
  (cd "$dir" && git checkout main -q && git merge feature -q -m "merge feature" --no-ff)

  local before after
  before=$(chain_history_length "$repo_id")
  bit_push "$dir" >/dev/null
  assert_rc "push exits non-zero" 1 "$BIT_LAST_RC"
  assert_contains "error names the merge commit as unsupported" "$BIT_LAST_OUT" "merge commit diff push is not supported yet"

  after=$(chain_history_length "$repo_id")
  if [ "$after" -eq $((before + 1)) ]; then
    log_pass "the non-merge commit before the merge commit was still recorded ($before -> $after)"
  else
    log_fail "expected history to grow by exactly 1 despite the abort (before=$before after=$after)"
  fi
}

# PUSH-3: locally rewriting already-pushed history (amend/rebase) is
# rejected client-side, before any chain call.
#
# Naively you might expect this to surface as an on-chain FirstParentMismatch
# revert (CommitRegistry.sol checks `parents[0] == currentCommit`). What
# actually happens is earlier and safer: `git.CommitsAfter` in
# internal/git/reader.go explicitly checks that the remote's recorded head
# is an ancestor of local HEAD before computing anything, and refuses with
# a plain Go error if it isn't. No git plumbing runs, no chain transaction
# is sent, and the chain head is provably untouched.
case_push_amended_history_rejected_client_side() {
  log_case "PUSH-3: amending an already-pushed commit is rejected before touching the chain"
  local dir repo_id before_head after_head
  dir=$(new_dir push3)
  repo_id=$(_push_setup_pushed_repo "$dir")
  before_head=$(chain_branch_commit "$repo_id")

  (cd "$dir" && git commit --amend -q -m "root amended")

  bit_push "$dir" >/dev/null
  assert_rc "push exits non-zero" 1 "$BIT_LAST_RC"
  assert_contains "error explains the remote head is not an ancestor of local HEAD" "$BIT_LAST_OUT" "is not an ancestor of local HEAD"

  after_head=$(chain_branch_commit "$repo_id")
  assert_eq "chain branch head is unchanged" "$before_head" "$after_head"
}

# PUSH-4: a Contributor (not Maintainer/Owner) cannot record commits.
#
# `recordCommit` is `onlyMaintainer` (CommitRegistry.sol). This proves the
# permission boundary actually holds end-to-end through the CLI, not just
# in the Solidity tests: the contributor's diff/manifest still get
# uploaded to IPFS (that part has no permission check) before the chain
# call reverts - worth knowing, since it means a permission failure still
# costs IPFS traffic, and a retry after being promoted to Maintainer will
# re-upload rather than reuse that already-uploaded content.
case_push_contributor_lacks_permission() {
  log_case "PUSH-4: Contributor role cannot push (MaintainerRequired)"
  local dir repo_id addr key
  dir=$(new_dir push4)
  repo_id=$(_push_setup_pushed_repo "$dir")

  read -r addr key < <(new_funded_wallet)
  grant_role "$repo_id" "$addr" 1 # 1 = Contributor
  assert_eq "role granted is Contributor" "1" "$(chain_role "$repo_id" "$addr")"

  local peer
  peer=$(new_dir push4_peer)
  clone_as_peer "$dir" "$peer"
  set_identity "$peer" "$key"
  commit_file "$peer" contrib.txt x "commit from contributor"

  local before after
  before=$(chain_history_length "$repo_id")
  bit_push "$peer" >/dev/null
  assert_rc "push exits non-zero" 1 "$BIT_LAST_RC"
  assert_contains "error reports the chain rejected the commit" "$BIT_LAST_OUT" "체인 커밋 기록 실패"
  after=$(chain_history_length "$repo_id")
  assert_eq "chain history is unchanged" "$before" "$after"
}

# PUSH-5 (from the earlier discussion) is intentionally not automated here:
# recordCommit's CommitMetadataMismatch guard (same commitHash, different
# treeHash) is a defense-in-depth check that is not reachable through
# normal `bit push` usage, because a git commit hash already cryptographically
# commits to its tree hash - you cannot construct two different treeHashes
# for the same commitHash without forging a raw `recordCommit` call that
# bypasses the CLI entirely (see PULL-5 below for that style of test, used
# there because tampering IS reachable from the read/pull side).

# PUSH-6: pushing with nothing new to send is a client-side no-op - it must
# not send a transaction at all.
case_push_already_up_to_date() {
  log_case "PUSH-6: pushing with no new commits sends no transaction"
  local dir repo_id before after
  dir=$(new_dir push6)
  repo_id=$(_push_setup_pushed_repo "$dir")
  before=$(chain_history_length "$repo_id")

  bit_push "$dir" >/dev/null
  assert_rc "push still exits 0" 0 "$BIT_LAST_RC"
  assert_contains "reports already up to date" "$BIT_LAST_OUT" "is already up to date"

  after=$(chain_history_length "$repo_id")
  assert_eq "chain history length is unchanged" "$before" "$after"
}

# PUSH-8: killing `bit push` mid-batch and re-running it must resume, not
# duplicate or corrupt history.
#
# Each commit is its own blocking on-chain transaction
# (`chainClient.RecordCommit` waits for the receipt before moving on), so a
# process killed between commit N and commit N+1 leaves commit N fully
# recorded and nothing else in flight. Re-running `bit push` re-reads the
# live chain head first, so it should pick up exactly where it left off.
case_push_interrupted_then_resumed() {
  log_case "PUSH-8: bit push resumes correctly after being killed mid-batch"
  local dir repo_id
  dir=$(new_dir push8)
  repo_id=$(_push_setup_pushed_repo "$dir")
  commit_file "$dir" a.txt a "commit A"
  commit_file "$dir" b.txt b "commit B"
  commit_file "$dir" c.txt c "commit C"

  local logfile
  logfile="$WORKDIR/push8.log"
  (cd "$dir" && "$BIT_BIN" push origin >"$logfile" 2>&1) &
  local pid=$!

  # Wait for the first commit's transaction to land, then kill the process
  # before it can move on to the next one.
  local waited=0
  while [ ! -s "$logfile" ] || ! grep -q "commit push 완료" "$logfile"; do
    sleep 0.2
    waited=$((waited + 1))
    if [ "$waited" -gt 50 ]; then break; fi # ~10s safety cap
  done
  kill -9 "$pid" 2>/dev/null
  wait "$pid" 2>/dev/null

  local mid_length
  mid_length=$(chain_history_length "$repo_id")
  if [ "$mid_length" -ge 2 ] && [ "$mid_length" -le 3 ]; then
    log_pass "killed push left a valid partial history ($mid_length commits recorded)"
  else
    log_fail "expected 2 or 3 commits recorded after kill, got $mid_length"
  fi

  bit_push "$dir" >/dev/null
  assert_rc "resumed push succeeds" 0 "$BIT_LAST_RC"

  local final_length
  final_length=$(chain_history_length "$repo_id")
  assert_eq "final chain history has all 4 commits (root + A + B + C)" "4" "$final_length"
  assert_chain_head_matches_git "chain head matches local git HEAD after resume" "$repo_id" "$dir"
}

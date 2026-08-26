#!/usr/bin/env bash
# Requires a running Anvil node and IPFS daemon.

set -uo pipefail

SUITE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "$SUITE_DIR/lib.sh"
trap cleanup EXIT

log "workspace: $WORKDIR"
check_deps
check_anvil
check_ipfs
build_bit
deploy_contract

for f in "$SUITE_DIR"/cases/*.sh; do
  # shellcheck source=/dev/null
  source "$f"
done

case_init_missing_git
case_init_sequential_repo_ids

case_remote_add_ok
case_remote_add_branch_fragment
case_remote_add_too_few_segments
case_remote_add_non_numeric_repo_id

case_push_cas_race
case_push_merge_commit_partial_then_abort
case_push_amended_history_rejected_client_side
case_push_contributor_lacks_permission
case_push_already_up_to_date
case_push_interrupted_then_resumed

case_pull_fresh_clone_reconstructs_hash
case_pull_incremental
case_pull_diverged_head_rejected
case_pull_dirty_worktree_rejected
case_pull_tampered_chain_record_rejected
case_pull_pagination_boundary
case_pull_timezone_reproducibility_manual_note

log ""
log "================================================================"
log " RESULTS: $PASS_COUNT passed, $FAIL_COUNT failed, $SKIP_COUNT skipped"
log "================================================================"

[ "$FAIL_COUNT" -eq 0 ]

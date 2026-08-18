#!/usr/bin/env bash
# Shared helpers for the bit CLI test suite (test/cli-test).
#
# Every case file sources this to talk to the locally built `bit` binary,
# a locally running anvil chain, and a locally running IPFS daemon. Nothing
# here is bit-specific business logic - it's all plumbing (build, deploy,
# git repo setup, chain reads, assertions).
#
# All behavior encoded in the case files (test/cli-test/cases/*.sh) was
# manually verified against a live anvil + ipfs daemon before being written
# down, so the comments there describe what actually happens, not guesses.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RPC_URL="${RPC_URL:-http://127.0.0.1:8545}"
IPFS_URL="${IPFS_URL:-http://127.0.0.1:5001}"
WORKDIR="${WORKDIR:-$(mktemp -d /tmp/bit-cli-test.XXXXXX)}"
BIT_BIN="$WORKDIR/bit"
KEEP_WORKDIR="${KEEP_WORKDIR:-0}"
RUN_SLOW="${RUN_SLOW:-0}" # set to 1 to also run the expensive/slow cases

# Deployer account. This is anvil's default account #0 under the standard
# "test test test ... junk" mnemonic - safe to hardcode, it only ever holds
# funds on a local anvil instance. It becomes Owner+Maintainer of every repo
# it calls `bit init` for.
KEY_OWNER=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
ADDR_OWNER=0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

CONTRACT="" # set once by deploy_contract, shared by every case in the run

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

# ---------------------------------------------------------------- logging --
log()      { printf '%s\n' "$*" >&2; }
log_case() { printf '\n=== %s ===\n' "$*" >&2; }
log_pass() { PASS_COUNT=$((PASS_COUNT + 1)); printf '  [PASS] %s\n' "$*" >&2; }
log_fail() { FAIL_COUNT=$((FAIL_COUNT + 1)); printf '  [FAIL] %s\n' "$*" >&2; }
log_skip() { SKIP_COUNT=$((SKIP_COUNT + 1)); printf '  [SKIP] %s\n' "$*" >&2; }
log_info() { printf '  [INFO] %s\n' "$*" >&2; }

# ------------------------------------------------------------- assertions --
# assert_rc <desc> <expected_rc> <actual_rc>
assert_rc() {
  local desc="$1" expected="$2" actual="$3"
  if [ "$actual" -eq "$expected" ]; then
    log_pass "$desc (exit=$actual)"
  else
    log_fail "$desc (expected exit=$expected, got=$actual)"
  fi
}

# assert_contains <desc> <haystack> <needle>
assert_contains() {
  local desc="$1" haystack="$2" needle="$3"
  if grep -qF -- "$needle" <<<"$haystack"; then
    log_pass "$desc"
  else
    log_fail "$desc (expected output to contain: '$needle')"
    log_info "actual output:"
    log_info "$haystack"
  fi
}

# assert_eq <desc> <expected> <actual>
assert_eq() {
  local desc="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    log_pass "$desc"
  else
    log_fail "$desc (expected '$expected', got '$actual')"
  fi
}

# ------------------------------------------------------------- preflight --
check_deps() {
  for bin in go forge cast anvil ipfs curl jq git; do
    command -v "$bin" >/dev/null 2>&1 || { log "missing required tool: $bin"; exit 1; }
  done
}

check_anvil() {
  if ! cast chain-id --rpc-url "$RPC_URL" >/dev/null 2>&1; then
    log "cannot reach anvil at $RPC_URL - start it first with: anvil"
    exit 1
  fi
}

check_ipfs() {
  if ! curl -fsS -m 5 -X POST "$IPFS_URL/api/v0/version" >/dev/null 2>&1; then
    log "cannot reach IPFS API at $IPFS_URL - start it first with: ipfs daemon"
    exit 1
  fi
}

build_bit() {
  log "building bit binary -> $BIT_BIN"
  (cd "$REPO_ROOT" && go build -o "$BIT_BIN" .) || { log "go build failed"; exit 1; }
}

deploy_contract() {
  log "deploying BitRegistry to $RPC_URL"
  local out
  out=$(cd "$REPO_ROOT" && forge create --broadcast --rpc-url "$RPC_URL" \
    --private-key "$KEY_OWNER" contracts/src/BitRegistry.sol:BitRegistry 2>&1)
  CONTRACT=$(grep -oE 'Deployed to: 0x[0-9a-fA-F]{40}' <<<"$out" | awk '{print $NF}')
  if [ -z "$CONTRACT" ]; then
    log "contract deploy failed, forge output:"
    log "$out"
    exit 1
  fi
  log "BitRegistry deployed at $CONTRACT"
}

# ------------------------------------------------------- repo/git helpers --
new_dir() {
  local d="$WORKDIR/$1"
  mkdir -p "$d"
  echo "$d"
}

# git_init_repo <dir> - `git init` with a hardcoded default branch name so
# the suite doesn't depend on the machine's global init.defaultBranch config.
git_init_repo() {
  local dir="$1"
  (cd "$dir" && git init -q -b main \
    && git config user.email test@bit.local \
    && git config user.name bit-cli-test)
}

commit_file() {
  local dir="$1" file="$2" content="$3" msg="$4"
  (cd "$dir" && echo "$content" >>"$file" && git add "$file" && git commit -q -m "$msg")
}

git_head() {
  (cd "$1" && git rev-parse HEAD)
}

# ------------------------------------------------------------- bit CLI --
# bit_init <dir> <key> - captures combined stdout+stderr and rc in
# BIT_LAST_OUT / BIT_LAST_RC, same as the other bit_* helpers below.
#
# Deliberately does NOT return the repoId via stdout: callers that need it
# must call repo_id_of() afterwards. Doing it this way (instead of `echo`ing
# the repoId) matters because a caller wrapping this in `id=$(bit_init ...)`
# to grab that output would run the whole function in a subshell, silently
# discarding the BIT_LAST_RC/BIT_LAST_OUT assignments the moment the
# subshell exits - the caller would still "see" the repoId (stdout survives
# command substitution) but any rc/message check right after would silently
# see a stale value from whatever bit_* call ran last in the real shell.
bit_init() {
  local dir="$1" key="$2"
  BIT_LAST_OUT=$(cd "$dir" && "$BIT_BIN" init --rpc "$RPC_URL" --contract "$CONTRACT" \
    --key "$key" --ipfs "$IPFS_URL" 2>&1)
  BIT_LAST_RC=$?
}

# repo_id_of <dir> - reads back the repoId `bit init` wrote into
# .bit/config.json for that directory.
repo_id_of() {
  jq -r '.repoId' "$1/.bit/config.json"
}

bit_remote_add() {
  local dir="$1" name="$2" repo_id="$3"
  BIT_LAST_OUT=$(cd "$dir" && "$BIT_BIN" remote add "$name" "bit://local/$CONTRACT/$repo_id" 2>&1)
  BIT_LAST_RC=$?
}

bit_push() {
  local dir="$1" remote="${2:-origin}"
  BIT_LAST_OUT=$(cd "$dir" && "$BIT_BIN" push "$remote" 2>&1)
  BIT_LAST_RC=$?
}

bit_pull() {
  local dir="$1" branch="${2:-main}" remote="${3:-origin}"
  BIT_LAST_OUT=$(cd "$dir" && "$BIT_BIN" pull "$remote" "$branch" 2>&1)
  BIT_LAST_RC=$?
}

# clone_as_peer <src_dir> <dst_dir> - simulates "another clone of the same
# repo": copies the git history AND the .bit/config.json (remotes + repoId)
# wholesale, so dst can push/pull against the exact same on-chain repo as
# src without running its own `bit init` (which would create a brand new,
# unrelated repoId).
clone_as_peer() {
  local src="$1" dst="$2"
  git clone -q "$src" "$dst"
  mkdir -p "$dst/.bit"
  cp "$src/.bit/config.json" "$dst/.bit/config.json"
}

# set_identity <dir> <private_key> - swaps which wallet a repo's config
# signs transactions with, without touching anything else (remotes, repoId).
set_identity() {
  local dir="$1" key="$2"
  jq --arg k "$key" '.privateKey = $k' "$dir/.bit/config.json" >"$dir/.bit/config.json.tmp"
  mv "$dir/.bit/config.json.tmp" "$dir/.bit/config.json"
}

# new_funded_wallet -> prints "<address> <private_key>", pre-funded with 1
# ETH from KEY_OWNER. Used instead of hardcoding more anvil default keys
# (hand-typed hex is easy to get subtly wrong - see PUSH-4's history).
new_funded_wallet() {
  local json addr key
  json=$(cast wallet new --json)
  addr=$(jq -r '.[0].address' <<<"$json")
  key=$(jq -r '.[0].private_key' <<<"$json")
  cast send "$addr" --value 1ether --rpc-url "$RPC_URL" --private-key "$KEY_OWNER" >/dev/null
  printf '%s %s\n' "$addr" "$key"
}

# ----------------------------------------------------------- chain reads --
branch_hash() { cast keccak "${1:-main}"; }

# chain_branch_commit <repoId> [branch] -> "0x"-prefixed 20-byte git commit
# hash currently recorded as the branch head on-chain.
chain_branch_commit() {
  local repo_id="$1" branch="${2:-main}"
  cast call "$CONTRACT" "getBranchCommit(uint256,bytes32)(bytes20)" \
    "$repo_id" "$(branch_hash "$branch")" --rpc-url "$RPC_URL"
}

chain_history_length() {
  local repo_id="$1" branch="${2:-main}"
  cast call "$CONTRACT" "getBranchHistoryLength(uint256,bytes32)(uint256)" \
    "$repo_id" "$(branch_hash "$branch")" --rpc-url "$RPC_URL"
}

chain_role() {
  local repo_id="$1" addr="$2"
  cast call "$CONTRACT" "getRole(uint256,address)(uint8)" "$repo_id" "$addr" --rpc-url "$RPC_URL"
}

grant_role() {
  # role: 0=None 1=Contributor 2=Maintainer 3=Owner
  local repo_id="$1" addr="$2" role="$3"
  cast send "$CONTRACT" "setRole(uint256,address,uint8)" "$repo_id" "$addr" "$role" \
    --rpc-url "$RPC_URL" --private-key "$KEY_OWNER" >/dev/null
}

# assert_chain_head_matches_git <desc> <repoId> <dir> - the single most
# important invariant in this whole system: the commit hash recorded
# on-chain must be byte-for-byte the same SHA-1 git already computes
# locally, since bit stores raw git hashes on-chain rather than re-deriving
# them from the diff.
assert_chain_head_matches_git() {
  local desc="$1" repo_id="$2" dir="$3"
  local chain_head local_head
  chain_head=$(chain_branch_commit "$repo_id")
  local_head="0x$(git_head "$dir")"
  assert_eq "$desc" "$(tr '[:upper:]' '[:lower:]' <<<"$local_head")" \
    "$(tr '[:upper:]' '[:lower:]' <<<"$chain_head")"
}

cleanup() {
  if [ "$KEEP_WORKDIR" = "1" ]; then
    log "KEEP_WORKDIR=1, leaving workspace at $WORKDIR for inspection"
  else
    rm -rf "$WORKDIR"
  fi
}

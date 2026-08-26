#!/usr/bin/env bash
# Shared helpers for the Bit CLI end-to-end suite.

set -uo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
RPC_URL="${RPC_URL:-http://127.0.0.1:8545}"
IPFS_URL="${IPFS_URL:-http://127.0.0.1:5001}"
WORKDIR="${WORKDIR:-$(mktemp -d /tmp/bit-cli-test.XXXXXX)}"
BIT_BIN="$WORKDIR/bit"
KEEP_WORKDIR="${KEEP_WORKDIR:-0}"
RUN_SLOW="${RUN_SLOW:-0}"

# Anvil's default funded account; only used by local tests.
KEY_OWNER=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
ADDR_OWNER=0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266

CONTRACT=""

PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

log()      { printf '%s\n' "$*" >&2; }
log_case() { printf '\n=== %s ===\n' "$*" >&2; }
log_pass() { PASS_COUNT=$((PASS_COUNT + 1)); printf '  [PASS] %s\n' "$*" >&2; }
log_fail() { FAIL_COUNT=$((FAIL_COUNT + 1)); printf '  [FAIL] %s\n' "$*" >&2; }
log_skip() { SKIP_COUNT=$((SKIP_COUNT + 1)); printf '  [SKIP] %s\n' "$*" >&2; }
log_info() { printf '  [INFO] %s\n' "$*" >&2; }

assert_rc() {
  local desc="$1" expected="$2" actual="$3"
  if [ "$actual" -eq "$expected" ]; then
    log_pass "$desc (exit=$actual)"
  else
    log_fail "$desc (expected exit=$expected, got=$actual)"
  fi
}

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

assert_eq() {
  local desc="$1" expected="$2" actual="$3"
  if [ "$expected" = "$actual" ]; then
    log_pass "$desc"
  else
    log_fail "$desc (expected '$expected', got '$actual')"
  fi
}

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
  (cd "$REPO_ROOT" && go build -o "$BIT_BIN" ./cmd/bit) || { log "go build failed"; exit 1; }
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

new_dir() {
  local d="$WORKDIR/$1"
  mkdir -p "$d"
  echo "$d"
}

# Pin the default branch to keep tests independent of global Git config.
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

# Store command output in globals to avoid command-substitution subshells.
bit_init() {
  local dir="$1" key="$2"
  BIT_LAST_OUT=$(cd "$dir" && "$BIT_BIN" init --rpc "$RPC_URL" --contract "$CONTRACT" \
    --key "$key" --ipfs "$IPFS_URL" 2>&1)
  BIT_LAST_RC=$?
}

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

# Copy Bit metadata so the peer uses the same on-chain repository.
clone_as_peer() {
  local src="$1" dst="$2"
  git clone -q "$src" "$dst"
  mkdir -p "$dst/.bit"
  cp "$src/.bit/config.json" "$dst/.bit/config.json"
}

set_identity() {
  local dir="$1" key="$2"
  jq --arg k "$key" '.privateKey = $k' "$dir/.bit/config.json" >"$dir/.bit/config.json.tmp"
  mv "$dir/.bit/config.json.tmp" "$dir/.bit/config.json"
}

# Create a funded local-test wallet instead of hardcoding additional keys.
new_funded_wallet() {
  local json addr key
  json=$(cast wallet new --json)
  addr=$(jq -r '.[0].address' <<<"$json")
  key=$(jq -r '.[0].private_key' <<<"$json")
  cast send "$addr" --value 1ether --rpc-url "$RPC_URL" --private-key "$KEY_OWNER" >/dev/null
  printf '%s %s\n' "$addr" "$key"
}

branch_hash() { cast keccak "${1:-main}"; }

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
  local repo_id="$1" addr="$2" role="$3"
  cast send "$CONTRACT" "setRole(uint256,address,uint8)" "$repo_id" "$addr" "$role" \
    --rpc-url "$RPC_URL" --private-key "$KEY_OWNER" >/dev/null
}

# Bit records Git's original commit hash on-chain.
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

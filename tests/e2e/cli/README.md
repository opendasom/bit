<p align="right">
  <strong>English</strong> · <a href="README.ko.md">한국어</a>
</p>

# bit CLI Test Suite

A bash test suite that end-to-end verifies `bit init` / `remote add` /
`push` / `pull` against a real anvil chain + a local IPFS daemon. Every
case was actually run once against a live environment to confirm its
behavior before being written down (verified behavior, not guesses).

## Prerequisites

Two things need to already be running (this script does not start them
for you):

```bash
anvil                # http://127.0.0.1:8545
ipfs daemon           # API http://127.0.0.1:5001
```

Required tools: `go`, `forge`, `cast`, `anvil`, `ipfs`, `jq`, `git`, `curl`
(the script errors out immediately at startup if any are missing)

## Running it

**Just `./tests/e2e/cli/run.sh`.** The script builds the `bit` binary and
deploys a fresh contract every time it runs (a brand-new contract each
run, so state never bleeds across separate runs).

```bash
./tests/e2e/cli/run.sh                 # standard cases only (48 assertions)
RUN_SLOW=1 ./tests/e2e/cli/run.sh      # also run slow cases (e.g. pagination)
KEEP_WORKDIR=1 ./tests/e2e/cli/run.sh  # keep the temp workspace after the run (for inspecting failures)
```

At the end it prints a `RESULTS: N passed, N failed, N skipped` summary.
Exit code is 1 if there's any `FAIL`.

## Test case list

### `bit init` (`cases/10_init.sh`)

| ID | What it checks |
|---|---|
| INIT-1 | `bit init` fails cleanly (no config written) in a directory without `.git` |
| INIT-2 | repoId increments by exactly 1 across two `bit init` calls from the same wallet (no global counter collision) |

### `bit remote add` (`cases/20_remote.sh`)

| ID | What it checks |
|---|---|
| REMOTE-1 | a well-formed URL (`bit://local/<addr>/<id>`) is stored correctly in config.json |
| REMOTE-2 | a `#branch` fragment doesn't leak into the repoId parse and gets stripped correctly |
| REMOTE-3 | a URL with too few path segments is rejected cleanly, not a panic |
| REMOTE-4 | a non-numeric repoId is rejected cleanly |

### `bit push` (`cases/30_push.sh`)

| ID | What it checks |
|---|---|
| PUSH-1 | when two clones push at the same time, CAS lets exactly one succeed and the loser gets a clear error (the race is actually reproduced) |
| PUSH-2 | a merge commit blocks the push — **but the plain commits before it in the same batch are already recorded on-chain by the time it aborts** (not atomic; this is the actual, reproduced behavior) |
| PUSH-3 | amending an already-pushed commit and re-pushing gets rejected client-side, before any chain transaction is sent |
| PUSH-4 | a wallet with only Contributor role is rejected on push (`MaintainerRequired`) |
| PUSH-6 | pushing with no new commits sends no transaction at all and just reports "up to date" |
| PUSH-8 | if the process is killed mid-push (some commits recorded, some not), re-running it resumes and finishes correctly |

### `bit pull` (`cases/40_pull.sh`)

| ID | What it checks |
|---|---|
| PULL-1 | pulling into a brand-new directory reconstructs a commit hash 100% identical to the original |
| PULL-2 | a second pull after a partial one only applies the newly-pushed commits |
| PULL-3 | pull refuses to overwrite a local commit the remote never saw, instead of silently discarding it (the single most important safety check) |
| PULL-4 | pull is rejected when the worktree has uncommitted changes |
| PULL-5 | a tampered commit record is injected directly on-chain via `cast send` (bypassing `bit push` entirely), and pull's 3-way verification (commit hash / diff CID / tree hash) actually catches it |
| PULL-6 *(RUN_SLOW=1)* | after pushing past 100 commits, pull correctly crosses the pageSize=100 pagination boundary |
| PULL-7 | commit hash reproduction depends on timezone (`GIT_AUTHOR_DATE`/`GIT_COMMITTER_DATE`) — not automated since it needs two machines, just prints manual-check instructions |

## Notes

- A fresh contract is deployed on every run, so repeatedly running this
  against a long-lived anvil instance never collides with a previous
  run's repoId.
- `.bit/config.json` never stores the private key. Test keys are supplied
  through environment variables, and the temporary `$WORKDIR` is deleted
  when the script exits unless `KEEP_WORKDIR=1`.

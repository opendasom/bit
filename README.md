<p align="right">
  <strong>English</strong> · <a href="README.ko.md">한국어</a>
</p>

<p align="center">
  <img src="docs/assets/bit-logo-readme.png" alt="Bit logo" width="300" />
</p>

<h1 align="center">Bit</h1>

<p align="center">
  <strong>An open protocol for verifiable source history.</strong><br />
  Git-compatible workflows, content-addressed data on IPFS, and repository state verified on Ethereum.
</p>

<p align="center">
  <a href="https://github.com/opendasom/bit/actions/workflows/ci.yml"><img src="https://github.com/opendasom/bit/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI status" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-7cd39c?style=flat-square" alt="MIT License" /></a>
  <img src="https://img.shields.io/badge/status-alpha-f3ae84?style=flat-square" alt="Alpha status" />
</p>

<p align="center">
  <a href="#quick-start">Quick start</a> ·
  <a href="#demos">Demos</a> ·
  <a href="#cli-reference">CLI reference</a> ·
  <a href="CONTRIBUTING.md">Contributing</a> ·
  <a href="SECURITY.md">Security</a>
</p>

Bit is an experimental distributed version-control protocol built on IPFS and Ethereum. Commit diffs and metadata are stored on IPFS, while branch and commit state are recorded by the `BitRegistry` smart contract.

> [!WARNING]
> **Alpha software.** The protocol and storage formats may change. Bit has not received a security audit for production use or environments involving assets of real value.

Bit stores source history in a content-addressed form without relying on a central Git server, allowing anyone to verify it. On-chain records protect history integrity, but IPFS availability depends on pinning and replication. Preserve important data on your own IPFS node or a trusted pinning service.

## Architecture

| Layer | Responsibility |
|---|---|
| **Git client** | Create familiar Git repositories and commits locally, then synchronize remote state with the `bit` CLI. |
| **IPFS** | Store and replicate diffs, manifests, and repository metadata as content-addressed objects. |
| **Ethereum** | Record repository state, Git commit-based branch heads, manifest/diff CID digests, and repository roles in `BitRegistry`. |
| **Web explorer** | Use [`bit-w3`](https://github.com/opendasom/bit-w3) to inspect IPFS and chain state and sign fork, role, and pull-request operations with MetaMask. |

<p align="center">
  <img src="docs/assets/bit-architecture.png" alt="Bit architecture showing participants, the local Git and CLI environment, IPFS, Ethereum, and the Web3 client" width="100%" />
</p>

### Web explorer

Explore repositories and sign registry operations at the deployed [Bit Web3 explorer](https://bitweb.space/).

<table>
  <tr>
    <td width="50%"><img src="docs/assets/web-explorer-home.png" alt="Bit Web3 explorer landing page with Ethereum connection controls and a MetaMask connect button" /></td>
    <td width="50%"><img src="docs/assets/web-explorer-pull-requests.png" alt="Bit Web3 explorer pull-request view with a pull request and signer authorization details" /></td>
  </tr>
  <tr>
    <td align="center">Landing page</td>
    <td align="center">Pull-request review</td>
  </tr>
</table>

## Demos

<table>
  <tr>
    <td align="center" width="33%">
      <a href="https://youtu.be/f56hsj97zWA"><img src="https://img.youtube.com/vi/f56hsj97zWA/0.jpg" alt="Maintainer creating a repository" /></a><br />
      <strong>Maintainer creates a repository</strong>
    </td>
    <td align="center" width="33%">
      <a href="https://youtu.be/HiM1U8WBm3g"><img src="https://img.youtube.com/vi/HiM1U8WBm3g/0.jpg" alt="Contributor developing" /></a><br />
      <strong>Contributor develops a change</strong>
    </td>
    <td align="center" width="33%">
      <a href="https://youtu.be/qIJjybwHhjo"><img src="https://img.youtube.com/vi/qIJjybwHhjo/0.jpg" alt="Maintainer approving a pull request" /></a><br />
      <strong>Maintainer approves a pull request</strong>
    </td>
  </tr>
</table>

## Installation

Prerequisites:

- Go 1.25 language level; `go.mod` selects the Go 1.26.6 toolchain
- An IPFS daemon such as Kubo (`ipfs daemon`)
- Access to an Ethereum node, such as local Anvil or Sepolia
- Foundry 1.7.1+ (`anvil`, `forge`) for local-chain and contract development (optional)
- Node.js 20.19+ for generating the contract ABI artifact (optional)

```bash
git clone https://github.com/opendasom/bit.git
cd bit
go build -o bit ./cmd/bit

# Optional: install as a global command.
sudo cp ./bit /usr/local/bin/bit
```

## Local test environment

Open three terminals and start each component in order.

**Terminal 1 — start Anvil**

```bash
anvil
# Copy a private key from the output; the first default key is sufficient.
```

**Terminal 2 — start IPFS**

```bash
ipfs daemon
```

**Terminal 3 — deploy the contract from the repository root**

```bash
forge create --broadcast \
  --rpc-url http://127.0.0.1:8545 \
  --private-key 0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 \
  contracts/src/BitRegistry.sol:BitRegistry
# Use the "Deployed to: 0x..." address with the --contract flag.
```

The first deployment address on a fresh default Anvil instance is `0x5FbDB2315678afecb367f032d93F642f64180aa3`. Once local IPFS and the new registry are running, seed demo repositories, branches, commits, and pull requests from the Web3 repository:

```bash
git clone --recurse-submodules https://github.com/opendasom/bit-w3.git
cd bit-w3
npm ci
npm run anvil:seed
```

The seed command uses four default Anvil accounts and must run against an empty registry. The private key above is a publicly known Anvil key for **local testing only**. Never use it on a testnet or mainnet.

## Quick start

### 1. Initialize a repository

```bash
mkdir my-project && cd my-project
git init -b main

export BIT_PRIVATE_KEY=ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80
bit init \
  --rpc http://127.0.0.1:8545 \
  --contract 0xYourContractAddress \
  --name my-project
# Default when --ipfs is omitted: http://localhost:5001
```

Provide the private key through an environment variable before write operations. Do not store it in configuration or shell history.

On success, Bit creates `.bit/config.json` and registers the repository on-chain, assigning it a `repoId`.

### 2. Add a remote

```bash
# URL format: bit://<network>/<contractAddress>/<repoId>
bit remote add origin bit://local/0xYourContractAddress/1
```

The CLI reads `repoId` from the remote URL and uses the RPC URL and contract address stored by `bit init` in `.bit/config.json`. Ensure the URL network and contract identify the same deployment as the local configuration.

### 3. Push

```bash
git add . && git commit -m "first commit"
bit push origin
# The current branch is detected automatically.
```

> Pushing merge commits is not supported; history must be linear.

### 4. Pull and clone

Use the read-only `clone` command to restore code on another machine or in another directory. It does not require a private key or create a new on-chain repository.

```bash
bit clone bit://local/0xYourContractAddress/1 other-project \
  --rpc http://127.0.0.1:8545 \
  --ipfs http://127.0.0.1:5001 \
  --branch main
```

### 5. Create and manage forks and pull requests

The Web3 explorer can fork the current branch into a new on-chain repository and create, approve, reject, or close pull requests.

- **Fork `<branch>`** copies the branch's on-chain commit history and IPFS CID pointers in one atomic transaction.
- Fork and pull-request operations support up to 64 commits per operation to limit block gas usage.
- MetaMask signs all fork and pull-request state changes.

1. Select a source repository and branch in the explorer, then choose **Fork**.
2. Clone the generated fork with `bit clone <fork-url> my-fork`.
3. Commit locally and push with `BIT_PRIVATE_KEY=... bit push origin`.
4. Open the fork's **Pull Requests** tab and enter the target repository, target branch, source branch, and description.
5. A target-repository Maintainer can **Approve** or **Reject**. The author or a Maintainer can **Close** the request.

Creation is rejected if the target branch has advanced since the fork or if the source branch contains no new commits. Approval repeats the validation before applying a fast-forward update.

> **Compatibility notice:** Indexed pull-request ranges and atomic forks changed the ABI and are incompatible with older deployments. Pull-request descriptions are stored on-chain. After deploying a new `BitRegistry`, update `VITE_BIT_CONTRACT` in `bit-w3/.env.local` and restart the development server.

### 6. Run the Web3 explorer

The explorer is maintained in the independent [`opendasom/bit-w3`](https://github.com/opendasom/bit-w3) repository. It includes this repository as the `bit` submodule and consumes the same `BitRegistry` ABI as the CLI.

```bash
git clone --recurse-submodules https://github.com/opendasom/bit-w3.git
cd bit-w3
npm ci
cp .env.example .env.local
npm run dev
```

Web documentation, issues, security reports, and all other contributions are coordinated through this `bit` repository.

## CLI reference

### `bit init`

```text
BIT_PRIVATE_KEY=<privkey> bit init --rpc <url> --contract <addr> [--ipfs <url>] [--name <repo-name>] [--description <text>] [--branch <branch>]
```

- Requires an existing `.git` directory; run `git init` first.
- Uploads repository metadata to IPFS, creates the on-chain repository, and writes `.bit/config.json`.
- Never writes the private key to `.bit/config.json`; `.bit/` is automatically added to `.git/info/exclude`.
- `--key` is deprecated and retained only for compatibility. Use `BIT_PRIVATE_KEY`.
- Defaults the display name to the current directory name when `--name` is omitted.
- Defaults the Web3 explorer branch to `main` when `--branch` is omitted.

### `bit remote add`

```text
bit remote add <name> <url>
```

URL format: `bit://<network>/<contractAddress>/<repoId>`

### `bit push`

```text
bit push <remote>
```

- Detects the current branch automatically.
- Pushes commits after the current on-chain head in order.
- Uploads each commit's diff and manifest to IPFS before recording it on-chain.

### `bit pull`

```text
bit pull <remote> <branch>
```

- Rejects histories where local HEAD is absent from the remote history.
- Downloads missing commits from IPFS and reproduces their original Git hashes.
- Validates every manifest, diff, and parent before reconstructing Git objects in an isolated index, then updates the branch once.
- Refuses to run with a dirty working tree.

### `bit clone`

```text
bit clone <bit-url> [directory] --rpc <url> [--ipfs <url>] [--branch <branch>]
```

- Restores an existing repository without a private key.
- Creates the `origin` remote and local configuration before checking out the verified branch.

## Authorization model

| Role | Permissions |
|---|---|
| Owner | Assign roles through the explorer or `setRole`; the final Owner cannot be removed. |
| Maintainer | Push with `recordCommit`, and approve or reject pull requests. |
| Contributor | Explicit participant designation without write permission. |
| None | Read-only access. |

The repository creator becomes an Owner, and Owners include Maintainer permissions.

## Current limitations and security boundaries

- Push and merge support linear history only; merge commits are rejected.
- Atomic forks and individual pull requests process at most 64 commits.
- The Web3 commit view displays the 50 most recent commits on the selected branch.
- IPFS data is public and unencrypted. Anyone who knows a CID can read its diff and metadata.
- Persistent CID availability depends on at least one IPFS node pinning and serving the data.
- The protocol v2 client is incompatible with older `BitRegistry` deployments and checks `PROTOCOL_VERSION` when connecting.

## Verification

```bash
go test ./...
go vet ./...
forge test
npm ci
npm run compile
```

The CLI end-to-end test guide is available in [English](tests/e2e/cli/README.md) and [Korean](tests/e2e/cli/README.ko.md).

## Project structure

```text
bit/
├── cmd/
│   └── bit/
│       └── main.go   # bit CLI entry point
├── internal/
│   ├── cli/          # Cobra commands and CLI input/output
│   ├── app/          # Command execution logic
│   ├── chain/        # BitRegistry integration through go-ethereum
│   ├── git/          # Git repository access through go-git and Git
│   ├── ipfs/         # IPFS HTTP API client
│   ├── cid/          # Dependency-free CIDv0 to bytes32 conversion
│   ├── manifest/     # Manifest JSON encoding and decoding
│   └── config/       # .bit/config.json management
├── contracts/
│   ├── src/BitRegistry.sol   # Solidity contracts
│   ├── tests/                # Foundry contract tests
│   └── scripts/              # CLI-compatible ABI artifact generation
├── tests/
│   └── e2e/cli/              # CLI end-to-end tests
└── package.json              # Node.js package for ABI generation
```

## Primary dependencies

| Package | Purpose | License |
|---|---|---|
| `go-ethereum v1.17.0` | Ethereum client and ABI bindings | LGPL-3.0 (library) |
| `go-git/v5 v5.19.1` | Read Git repositories | Apache-2.0 |
| `cobra v1.8.1` | CLI framework | Apache-2.0 |
| `golang.org/x/crypto v0.51.0` | Cryptographic utilities | BSD-3-Clause |

## Development

```bash
go test ./...
go vet ./...

cd contracts && forge test && cd ..

npm ci
npm run compile
```

See [CONTRIBUTING.md](CONTRIBUTING.md) to contribute and [SECURITY.md](SECURITY.md) to report a vulnerability.

## License

Bit source code authored in this repository is distributed under the [MIT License](LICENSE). Third-party components retain their respective licenses. In particular, distributions of the CLI binary linked with `go-ethereum` must comply with LGPL-3.0 obligations.

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
  <a href="https://github.com/opendasom/bit/wiki">Wiki</a> ·
  <a href="CONTRIBUTING.md">Contributing</a> ·
  <a href="SECURITY.md">Security</a>
</p>

Bit is an experimental distributed version-control protocol built on IPFS and Ethereum. Commit diffs and metadata live on IPFS; `BitRegistry` records the corresponding repository state on Ethereum.

> [!WARNING]
> **Alpha software.** Bit has not received a production security audit. Use a disposable wallet and non-confidential source only. IPFS content requires pinning or replication to remain available.

<hr>

## Architecture

| Layer | Responsibility |
|---|---|
| **Git client** | Creates local repositories and commits; the `bit` CLI synchronizes protocol state. |
| **IPFS** | Stores repository metadata, diffs, and manifests as content-addressed objects. |
| **Ethereum** | Records Git commit-based branch heads, manifest/diff CID digests, and roles in `BitRegistry`. |
| **Web explorer** | Browses protocol state and signs role, fork, and pull-request operations with MetaMask. |

<p align="center">
  <img src="docs/assets/bit-architecture.png" alt="Bit architecture showing participants, the local Git and CLI environment, IPFS, Ethereum, and the Web3 client" width="100%" />
</p>

See the [architecture and data model](https://github.com/opendasom/bit/wiki/Architecture-and-Data-Model) for integrity checks, on-chain records, roles, and protocol limits.

### Web explorer

Browse repositories and sign registry operations at the deployed [Bit Web3 explorer](https://bitweb.space/).

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

For capabilities, wallet safety, and local setup, see the [Web3 Explorer guide](https://github.com/opendasom/bit/wiki/Web3-Explorer).
## Bit CLI Demos

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

<hr>

## Quick start

Build the CLI:

```bash
git clone https://github.com/opendasom/bit.git
cd bit
go build -o bit ./cmd/bit
```

The complete local walkthrough starts Anvil and IPFS, deploys `BitRegistry`, creates a repository, pushes a commit, and verifies it by cloning. Follow the [Quick Start guide](https://github.com/opendasom/bit/wiki/Quick-Start).

## Documentation

| Need | Guide |
|---|---|
| Create a local repository and verify a clone | [Quick Start](https://github.com/opendasom/bit/wiki/Quick-Start) |
| Command syntax, configuration, and credentials | [CLI Reference](https://github.com/opendasom/bit/wiki/CLI-Reference) |
| Protocol records, verification, roles, and limits | [Architecture and Data Model](https://github.com/opendasom/bit/wiki/Architecture-and-Data-Model) |
| Explorer use and local web setup | [Web3 Explorer](https://github.com/opendasom/bit/wiki/Web3-Explorer) |
| Privacy, wallet safety, and disclosure | [Security and Privacy](https://github.com/opendasom/bit/wiki/Security-and-Privacy) |
| Tests, ABI workflow, and code map | [Development Guide](https://github.com/opendasom/bit/wiki/Development-Guide) |
| Setup and command failures | [Troubleshooting](https://github.com/opendasom/bit/wiki/Troubleshooting) |

## Contributing and security

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request. All pull requests should target `develop` unless a maintainer requests otherwise. Report vulnerabilities through [SECURITY.md](SECURITY.md), not public issues.

## License

Bit source code authored in this repository is distributed under the [MIT License](LICENSE). Third-party components retain their respective licenses.

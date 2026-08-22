# Contributing to Bit

Thanks for helping build verifiable, portable source history.

## Before you start

- Discuss substantial protocol or smart-contract changes in an issue first.
- Keep pull requests focused and include tests for changed behaviour.
- Never commit private keys, RPC credentials, IPFS credentials, or generated
  build output outside the checked-in contract artifact.

## Development checks

Requirements: Go 1.25+, Node.js 20.19+, npm, Foundry, Git, and optionally
Kubo for live IPFS testing.

Run the relevant checks before opening a pull request:

```bash
go vet ./...
go test ./...
npm ci
npm run compile
forge fmt --check
forge test
```

`BIT_IPFS_API=http://127.0.0.1:5001 go test ./internal/ipfs` additionally
exercises a running local Kubo node.

## Smart-contract changes

Changes under `contracts/src` must update the checked-in ABI artifact used by
the Go CLI and by the web explorer through the `bit-w3` submodule:

```bash
npm run compile
git diff -- internal/chain/artifacts/BitRegistry.json
```

The CI workflow verifies this artifact is current. Treat deployed contracts as
immutable: document migration steps whenever an ABI or storage change requires
a new deployment.

## Web3 repository

The browser application is maintained in
[`opendasom/bit-w3`](https://github.com/opendasom/bit-w3). Its source checkout
contains this repository as the `bit` submodule and has an independent Node.js
package and CI workflow. Documentation, issues, security reports, and all other
contributions for both repositories are coordinated through this `bit`
repository.

## Pull requests

Use a concise title, explain the user-visible effect, and note validation you
ran. Maintainers may request tests, documentation, or a compatibility note
before merging.

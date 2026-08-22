# Security policy

## Supported versions

Security fixes are applied to the latest development version of Bit. Versions
published before the first stable release are experimental and should not be
used to protect valuable assets or private source code.

## Reporting a vulnerability

Please report suspected vulnerabilities through GitHub's **Report a
vulnerability** flow for this repository. Do not open a public issue or
include a proof of concept in a pull request.

Include the affected component, reproducible steps, potential impact, and any
suggested mitigation. We will acknowledge a report within 7 days and provide a
status update after triage.

If private vulnerability reporting is unavailable in the repository settings,
contact a maintainer through GitHub before sharing technical details publicly.

## Scope and safety

The smart contracts, CLI, and the web explorer maintained in the `bit-w3`
repository are in scope, along with their build scripts and CI configuration.
Never test against mainnet contracts or wallets that hold real
value. Use a local Anvil chain or a disposable testnet wallet.

Bit stores commit diffs and manifests on public content-addressed storage. Do
not treat it as a private-code hosting service, and never include credentials,
secrets, or production keys in a repository you push with Bit.

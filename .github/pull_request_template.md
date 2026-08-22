## Summary

<!-- Explain the change and its user-visible effect. -->

## Related issue

<!-- Use "Closes #123" when applicable. Substantial protocol and contract changes should have a related issue. -->

## Changes

-

## Validation

<!-- Check every command you ran. Explain omissions below. -->

- [ ] `go vet ./...`
- [ ] `go test ./...`
- [ ] `npm ci && npm run compile`
- [ ] `forge fmt --check`
- [ ] `forge test`
- [ ] Other relevant checks are described below.

Validation notes:

## Compatibility and security

- [ ] This change does not commit private keys, credentials, sensitive source data, or generated build output outside the checked-in contract artifact.
- [ ] User-visible behavior and operational or migration steps are documented where needed.
- [ ] Contract changes include an updated `internal/chain/artifacts/BitRegistry.json` generated with `npm run compile`.
- [ ] ABI, storage, protocol, CLI, and `bit-w3` compatibility effects are described below.
- [ ] Security-sensitive details have been reported privately rather than disclosed in this pull request.

Compatibility or security notes:

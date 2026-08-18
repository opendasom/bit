// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {PullRequestRegistry} from "./PullRequestRegistry.sol";

/// @notice On-chain git registry. Behaviour lives in the role-specific bases:
/// RepoRegistry (repos + roles) -> CommitRegistry (commits, branches, tags) -> PullRequestRegistry (PRs).
contract BitRegistry is PullRequestRegistry {
    uint256 public constant PROTOCOL_VERSION = 2;
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {PullRequestRegistry} from "./PullRequestRegistry.sol";

/// @notice On-chain Git registry.
contract BitRegistry is PullRequestRegistry {
    uint256 public constant PROTOCOL_VERSION = 2;
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {BitRegistryTypes} from "./BitRegistryTypes.sol";

abstract contract BitRegistryEvents is BitRegistryTypes {
    event RepoCreated(uint256 indexed repoId, address indexed owner, bytes metadataCID);
    event RepoForked(uint256 indexed repoId, uint256 indexed sourceRepoId, bytes32 indexed sourceBranch, address owner);
    event RoleChanged(uint256 indexed repoId, address indexed user, Role role);
    event CommitRecorded(
        uint256 indexed repoId,
        bytes32 indexed branch,
        bytes20 indexed commitHash,
        bytes20 treeHash,
        bytes20[] parents,
        bytes32 manifestDigest,
        bytes32 diffDigest,
        address updater
    );
    event BranchUpdated(
        uint256 indexed repoId,
        bytes32 indexed branch,
        bytes oldHead,
        bytes newHead,
        bytes gitCommit,
        bytes previousCommit,
        address indexed updater
    );
    event TagCreated(uint256 indexed repoId, bytes32 indexed tag, bytes target, address indexed creator);
    event PullRequestCreated(
        uint256 indexed prId,
        uint256 indexed targetRepoId,
        uint256 indexed sourceRepoId,
        bytes32 targetBranch,
        bytes32 sourceBranch,
        bytes20 baseCommit,
        bytes20 sourceHeadCommit,
        address author,
        bytes description
    );
    event PullRequestApproved(
        uint256 indexed prId,
        uint256 indexed targetRepoId,
        bytes32 indexed targetBranch,
        bytes20 oldHead,
        bytes20 newHead,
        address approver
    );
    event PullRequestRejected(uint256 indexed prId, address indexed rejectedBy);
    event PullRequestClosed(uint256 indexed prId, address indexed closedBy);
}

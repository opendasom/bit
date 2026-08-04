// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @notice Shared enums and structs. Declared in a base contract (not file level)
/// so callers can keep referring to them as `BitRegistry.PullRequestStatus`.
abstract contract BitRegistryTypes {
    enum Role {
        None,
        Contributor,
        Maintainer,
        Owner
    }

    enum PullRequestStatus {
        None,
        Open,
        Approved,
        Rejected,
        Closed
    }

    struct CommitRecord {
        bytes20 treeHash;
        bytes32 manifestDigest;
        bytes32 diffDigest;
        address updater;
        uint256 timestamp;
        bytes20[] parents;
        bool exists;
    }

    struct Repo {
        address owner;
        bytes metadataCID;
        mapping(address => Role) roles;
        mapping(bytes32 => bytes20) branchCommits;
        mapping(bytes32 => bytes20[]) branchHistory;
        mapping(bytes20 => CommitRecord) commits;
        mapping(bytes32 => bytes) tags;
        mapping(bytes32 => bool) tagExists;
    }

    struct PullRequest {
        uint256 id;
        uint256 targetRepoId;
        bytes32 targetBranch;
        uint256 sourceRepoId;
        bytes32 sourceBranch;
        bytes20 baseCommit;
        bytes20 sourceHeadCommit;
        address author;
        PullRequestStatus status;
        uint256 createdAt;
        uint256 updatedAt;
    }
}

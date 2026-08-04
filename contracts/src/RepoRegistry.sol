// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {BitRegistryErrors} from "./BitRegistryErrors.sol";
import {BitRegistryEvents} from "./BitRegistryEvents.sol";

/// @notice Repo lifecycle and access control: creation, metadata, roles.
abstract contract RepoRegistry is BitRegistryEvents, BitRegistryErrors {
    uint256 public nextRepoId = 1;
    mapping(uint256 => Repo) internal repos;

    modifier repoExists(uint256 repoId) {
        if (repos[repoId].owner == address(0)) revert RepoNotFound();
        _;
    }

    modifier onlyOwner(uint256 repoId) {
        if (repos[repoId].roles[msg.sender] != Role.Owner) revert OwnerRequired();
        _;
    }

    modifier onlyMaintainer(uint256 repoId) {
        Role role = repos[repoId].roles[msg.sender];
        if (role != Role.Maintainer && role != Role.Owner) revert MaintainerRequired();
        _;
    }

    function _isMaintainer(uint256 repoId, address user) internal view returns (bool) {
        Role role = repos[repoId].roles[user];
        return role == Role.Maintainer || role == Role.Owner;
    }

    function createRepo(bytes calldata metadataCID) external returns (uint256 repoId) {
        repoId = nextRepoId++;
        Repo storage repo = repos[repoId];
        repo.owner = msg.sender;
        repo.metadataCID = metadataCID;
        repo.roles[msg.sender] = Role.Owner;
        emit RepoCreated(repoId, msg.sender, metadataCID);
        emit RoleChanged(repoId, msg.sender, Role.Owner);
    }

    function getRepoCount() external view returns (uint256) {
        return nextRepoId - 1;
    }

    function getRepo(uint256 repoId) external view repoExists(repoId) returns (address owner, bytes memory metadataCID) {
        Repo storage repo = repos[repoId];
        return (repo.owner, repo.metadataCID);
    }

    function getRole(uint256 repoId, address user) external view repoExists(repoId) returns (Role) {
        return repos[repoId].roles[user];
    }

    function setRole(uint256 repoId, address user, Role role) external repoExists(repoId) onlyOwner(repoId) {
        if (user == address(0)) revert ZeroUser();
        repos[repoId].roles[user] = role;
        emit RoleChanged(repoId, user, role);
    }
}

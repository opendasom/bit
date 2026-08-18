// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {BitRegistryErrors} from "./BitRegistryErrors.sol";
import {BitRegistryEvents} from "./BitRegistryEvents.sol";

/// @notice Repo lifecycle and access control: creation, metadata, roles.
abstract contract RepoRegistry is BitRegistryEvents, BitRegistryErrors {
    uint256 public constant MAX_REPO_METADATA_CID_LENGTH = 128;
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
        return _createRepo(msg.sender, metadataCID);
    }

    function _createRepo(address owner, bytes memory metadataCID) internal returns (uint256 repoId) {
        if (metadataCID.length > MAX_REPO_METADATA_CID_LENGTH) revert MetadataTooLong();
        repoId = nextRepoId++;
        Repo storage repo = repos[repoId];
        repo.owner = owner;
        repo.ownerCount = 1;
        repo.metadataCID = metadataCID;
        repo.roles[owner] = Role.Owner;
        emit RepoCreated(repoId, owner, metadataCID);
        emit RoleChanged(repoId, owner, Role.Owner);
    }

    function getRepoCount() external view returns (uint256) {
        return nextRepoId - 1;
    }

    function getRepo(uint256 repoId)
        external
        view
        repoExists(repoId)
        returns (address owner, bytes memory metadataCID)
    {
        Repo storage repo = repos[repoId];
        return (repo.owner, repo.metadataCID);
    }

    function getRepos(uint256 start, uint256 limit)
        external
        view
        returns (uint256[] memory repoIds, address[] memory owners, bytes[] memory metadataCIDs)
    {
        uint256 total = nextRepoId - 1;
        if (start >= total || limit == 0) {
            return (new uint256[](0), new address[](0), new bytes[](0));
        }
        uint256 count = total - start;
        if (count > limit) count = limit;
        repoIds = new uint256[](count);
        owners = new address[](count);
        metadataCIDs = new bytes[](count);
        for (uint256 i = 0; i < count; i++) {
            uint256 repoId = start + i + 1;
            Repo storage repo = repos[repoId];
            repoIds[i] = repoId;
            owners[i] = repo.owner;
            metadataCIDs[i] = repo.metadataCID;
        }
    }

    function getRole(uint256 repoId, address user) external view repoExists(repoId) returns (Role) {
        return repos[repoId].roles[user];
    }

    function setRole(uint256 repoId, address user, Role role) external repoExists(repoId) onlyOwner(repoId) {
        if (user == address(0)) revert ZeroUser();
        Repo storage repo = repos[repoId];
        Role previousRole = repo.roles[user];
        if (previousRole == role) return;
        if (previousRole == Role.Owner && role != Role.Owner) {
            if (repo.ownerCount == 1) revert LastOwnerRequired();
            repo.ownerCount--;
        } else if (previousRole != Role.Owner && role == Role.Owner) {
            repo.ownerCount++;
        }
        repo.roles[user] = role;
        emit RoleChanged(repoId, user, role);
    }
}

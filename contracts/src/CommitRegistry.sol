// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {RepoRegistry} from "./RepoRegistry.sol";

/// @notice Refs and history: commits, branch heads, branch history, tags.
abstract contract CommitRegistry is RepoRegistry {
    uint256 public constant MAX_FORK_COMMITS = 64;

    function recordCommit(
        uint256 repoId,
        bytes32 branch,
        bytes20 expectedOldCommit,
        bytes20 commitHash,
        bytes20 treeHash,
        bytes20[] calldata parents,
        bytes32 manifestDigest,
        bytes32 diffDigest
    ) external repoExists(repoId) onlyMaintainer(repoId) {
        if (commitHash == bytes20(0)) revert ZeroCommit();
        if (branch == bytes32(0)) revert ZeroBranch();
        if (treeHash == bytes20(0)) revert ZeroTree();
        if (manifestDigest == bytes32(0) || diffDigest == bytes32(0)) revert ZeroDigest();
        if (parents.length > 1) revert MergeCommitNotSupported();

        Repo storage repo = repos[repoId];
        bytes20 currentCommit = repo.branchCommits[branch];
        if (currentCommit != expectedOldCommit) revert StaleBranchHead();
        if (currentCommit == bytes20(0)) {
            if (parents.length != 0) revert RootCommitHasParent();
        } else {
            if (parents.length == 0) revert MissingParent();
            if (parents[0] != currentCommit) revert FirstParentMismatch();
        }

        CommitRecord storage item = repo.commits[commitHash];
        if (item.exists) {
            _assertCommitMetadata(item, treeHash, manifestDigest, diffDigest, parents);
        } else {
            item.treeHash = treeHash;
            item.manifestDigest = manifestDigest;
            item.diffDigest = diffDigest;
            item.updater = msg.sender;
            item.timestamp = block.timestamp;
            item.exists = true;
            for (uint256 i = 0; i < parents.length; i++) {
                item.parents.push(parents[i]);
            }
        }

        bytes20 oldCommit = repo.branchCommits[branch];
        bytes memory oldHead =
            oldCommit == bytes20(0) ? bytes("") : abi.encodePacked(repo.commits[oldCommit].manifestDigest);
        bytes memory previousCommit = parents.length == 0 ? bytes("") : abi.encodePacked(parents[0]);
        _appendBranchCommit(repo, branch, commitHash);

        emit CommitRecorded(repoId, branch, commitHash, treeHash, parents, manifestDigest, diffDigest, msg.sender);
        emit BranchUpdated(
            repoId,
            branch,
            oldHead,
            abi.encodePacked(manifestDigest),
            abi.encodePacked(commitHash),
            previousCommit,
            msg.sender
        );
    }

    function forkRepo(uint256 sourceRepoId, bytes32 sourceBranch, bytes calldata metadataCID)
        external
        repoExists(sourceRepoId)
        returns (uint256 repoId)
    {
        if (sourceBranch == bytes32(0)) revert ZeroBranch();
        Repo storage sourceRepo = repos[sourceRepoId];
        bytes20[] storage sourceHistory = sourceRepo.branchHistory[sourceBranch];
        if (sourceHistory.length == 0) revert EmptySourceBranch();
        if (sourceHistory.length > MAX_FORK_COMMITS) revert ForkTooLarge();

        repoId = _createRepo(msg.sender, metadataCID);
        Repo storage targetRepo = repos[repoId];
        bytes20 currentHead;
        for (uint256 i = 0; i < sourceHistory.length; i++) {
            bytes20 commitHash = sourceHistory[i];
            _copyCommitRecord(sourceRepo, targetRepo, commitHash);
            CommitRecord storage item = targetRepo.commits[commitHash];
            _appendBranchCommit(targetRepo, sourceBranch, commitHash);
            emit CommitRecorded(
                repoId,
                sourceBranch,
                commitHash,
                item.treeHash,
                _parentsToMemory(item),
                item.manifestDigest,
                item.diffDigest,
                msg.sender
            );
            emit BranchUpdated(
                repoId,
                sourceBranch,
                currentHead == bytes20(0)
                    ? bytes("")
                    : abi.encodePacked(targetRepo.commits[currentHead].manifestDigest),
                abi.encodePacked(item.manifestDigest),
                abi.encodePacked(commitHash),
                currentHead == bytes20(0) ? bytes("") : abi.encodePacked(currentHead),
                msg.sender
            );
            currentHead = commitHash;
        }
        emit RepoForked(repoId, sourceRepoId, sourceBranch, msg.sender);
    }

    function getBranchHead(uint256 repoId, bytes32 branch) external view repoExists(repoId) returns (bytes memory) {
        bytes20 commitHash = repos[repoId].branchCommits[branch];
        if (commitHash == bytes20(0)) {
            return bytes("");
        }
        return abi.encodePacked(repos[repoId].commits[commitHash].manifestDigest);
    }

    function getBranchCommit(uint256 repoId, bytes32 branch) external view repoExists(repoId) returns (bytes20) {
        return repos[repoId].branchCommits[branch];
    }

    function getBranchHistoryLength(uint256 repoId, bytes32 branch) external view repoExists(repoId) returns (uint256) {
        return repos[repoId].branchHistory[branch].length;
    }

    function getRepoBranchCount(uint256 repoId) external view repoExists(repoId) returns (uint256) {
        return repos[repoId].branchKeys.length;
    }

    function getRepoBranches(uint256 repoId, uint256 start, uint256 limit)
        external
        view
        repoExists(repoId)
        returns (
            bytes32[] memory branchKeys,
            bytes20[] memory headCommits,
            uint256[] memory historyLengths,
            bytes32[] memory headManifestDigests
        )
    {
        Repo storage repo = repos[repoId];
        uint256 total = repo.branchKeys.length;
        if (start >= total || limit == 0) {
            return (new bytes32[](0), new bytes20[](0), new uint256[](0), new bytes32[](0));
        }

        uint256 count = total - start;
        if (count > limit) count = limit;
        branchKeys = new bytes32[](count);
        headCommits = new bytes20[](count);
        historyLengths = new uint256[](count);
        headManifestDigests = new bytes32[](count);

        for (uint256 i = 0; i < count; i++) {
            bytes32 branch = repo.branchKeys[start + i];
            bytes20 headCommit = repo.branchCommits[branch];
            branchKeys[i] = branch;
            headCommits[i] = headCommit;
            historyLengths[i] = repo.branchHistory[branch].length;
            headManifestDigests[i] = repo.commits[headCommit].manifestDigest;
        }
    }

    function getBranchCommitAt(uint256 repoId, bytes32 branch, uint256 index)
        external
        view
        repoExists(repoId)
        returns (bytes20)
    {
        return repos[repoId].branchHistory[branch][index];
    }

    function getBranchCommitsWithMetadata(uint256 repoId, bytes32 branch, uint256 start, uint256 limit)
        external
        view
        repoExists(repoId)
        returns (
            bytes20[] memory commitHashes,
            bytes20[] memory treeHashes,
            bytes32[] memory manifestDigests,
            bytes32[] memory diffDigests
        )
    {
        bytes20[] storage history = repos[repoId].branchHistory[branch];
        if (start >= history.length || limit == 0) {
            return (new bytes20[](0), new bytes20[](0), new bytes32[](0), new bytes32[](0));
        }
        uint256 count = history.length - start;
        if (count > limit) {
            count = limit;
        }
        commitHashes = new bytes20[](count);
        treeHashes = new bytes20[](count);
        manifestDigests = new bytes32[](count);
        diffDigests = new bytes32[](count);
        for (uint256 i = 0; i < count; i++) {
            bytes20 commitHash = history[start + i];
            CommitRecord storage item = repos[repoId].commits[commitHash];
            commitHashes[i] = commitHash;
            treeHashes[i] = item.treeHash;
            manifestDigests[i] = item.manifestDigest;
            diffDigests[i] = item.diffDigest;
        }
    }

    function getCommit(uint256 repoId, bytes20 commitHash)
        external
        view
        repoExists(repoId)
        returns (bytes20 treeHash, bytes32 manifestDigest, bytes32 diffDigest, address updater, uint256 timestamp)
    {
        CommitRecord storage item = repos[repoId].commits[commitHash];
        if (!item.exists) revert CommitNotFound();
        return (item.treeHash, item.manifestDigest, item.diffDigest, item.updater, item.timestamp);
    }

    function getCommitParentCount(uint256 repoId, bytes20 commitHash)
        external
        view
        repoExists(repoId)
        returns (uint256)
    {
        CommitRecord storage item = repos[repoId].commits[commitHash];
        if (!item.exists) revert CommitNotFound();
        return item.parents.length;
    }

    function getCommitParentAt(uint256 repoId, bytes20 commitHash, uint256 index)
        external
        view
        repoExists(repoId)
        returns (bytes20)
    {
        CommitRecord storage item = repos[repoId].commits[commitHash];
        if (!item.exists) revert CommitNotFound();
        return item.parents[index];
    }

    function getBranchHistoryAt(uint256 repoId, bytes32 branch, uint256 index)
        external
        view
        repoExists(repoId)
        returns (
            bytes memory oldHead,
            bytes memory newHead,
            bytes memory gitCommit,
            bytes memory previousCommit,
            address updater,
            uint256 timestamp
        )
    {
        Repo storage repo = repos[repoId];
        bytes20 commitHash = repo.branchHistory[branch][index];
        CommitRecord storage item = repo.commits[commitHash];
        bytes20 previous;
        if (item.parents.length > 0) {
            previous = item.parents[0];
            oldHead = abi.encodePacked(repo.commits[previous].manifestDigest);
            previousCommit = abi.encodePacked(previous);
        }
        return (
            oldHead,
            abi.encodePacked(item.manifestDigest),
            abi.encodePacked(commitHash),
            previousCommit,
            item.updater,
            item.timestamp
        );
    }

    function createTag(uint256 repoId, bytes32 tag, bytes calldata target)
        external
        repoExists(repoId)
        onlyMaintainer(repoId)
    {
        if (repos[repoId].tagExists[tag]) revert TagExists();
        repos[repoId].tagExists[tag] = true;
        repos[repoId].tags[tag] = target;
        emit TagCreated(repoId, tag, target, msg.sender);
    }

    function getTag(uint256 repoId, bytes32 tag) external view repoExists(repoId) returns (bytes memory) {
        if (!repos[repoId].tagExists[tag]) revert TagNotFound();
        return repos[repoId].tags[tag];
    }

    /// @dev Rejects conflicting metadata while copying commit ancestry.
    function _copyCommitRecord(Repo storage sourceRepo, Repo storage targetRepo, bytes20 commitHash) internal {
        CommitRecord storage source = sourceRepo.commits[commitHash];
        if (!source.exists) revert CommitNotFound();

        CommitRecord storage target = targetRepo.commits[commitHash];
        if (target.exists) {
            if (
                target.treeHash != source.treeHash || target.manifestDigest != source.manifestDigest
                    || target.diffDigest != source.diffDigest || target.parents.length != source.parents.length
            ) {
                revert CommitMetadataMismatch();
            }
            for (uint256 i = 0; i < source.parents.length; i++) {
                if (target.parents[i] != source.parents[i]) revert CommitMetadataMismatch();
            }
            return;
        }

        target.treeHash = source.treeHash;
        target.manifestDigest = source.manifestDigest;
        target.diffDigest = source.diffDigest;
        target.updater = source.updater;
        target.timestamp = source.timestamp;
        target.exists = true;
        for (uint256 i = 0; i < source.parents.length; i++) {
            target.parents.push(source.parents[i]);
        }
    }

    function _assertCommitMetadata(
        CommitRecord storage item,
        bytes20 treeHash,
        bytes32 manifestDigest,
        bytes32 diffDigest,
        bytes20[] calldata parents
    ) private view {
        if (
            item.treeHash != treeHash || item.manifestDigest != manifestDigest || item.diffDigest != diffDigest
                || item.parents.length != parents.length
        ) revert CommitMetadataMismatch();
        for (uint256 i = 0; i < parents.length; i++) {
            if (item.parents[i] != parents[i]) revert CommitMetadataMismatch();
        }
    }

    function _appendBranchCommit(Repo storage repo, bytes32 branch, bytes20 commitHash) internal {
        if (repo.branchCommitPositions[branch][commitHash] != 0) revert CommitAlreadyOnBranch();
        _registerBranch(repo, branch);
        repo.branchHistory[branch].push(commitHash);
        repo.branchCommitPositions[branch][commitHash] = repo.branchHistory[branch].length;
        repo.branchCommits[branch] = commitHash;
    }

    function _registerBranch(Repo storage repo, bytes32 branch) internal {
        if (repo.branchExists[branch]) return;
        repo.branchExists[branch] = true;
        repo.branchKeys.push(branch);
    }

    function _parentsToMemory(CommitRecord storage item) internal view returns (bytes20[] memory parents) {
        parents = new bytes20[](item.parents.length);
        for (uint256 i = 0; i < item.parents.length; i++) {
            parents[i] = item.parents[i];
        }
    }
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {CommitRegistry} from "./CommitRegistry.sol";

/// @notice Pull request lifecycle: create, approve (fast-forward merge), reject, close.
abstract contract PullRequestRegistry is CommitRegistry {
    uint256 public nextPullRequestId = 1;
    mapping(uint256 => PullRequest) private pullRequests;
    mapping(uint256 => uint256[]) private repoPullRequests;
    mapping(uint256 => uint256[]) private sourceRepoPullRequests;

    uint256 public constant MAX_PR_DESCRIPTION_LENGTH = 2048;

    function createPullRequest(
        uint256 targetRepoId,
        bytes32 targetBranch,
        uint256 sourceRepoId,
        bytes32 sourceBranch,
        bytes calldata description
    ) external repoExists(targetRepoId) repoExists(sourceRepoId) returns (uint256 prId) {
        if (description.length > MAX_PR_DESCRIPTION_LENGTH) revert DescriptionTooLong();

        Repo storage targetRepo = repos[targetRepoId];
        Repo storage sourceRepo = repos[sourceRepoId];
        bytes20 baseCommit = targetRepo.branchCommits[targetBranch];
        bytes20 sourceHeadCommit = sourceRepo.branchCommits[sourceBranch];
        if (sourceHeadCommit == bytes20(0)) revert EmptySourceBranch();

        _getPullRequestCommitRange(sourceRepo, sourceBranch, baseCommit, sourceHeadCommit);

        prId = nextPullRequestId++;
        pullRequests[prId] = PullRequest({
            id: prId,
            targetRepoId: targetRepoId,
            targetBranch: targetBranch,
            sourceRepoId: sourceRepoId,
            sourceBranch: sourceBranch,
            baseCommit: baseCommit,
            sourceHeadCommit: sourceHeadCommit,
            author: msg.sender,
            status: PullRequestStatus.Open,
            createdAt: block.timestamp,
            updatedAt: block.timestamp,
            description: description
        });
        repoPullRequests[targetRepoId].push(prId);
        sourceRepoPullRequests[sourceRepoId].push(prId);

        emit PullRequestCreated(
            prId,
            targetRepoId,
            sourceRepoId,
            targetBranch,
            sourceBranch,
            baseCommit,
            sourceHeadCommit,
            msg.sender,
            description
        );
    }

    function approvePullRequest(uint256 prId) external {
        PullRequest storage pr = _requireOpenPullRequest(prId);
        if (!_isMaintainer(pr.targetRepoId, msg.sender)) revert MaintainerRequired();

        Repo storage targetRepo = repos[pr.targetRepoId];
        Repo storage sourceRepo = repos[pr.sourceRepoId];
        bytes20 oldHead = targetRepo.branchCommits[pr.targetBranch];
        if (oldHead != pr.baseCommit) revert StaleBranchHead();

        (uint256 start, uint256 end) =
            _getPullRequestCommitRange(sourceRepo, pr.sourceBranch, pr.baseCommit, pr.sourceHeadCommit);

        _registerBranch(targetRepo, pr.targetBranch);
        bytes20 currentHead = oldHead;
        bytes20 newHead = oldHead;
        bytes20[] storage sourceHistory = sourceRepo.branchHistory[pr.sourceBranch];
        for (uint256 i = start; i < end; i++) {
            bytes20 commitHash = sourceHistory[i];
            _copyCommitRecord(sourceRepo, targetRepo, commitHash);
            CommitRecord storage item = targetRepo.commits[commitHash];
            if (item.parents.length > 1) revert MergeCommitNotSupported();
            if (currentHead != bytes20(0)) {
                if (item.parents.length == 0) revert MissingParent();
                if (item.parents[0] != currentHead) revert FirstParentMismatch();
            }

            targetRepo.branchCommits[pr.targetBranch] = commitHash;
            targetRepo.branchHistory[pr.targetBranch].push(commitHash);
            newHead = commitHash;

            emit CommitRecorded(
                pr.targetRepoId,
                pr.targetBranch,
                commitHash,
                item.treeHash,
                _parentsToMemory(item),
                item.manifestDigest,
                item.diffDigest,
                msg.sender
            );
            emit BranchUpdated(
                pr.targetRepoId,
                pr.targetBranch,
                currentHead == bytes20(0) ? bytes("") : abi.encodePacked(targetRepo.commits[currentHead].manifestDigest),
                abi.encodePacked(item.manifestDigest),
                abi.encodePacked(commitHash),
                item.parents.length == 0 ? bytes("") : abi.encodePacked(item.parents[0]),
                msg.sender
            );

            currentHead = commitHash;
        }

        pr.status = PullRequestStatus.Approved;
        pr.updatedAt = block.timestamp;
        emit PullRequestApproved(prId, pr.targetRepoId, pr.targetBranch, oldHead, newHead, msg.sender);
    }

    function rejectPullRequest(uint256 prId) external {
        PullRequest storage pr = _requireOpenPullRequest(prId);
        if (!_isMaintainer(pr.targetRepoId, msg.sender)) revert MaintainerRequired();

        pr.status = PullRequestStatus.Rejected;
        pr.updatedAt = block.timestamp;
        emit PullRequestRejected(prId, msg.sender);
    }

    function closePullRequest(uint256 prId) external {
        PullRequest storage pr = _requireOpenPullRequest(prId);
        if (msg.sender != pr.author && !_isMaintainer(pr.targetRepoId, msg.sender)) {
            revert UnauthorizedPullRequestAction();
        }

        pr.status = PullRequestStatus.Closed;
        pr.updatedAt = block.timestamp;
        emit PullRequestClosed(prId, msg.sender);
    }

    function getPullRequest(uint256 prId) external view returns (PullRequest memory) {
        PullRequest storage pr = pullRequests[prId];
        if (pr.status == PullRequestStatus.None) revert PullRequestNotFound();
        return pr;
    }

    function getRepoPullRequestCount(uint256 repoId) external view repoExists(repoId) returns (uint256) {
        return repoPullRequests[repoId].length;
    }

    function getRepoPullRequestAt(uint256 repoId, uint256 index) external view repoExists(repoId) returns (uint256) {
        return repoPullRequests[repoId][index];
    }

    function getRepoPullRequestIds(uint256 repoId, uint256 start, uint256 limit)
        external
        view
        repoExists(repoId)
        returns (uint256[] memory)
    {
        return _slicePullRequestIds(repoPullRequests[repoId], start, limit);
    }

    function getSourceRepoPullRequestCount(uint256 repoId) external view repoExists(repoId) returns (uint256) {
        return sourceRepoPullRequests[repoId].length;
    }

    function getSourceRepoPullRequestAt(uint256 repoId, uint256 index)
        external
        view
        repoExists(repoId)
        returns (uint256)
    {
        return sourceRepoPullRequests[repoId][index];
    }

    function getSourceRepoPullRequestIds(uint256 repoId, uint256 start, uint256 limit)
        external
        view
        repoExists(repoId)
        returns (uint256[] memory)
    {
        return _slicePullRequestIds(sourceRepoPullRequests[repoId], start, limit);
    }

    function _requireOpenPullRequest(uint256 prId) private view returns (PullRequest storage pr) {
        pr = pullRequests[prId];
        if (pr.status == PullRequestStatus.None) revert PullRequestNotFound();
        if (pr.status != PullRequestStatus.Open) revert PullRequestNotOpen();
    }

    function _slicePullRequestIds(uint256[] storage values, uint256 start, uint256 limit)
        private
        view
        returns (uint256[] memory ids)
    {
        if (start >= values.length || limit == 0) return new uint256[](0);
        uint256 count = values.length - start;
        if (count > limit) count = limit;
        ids = new uint256[](count);
        for (uint256 i = 0; i < count; i++) {
            ids[i] = values[start + i];
        }
    }

    /// @dev Half-open range [start, end) of source branch history to fast-forward onto the target.
    function _getPullRequestCommitRange(
        Repo storage sourceRepo,
        bytes32 sourceBranch,
        bytes20 baseCommit,
        bytes20 sourceHeadCommit
    ) private view returns (uint256 start, uint256 end) {
        bytes20[] storage history = sourceRepo.branchHistory[sourceBranch];
        if (sourceHeadCommit == bytes20(0)) revert EmptySourceBranch();

        bool foundBase = baseCommit == bytes20(0);
        bool foundHead;
        start = 0;
        for (uint256 i = 0; i < history.length; i++) {
            bytes20 commitHash = history[i];
            if (!foundBase && commitHash == baseCommit) {
                foundBase = true;
                start = i + 1;
                continue;
            }
            if (foundBase && commitHash == sourceHeadCommit) {
                foundHead = true;
                end = i + 1;
                break;
            }
        }

        if (!foundBase) revert SourceHistoryDoesNotContainBase();
        if (!foundHead) revert SourceHeadNotFound();
        if (start >= end) revert NoCommitsToMerge();
    }
}

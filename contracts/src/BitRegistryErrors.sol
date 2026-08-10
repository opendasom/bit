// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/// @notice Every revert reason the registry can raise.
abstract contract BitRegistryErrors {
    error RepoNotFound();
    error OwnerRequired();
    error MaintainerRequired();
    error ZeroUser();
    error ZeroCommit();
    error StaleBranchHead();
    error MissingParent();
    error FirstParentMismatch();
    error CommitMetadataMismatch();
    error CommitNotFound();
    error TagExists();
    error TagNotFound();
    error PullRequestNotFound();
    error PullRequestNotOpen();
    error EmptySourceBranch();
    error NoCommitsToMerge();
    error SourceHistoryDoesNotContainBase();
    error SourceHeadNotFound();
    error UnauthorizedPullRequestAction();
    error MergeCommitNotSupported();
    error DescriptionTooLong();
}

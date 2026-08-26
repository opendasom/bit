// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

abstract contract BitRegistryErrors {
    error RepoNotFound();
    error OwnerRequired();
    error MaintainerRequired();
    error ZeroUser();
    error ZeroCommit();
    error ZeroBranch();
    error ZeroTree();
    error ZeroDigest();
    error StaleBranchHead();
    error MissingParent();
    error RootCommitHasParent();
    error FirstParentMismatch();
    error CommitAlreadyOnBranch();
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
    error MetadataTooLong();
    error LastOwnerRequired();
    error SourceRoleRequired();
    error TooManyPullRequestCommits();
    error ForkTooLarge();
}

// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {BitRegistry} from "../src/BitRegistry.sol";
import {BitRegistryTypes} from "../src/BitRegistryTypes.sol";

contract RegistryActor {
    BitRegistry private immutable registry;

    constructor(BitRegistry registry_) {
        registry = registry_;
    }

    function createRepo() external returns (uint256) {
        return registry.createRepo("");
    }

    function recordCommit(
        uint256 repoId,
        bytes32 branch,
        bytes20 expectedOldCommit,
        bytes20 commitHash,
        bytes20 treeHash,
        bytes20[] memory parents,
        bytes32 manifestDigest,
        bytes32 diffDigest
    ) external {
        registry.recordCommit(
            repoId, branch, expectedOldCommit, commitHash, treeHash, parents, manifestDigest, diffDigest
        );
    }

    function createPullRequest(
        uint256 targetRepoId,
        bytes32 targetBranch,
        uint256 sourceRepoId,
        bytes32 sourceBranch,
        bytes calldata description
    ) external returns (uint256) {
        return registry.createPullRequest(targetRepoId, targetBranch, sourceRepoId, sourceBranch, description);
    }

    function approvePullRequest(uint256 prId) external {
        registry.approvePullRequest(prId);
    }

    function rejectPullRequest(uint256 prId) external {
        registry.rejectPullRequest(prId);
    }

    function closePullRequest(uint256 prId) external {
        registry.closePullRequest(prId);
    }
}

contract BitRegistryPullRequestTest {
    BitRegistry private registry;
    RegistryActor private targetOwner;
    RegistryActor private sourceOwner;
    RegistryActor private stranger;

    bytes32 private constant MAIN = keccak256("main");

    bytes20 private constant A = hex"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
    bytes20 private constant B = hex"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";
    bytes20 private constant C = hex"cccccccccccccccccccccccccccccccccccccccc";
    bytes20 private constant D = hex"dddddddddddddddddddddddddddddddddddddddd";
    bytes20 private constant E = hex"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee";

    function setUp() public {
        registry = new BitRegistry();
        targetOwner = new RegistryActor(registry);
        sourceOwner = new RegistryActor(registry);
        stranger = new RegistryActor(registry);
    }

    function testCreatePullRequestStoresSnapshot() public {
        (uint256 targetRepoId, uint256 sourceRepoId) = _seedTargetAndSource();

        uint256 prId = sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, "adds feature X");
        BitRegistryTypes.PullRequest memory pr = registry.getPullRequest(prId);

        require(pr.id == prId, "wrong pr id");
        require(pr.targetRepoId == targetRepoId, "wrong target repo");
        require(pr.sourceRepoId == sourceRepoId, "wrong source repo");
        require(pr.baseCommit == B, "wrong base");
        require(pr.sourceHeadCommit == D, "wrong source head");
        require(pr.author == address(sourceOwner), "wrong author");
        require(pr.status == BitRegistryTypes.PullRequestStatus.Open, "wrong status");
        require(keccak256(pr.description) == keccak256("adds feature X"), "wrong description");
        require(registry.getRepoPullRequestCount(targetRepoId) == 1, "wrong pr count");
        require(registry.getRepoPullRequestAt(targetRepoId, 0) == prId, "wrong pr index");
    }

    function testCreatePullRequestRevertsWhenDescriptionTooLong() public {
        (uint256 targetRepoId, uint256 sourceRepoId) = _seedTargetAndSource();

        bytes memory long = new bytes(2049);
        try sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, long) {
            revert("expected description too long revert");
        } catch {}
    }

    function testApprovePullRequestFastForwardsTargetBranchAndCopiesMetadata() public {
        (uint256 targetRepoId, uint256 sourceRepoId) = _seedTargetAndSource();
        uint256 prId = sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, "");

        sourceOwner.recordCommit(sourceRepoId, MAIN, D, E, E, _parents(D), _digest(E, 1), _digest(E, 2));
        targetOwner.approvePullRequest(prId);

        require(registry.getBranchCommit(targetRepoId, MAIN) == D, "target did not fast-forward");
        require(registry.getBranchHistoryLength(targetRepoId, MAIN) == 4, "wrong target history length");
        require(registry.getBranchCommitAt(targetRepoId, MAIN, 2) == C, "missing first pr commit");
        require(registry.getBranchCommitAt(targetRepoId, MAIN, 3) == D, "missing second pr commit");

        (bytes20 treeHash, bytes32 manifestDigest, bytes32 diffDigest,,) = registry.getCommit(targetRepoId, C);
        require(treeHash == C, "tree was not copied");
        require(manifestDigest == _digest(C, 1), "manifest was not copied");
        require(diffDigest == _digest(C, 2), "diff was not copied");
        require(registry.getCommitParentCount(targetRepoId, C) == 1, "wrong copied parent count");
        require(registry.getCommitParentAt(targetRepoId, C, 0) == B, "wrong copied parent");

        BitRegistryTypes.PullRequest memory pr = registry.getPullRequest(prId);
        require(pr.status == BitRegistryTypes.PullRequestStatus.Approved, "pr was not approved");
    }

    function testApprovePullRequestRevertsWhenTargetMoved() public {
        (uint256 targetRepoId, uint256 sourceRepoId) = _seedTargetAndSource();
        uint256 prId = sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, "");
        targetOwner.recordCommit(targetRepoId, MAIN, B, E, E, _parents(B), _digest(E, 1), _digest(E, 2));

        try targetOwner.approvePullRequest(prId) {
            revert("expected stale branch revert");
        } catch {}
    }

    function testApprovePullRequestRequiresMaintainer() public {
        (uint256 targetRepoId, uint256 sourceRepoId) = _seedTargetAndSource();
        uint256 prId = sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, "");

        try stranger.approvePullRequest(prId) {
            revert("expected maintainer revert");
        } catch {}
    }

    function testCreatePullRequestRequiresSourceToContainTargetBase() public {
        uint256 targetRepoId = targetOwner.createRepo();
        uint256 sourceRepoId = sourceOwner.createRepo();
        _record(targetOwner, targetRepoId, A, bytes20(0));
        _record(sourceOwner, sourceRepoId, C, bytes20(0));

        try sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, "") {
            revert("expected base history revert");
        } catch {}
    }

    function testRejectAndClosePullRequest() public {
        (uint256 targetRepoId, uint256 sourceRepoId) = _seedTargetAndSource();
        uint256 rejectedPrId = sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, "");
        targetOwner.rejectPullRequest(rejectedPrId);
        require(
            registry.getPullRequest(rejectedPrId).status == BitRegistryTypes.PullRequestStatus.Rejected,
            "pr was not rejected"
        );

        uint256 closedPrId = sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, "");
        sourceOwner.closePullRequest(closedPrId);
        require(registry.getPullRequest(closedPrId).status == BitRegistryTypes.PullRequestStatus.Closed, "pr was not closed");
    }

    function _seedTargetAndSource() private returns (uint256 targetRepoId, uint256 sourceRepoId) {
        targetRepoId = targetOwner.createRepo();
        sourceRepoId = sourceOwner.createRepo();

        _record(targetOwner, targetRepoId, A, bytes20(0));
        _record(targetOwner, targetRepoId, B, A);

        _record(sourceOwner, sourceRepoId, A, bytes20(0));
        _record(sourceOwner, sourceRepoId, B, A);
        _record(sourceOwner, sourceRepoId, C, B);
        _record(sourceOwner, sourceRepoId, D, C);
    }

    function _record(RegistryActor actor, uint256 repoId, bytes20 commitHash, bytes20 parent) private {
        actor.recordCommit(
            repoId,
            MAIN,
            parent,
            commitHash,
            commitHash,
            _parents(parent),
            _digest(commitHash, 1),
            _digest(commitHash, 2)
        );
    }

    function _parents(bytes20 parent) private pure returns (bytes20[] memory parents) {
        if (parent == bytes20(0)) {
            return new bytes20[](0);
        }
        parents = new bytes20[](1);
        parents[0] = parent;
    }

    function _digest(bytes20 value, uint8 salt) private pure returns (bytes32) {
        return keccak256(abi.encodePacked(value, salt));
    }
}

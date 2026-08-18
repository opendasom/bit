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

    function createRepo(bytes calldata metadataCID) external returns (uint256) {
        return registry.createRepo(metadataCID);
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

    function forkRepo(uint256 sourceRepoId, bytes32 sourceBranch, bytes calldata metadataCID)
        external
        returns (uint256)
    {
        return registry.forkRepo(sourceRepoId, sourceBranch, metadataCID);
    }

    function setOwnRole(uint256 repoId, BitRegistryTypes.Role role) external {
        registry.setRole(repoId, address(this), role);
    }

    function setRole(uint256 repoId, address user, BitRegistryTypes.Role role) external {
        registry.setRole(repoId, user, role);
    }

    function createTag(uint256 repoId, bytes32 tag, bytes calldata target) external {
        registry.createTag(repoId, tag, target);
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
    bytes32 private constant FEATURE = keccak256("feature");

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
        require(pr.sourceStart == 2 && pr.sourceEnd == 4, "wrong indexed source range");
        require(keccak256(pr.description) == keccak256("adds feature X"), "wrong description");
        require(registry.getRepoPullRequestCount(targetRepoId) == 1, "wrong pr count");
        require(registry.getRepoPullRequestAt(targetRepoId, 0) == prId, "wrong pr index");
        require(registry.getSourceRepoPullRequestCount(sourceRepoId) == 1, "wrong source pr count");
        require(registry.getSourceRepoPullRequestAt(sourceRepoId, 0) == prId, "wrong source pr index");

        uint256[] memory targetIds = registry.getRepoPullRequestIds(targetRepoId, 0, 10);
        uint256[] memory sourceIds = registry.getSourceRepoPullRequestIds(sourceRepoId, 0, 10);
        require(targetIds.length == 1 && targetIds[0] == prId, "wrong target pr page");
        require(sourceIds.length == 1 && sourceIds[0] == prId, "wrong source pr page");
        require(registry.getRepoPullRequestIds(targetRepoId, 1, 10).length == 0, "target page should be empty");
        require(
            registry.getSourceRepoPullRequestIds(sourceRepoId, 0, 0).length == 0,
            "zero-sized source page should be empty"
        );
    }

    function testExistingCommitRejectsConflictingMetadata() public {
        uint256 repoId = targetOwner.createRepo();
        _record(targetOwner, repoId, A, bytes20(0));

        try targetOwner.recordCommit(
            repoId, FEATURE, bytes20(0), A, A, new bytes20[](0), _digest(A, 9), _digest(A, 2)
        ) {
            revert("expected metadata mismatch revert");
        } catch {}
        require(registry.getBranchHistoryLength(repoId, FEATURE) == 0, "conflicting commit was appended");
    }

    function testCannotAppendSameCommitTwiceToBranch() public {
        uint256 repoId = targetOwner.createRepo();
        _record(targetOwner, repoId, A, bytes20(0));

        try targetOwner.recordCommit(repoId, MAIN, A, A, A, _parents(A), _digest(A, 1), _digest(A, 2)) {
            revert("expected duplicate branch commit revert");
        } catch {}
        require(registry.getBranchHistoryLength(repoId, MAIN) == 1, "duplicate commit was appended");
    }

    function testForkCopiesBranchInSingleCall() public {
        uint256 sourceRepoId = sourceOwner.createRepo();
        _record(sourceOwner, sourceRepoId, A, bytes20(0));
        _record(sourceOwner, sourceRepoId, B, A);

        uint256 forkRepoId = stranger.forkRepo(sourceRepoId, MAIN, "fork-metadata");
        (address owner, bytes memory metadataCID) = registry.getRepo(forkRepoId);
        require(owner == address(stranger), "wrong fork owner");
        require(keccak256(metadataCID) == keccak256("fork-metadata"), "wrong fork metadata");
        require(registry.getRole(forkRepoId, address(stranger)) == BitRegistryTypes.Role.Owner, "wrong fork role");
        require(registry.getBranchHistoryLength(forkRepoId, MAIN) == 2, "wrong fork history length");
        require(registry.getBranchCommit(forkRepoId, MAIN) == B, "wrong fork head");
    }

    function testLastOwnerCannotRemoveOwnRole() public {
        uint256 repoId = targetOwner.createRepo();
        try targetOwner.setOwnRole(repoId, BitRegistryTypes.Role.None) {
            revert("expected last owner revert");
        } catch {}
        require(registry.getRole(repoId, address(targetOwner)) == BitRegistryTypes.Role.Owner, "owner was removed");
    }

    function testPullRequestAuthorMustControlSourceRepo() public {
        (uint256 targetRepoId, uint256 sourceRepoId) = _seedTargetAndSource();
        try stranger.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, "impersonated source") {
            revert("expected source role revert");
        } catch {}
        require(registry.getRepoPullRequestCount(targetRepoId) == 0, "unauthorized PR was created");
    }

    function testOwnerRoleCanTransferWithoutLockingRepository() public {
        uint256 repoId = targetOwner.createRepo();
        targetOwner.setRole(repoId, address(sourceOwner), BitRegistryTypes.Role.Owner);
        targetOwner.setOwnRole(repoId, BitRegistryTypes.Role.None);
        require(registry.getRole(repoId, address(targetOwner)) == BitRegistryTypes.Role.None, "old owner still active");
        require(registry.getRole(repoId, address(sourceOwner)) == BitRegistryTypes.Role.Owner, "new owner missing");
        try sourceOwner.setOwnRole(repoId, BitRegistryTypes.Role.Maintainer) {
            revert("expected final owner protection");
        } catch {}
    }

    function testRejectsMalformedCommitInputs() public {
        uint256 repoId = targetOwner.createRepo();
        bytes20[] memory noParents = new bytes20[](0);
        bytes20[] memory twoParents = new bytes20[](2);
        twoParents[0] = A;
        twoParents[1] = B;

        try targetOwner.recordCommit(repoId, bytes32(0), bytes20(0), A, A, noParents, _digest(A, 1), _digest(A, 2)) {
            revert("expected zero branch revert");
        } catch {}
        try targetOwner.recordCommit(repoId, MAIN, bytes20(0), A, bytes20(0), noParents, _digest(A, 1), _digest(A, 2)) {
            revert("expected zero tree revert");
        } catch {}
        try targetOwner.recordCommit(repoId, MAIN, bytes20(0), A, A, noParents, bytes32(0), _digest(A, 2)) {
            revert("expected zero digest revert");
        } catch {}
        try targetOwner.recordCommit(repoId, MAIN, bytes20(0), A, A, twoParents, _digest(A, 1), _digest(A, 2)) {
            revert("expected merge commit revert");
        } catch {}
        try targetOwner.recordCommit(repoId, MAIN, bytes20(0), A, A, _parents(B), _digest(A, 1), _digest(A, 2)) {
            revert("expected root parent revert");
        } catch {}
        require(registry.getBranchHistoryLength(repoId, MAIN) == 0, "malformed commit was appended");
    }

    function testBoundedForkAndPullRequestWork() public {
        uint256 targetRepoId = targetOwner.createRepo();
        uint256 sourceRepoId = sourceOwner.createRepo();
        bytes20 parent;
        for (uint256 i = 1; i <= registry.MAX_PR_COMMITS() + 1; i++) {
            // forge-lint: disable-next-line(unsafe-typecast)
            bytes20 commitHash = bytes20(uint160(i));
            _record(sourceOwner, sourceRepoId, commitHash, parent);
            parent = commitHash;
        }

        try sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, "too large") {
            revert("expected PR limit revert");
        } catch {}
        try stranger.forkRepo(sourceRepoId, MAIN, "too-large-fork") {
            revert("expected fork limit revert");
        } catch {}
    }

    function testForkAcceptsCommitLimit() public {
        uint256 sourceRepoId = sourceOwner.createRepo();
        bytes20 parent;
        for (uint256 i = 1; i <= registry.MAX_FORK_COMMITS(); i++) {
            // forge-lint: disable-next-line(unsafe-typecast)
            bytes20 commitHash = bytes20(uint160(i));
            _record(sourceOwner, sourceRepoId, commitHash, parent);
            parent = commitHash;
        }

        uint256 forkRepoId = stranger.forkRepo(sourceRepoId, MAIN, "at-limit");
        require(
            registry.getBranchHistoryLength(forkRepoId, MAIN) == registry.MAX_FORK_COMMITS(),
            "fork did not copy the bounded history"
        );
    }

    function testPullRequestAcceptsCommitLimit() public {
        uint256 targetRepoId = targetOwner.createRepo();
        uint256 sourceRepoId = sourceOwner.createRepo();
        bytes20 parent;
        for (uint256 i = 1; i <= registry.MAX_PR_COMMITS(); i++) {
            // forge-lint: disable-next-line(unsafe-typecast)
            bytes20 commitHash = bytes20(uint160(i));
            _record(sourceOwner, sourceRepoId, commitHash, parent);
            parent = commitHash;
        }

        uint256 prId = sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, MAIN, "at-limit");
        targetOwner.approvePullRequest(prId);
        require(
            registry.getBranchHistoryLength(targetRepoId, MAIN) == registry.MAX_PR_COMMITS(),
            "PR did not copy the bounded history"
        );
    }

    function testPullRequestRejectsZeroBranch() public {
        (uint256 targetRepoId, uint256 sourceRepoId) = _seedTargetAndSource();
        try sourceOwner.createPullRequest(targetRepoId, bytes32(0), sourceRepoId, MAIN, "zero target") {
            revert("expected zero target branch revert");
        } catch {}
        try sourceOwner.createPullRequest(targetRepoId, MAIN, sourceRepoId, bytes32(0), "zero source") {
            revert("expected zero source branch revert");
        } catch {}
    }

    function testRejectsEmptyForkAndOversizedMetadata() public {
        uint256 repoId = sourceOwner.createRepo();
        try stranger.forkRepo(repoId, MAIN, "fork") {
            revert("expected empty fork revert");
        } catch {}

        bytes memory oversized = new bytes(registry.MAX_REPO_METADATA_CID_LENGTH() + 1);
        try stranger.createRepo(oversized) {
            revert("expected metadata limit revert");
        } catch {}
    }

    function testCreatesAndReadsTag() public {
        uint256 repoId = targetOwner.createRepo();
        targetOwner.createTag(repoId, keccak256("v1.0.0"), abi.encodePacked(A));
        require(
            keccak256(registry.getTag(repoId, keccak256("v1.0.0"))) == keccak256(abi.encodePacked(A)),
            "wrong tag target"
        );
        try targetOwner.createTag(repoId, keccak256("v1.0.0"), abi.encodePacked(B)) {
            revert("expected duplicate tag revert");
        } catch {}
    }

    function testEnumeratesBranchesWithoutDuplicatesAndPaginates() public {
        uint256 repoId = targetOwner.createRepo();
        _record(targetOwner, repoId, A, bytes20(0));
        _record(targetOwner, repoId, B, A);
        _recordOnBranch(targetOwner, repoId, FEATURE, C, bytes20(0));
        _recordOnBranch(targetOwner, repoId, FEATURE, D, C);

        require(registry.getRepoBranchCount(repoId) == 2, "wrong branch count");
        (bytes32[] memory keys, bytes20[] memory heads, uint256[] memory lengths, bytes32[] memory digests) =
            registry.getRepoBranches(repoId, 0, 10);
        require(keys.length == 2, "wrong branch page length");
        require(keys[0] == MAIN && keys[1] == FEATURE, "wrong branch keys");
        require(heads[0] == B && heads[1] == D, "wrong branch heads");
        require(lengths[0] == 2 && lengths[1] == 2, "wrong branch history lengths");
        require(digests[0] == _digest(B, 1) && digests[1] == _digest(D, 1), "wrong head manifests");

        (bytes32[] memory pageKeys, bytes20[] memory pageHeads,,) = registry.getRepoBranches(repoId, 1, 1);
        require(pageKeys.length == 1 && pageKeys[0] == FEATURE, "wrong paginated branch key");
        require(pageHeads.length == 1 && pageHeads[0] == D, "wrong paginated branch head");
        (bytes32[] memory emptyKeys,,,) = registry.getRepoBranches(repoId, 2, 10);
        require(emptyKeys.length == 0, "branch page should be empty");
    }

    function testEnumeratesRepositoriesInPages() public {
        uint256 first = targetOwner.createRepo();
        uint256 second = sourceOwner.createRepo();
        (uint256[] memory ids, address[] memory owners, bytes[] memory metadataCIDs) = registry.getRepos(0, 1);
        require(ids.length == 1 && ids[0] == first, "wrong first repository page");
        require(owners[0] == address(targetOwner), "wrong first repository owner");
        require(metadataCIDs[0].length == 0, "wrong first repository metadata");
        (ids, owners, metadataCIDs) = registry.getRepos(1, 10);
        require(ids.length == 1 && ids[0] == second, "wrong second repository page");
        require(owners[0] == address(sourceOwner), "wrong second repository owner");
        require(metadataCIDs.length == 1, "wrong second metadata page");
        (ids,,) = registry.getRepos(2, 10);
        require(ids.length == 0, "repository page should be empty");
        require(registry.PROTOCOL_VERSION() == 2, "wrong protocol version");
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

    function testApprovePullRequestRegistersPreviouslyEmptyTargetBranch() public {
        uint256 targetRepoId = targetOwner.createRepo();
        uint256 sourceRepoId = sourceOwner.createRepo();
        _recordOnBranch(sourceOwner, sourceRepoId, FEATURE, C, bytes20(0));
        _recordOnBranch(sourceOwner, sourceRepoId, FEATURE, D, C);

        uint256 prId = sourceOwner.createPullRequest(targetRepoId, FEATURE, sourceRepoId, FEATURE, "new branch");
        targetOwner.approvePullRequest(prId);

        require(registry.getRepoBranchCount(targetRepoId) == 1, "target branch was not indexed");
        (bytes32[] memory keys, bytes20[] memory heads, uint256[] memory lengths,) =
            registry.getRepoBranches(targetRepoId, 0, 10);
        require(keys.length == 1 && keys[0] == FEATURE, "wrong target branch key");
        require(heads[0] == D, "wrong target branch head");
        require(lengths[0] == 2, "wrong target branch length");
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
        require(
            registry.getPullRequest(closedPrId).status == BitRegistryTypes.PullRequestStatus.Closed, "pr was not closed"
        );
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
        _recordOnBranch(actor, repoId, MAIN, commitHash, parent);
    }

    function _recordOnBranch(RegistryActor actor, uint256 repoId, bytes32 branch, bytes20 commitHash, bytes20 parent)
        private
    {
        actor.recordCommit(
            repoId,
            branch,
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

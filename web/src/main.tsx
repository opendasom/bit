import React, { useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  createPublicClient,
  createWalletClient,
  custom,
  hexToString,
  http,
  keccak256,
  stringToBytes,
  stringToHex,
  type Address,
  type Hex,
} from "viem";
import { foundry, sepolia } from "viem/chains";
import bitRegistryArtifact from "../../internal/chain/artifacts/BitRegistry.json";
import "./styles.css";

type RepoMetadata = {
  version?: number;
  name?: string;
  description?: string;
  defaultBranch?: string;
};

type RepoSummary = {
  id: bigint;
  owner: Address;
  metadataCID: string;
  metadata: RepoMetadata | null;
};

type BranchSummary = {
  name: string;
  branchHash: Hex;
  commitCount: number;
  headCommit: string;
};

type CommitSummary = {
  hash: string;
  treeHash: string;
  updater: Address;
  chainTimestamp: bigint;
  message: string;
  authorName: string;
  authorEmail: string;
  authorDate: string;
  committerName: string;
  committerEmail: string;
  committerDate: string;
  parents: string[];
};

type PullRequestSummary = {
  id: bigint;
  targetRepoId: bigint;
  targetBranch: Hex;
  sourceRepoId: bigint;
  sourceBranch: Hex;
  baseCommit: Hex;
  sourceHeadCommit: Hex;
  author: Address;
  status: bigint;
  createdAt: bigint;
  updatedAt: bigint;
  description: string;
};

type Manifest = {
  gitCommit: string;
  treeHash: string;
  branch?: string;
  parentCommits?: string[];
  author?: Identity;
  committer?: Identity;
  message?: string;
};

type Identity = {
  name?: string;
  email?: string;
  date?: string;
};

type EthereumProvider = {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
};

type PullRequestCreatedLog = {
  args: { prId?: bigint | null; sourceRepoId?: bigint | null };
};

type CommitRecordedLog = {
  args: { commitHash?: Hex; manifestDigest?: Hex };
};

type PageState = "home" | "project";

declare global {
  interface Window {
    ethereum?: EthereumProvider;
  }
}

const abi = bitRegistryArtifact.abi;
const configuredChain = Number(import.meta.env.VITE_BIT_CHAIN_ID ?? sepolia.id) === foundry.id ? foundry : sepolia;
const defaultRpcURL = import.meta.env.VITE_BIT_RPC_URL ?? readStoredValue("bit.rpcURL") ?? "https://ethereum-sepolia-rpc.publicnode.com";
const defaultContract = import.meta.env.VITE_BIT_CONTRACT ?? readStoredValue("bit.contract") ?? "0x34B9D83E03E2E7BF646E2452E0620E2F39cDbeE3";
const defaultGateway = import.meta.env.VITE_BIT_IPFS_GATEWAY ?? readStoredValue("bit.ipfsGateway") ?? "https://ipfs.sugang.click/ipfs";
const defaultIpfsAPI = "http://127.0.0.1:5001";
const APP_VERSION = "1.1.0";
const LOG_BLOCK_RANGE = 50_000n;
const ROLE_LABELS = ["None", "Contributor", "Maintainer", "Owner"] as const;
type RoleLabel = (typeof ROLE_LABELS)[number];
const PR_STATUS_LABELS = ["", "Open", "Approved", "Rejected", "Closed"] as const;

function App() {
  const initialRoute = routeFromLocation(window.location.pathname, window.location.search);
  const [page, setPage] = useState<PageState>(initialRoute.page);
  const [selectedRepoId, setSelectedRepoId] = useState<bigint | null>(initialRoute.repoId);
  const [selectedBranch, setSelectedBranch] = useState<string>(initialRoute.branch ?? "");
  const [selectedPrId, setSelectedPrId] = useState<bigint | null>(initialRoute.prId);
  const [activeTab, setActiveTab] = useState<"commits" | "prs">(initialRoute.prId ? "prs" : "commits");
  const [prFilter, setPrFilter] = useState<"open" | "all">("open");
  const [rpcURL, setRpcURL] = useState(defaultRpcURL);
  const [contractAddress, setContractAddress] = useState(defaultContract);
  const [ipfsGateway, setIpfsGateway] = useState(defaultGateway);
  const [repos, setRepos] = useState<RepoSummary[]>([]);
  const [branches, setBranches] = useState<BranchSummary[]>([]);
  const [commits, setCommits] = useState<CommitSummary[]>([]);
  const [pullRequests, setPullRequests] = useState<PullRequestSummary[]>([]);
  const [repoRole, setRepoRole] = useState<RoleLabel>("None");
  const [walletAddress, setWalletAddress] = useState<Address | null>(null);
  const [walletChainId, setWalletChainId] = useState<string>("");
  const [loadingRepos, setLoadingRepos] = useState(false);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [loadingAction, setLoadingAction] = useState<string | null>(null);
  const [copyState, setCopyState] = useState<"idle" | "copied">("idle");
  const [error, setError] = useState("");
  const [showCreatePr, setShowCreatePr] = useState(false);
  const [newPrTargetRepoId, setNewPrTargetRepoId] = useState("");
  const [newPrTargetBranch, setNewPrTargetBranch] = useState("");
  const [newPrSourceBranch, setNewPrSourceBranch] = useState("");
  const [newPrDescription, setNewPrDescription] = useState("");
  const [roleByTargetRepo, setRoleByTargetRepo] = useState<Map<string, RoleLabel>>(new Map());
  const [prDetailCommits, setPrDetailCommits] = useState<CommitSummary[]>([]);

  const publicClient = useMemo(() => createPublicClient({ transport: http(rpcURL) }), [rpcURL]);
  const selectedRepo = repos.find((repo) => repo.id === selectedRepoId) ?? null;
  const autoLoadedRouteRef = useRef<string | null>(null);
  const detailRepo =
    selectedRepo ?? (selectedRepoId ? { id: selectedRepoId, owner: "0x0000000000000000000000000000000000000000" as Address, metadataCID: "", metadata: null } : null);
  const repoNameById = useMemo(() => {
    const map = new Map<string, string>();
    for (const repo of repos) {
      map.set(repo.id.toString(), repo.metadata?.name || `Repo #${repo.id.toString()}`);
    }
    return map;
  }, [repos]);
  const branchNameByHash = useMemo(() => {
    const map = new Map<string, string>();
    for (const branch of branches) {
      map.set(branch.branchHash.toLowerCase(), branch.name);
    }
    return map;
  }, [branches]);

  useEffect(() => {
    const onPopState = () => {
      const nextRoute = routeFromLocation(window.location.pathname, window.location.search);
      setPage(nextRoute.page);
      setSelectedRepoId(nextRoute.repoId);
      setSelectedBranch(nextRoute.branch ?? "");
      setSelectedPrId(nextRoute.prId);
      setActiveTab(nextRoute.prId ? "prs" : "commits");
      setCopyState("idle");

      if (nextRoute.page === "home") {
        setCommits([]);
        setPullRequests([]);
        setBranches([]);
        setRepoRole("None");
        return;
      }

      if (nextRoute.repoId && repos.length > 0) {
        void loadRepoDetail(nextRoute.repoId, repos, nextRoute.branch ?? undefined);
      }
    };

    window.addEventListener("popstate", onPopState);
    return () => window.removeEventListener("popstate", onPopState);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [repos, rpcURL, contractAddress, ipfsGateway, walletAddress]);

  useEffect(() => {
    if (page === "home") {
      autoLoadedRouteRef.current = null;
      return;
    }
    if (!selectedRepoId) return;
    if (!/^0x[a-fA-F0-9]{40}$/.test(contractAddress)) return;
    const repo = repos.find((item) => item.id === selectedRepoId);
    const branch = selectedBranch || repo?.metadata?.defaultBranch || "main";
    const routeKey = `${contractAddress}:${selectedRepoId.toString()}:${branch}`;
    if (autoLoadedRouteRef.current === routeKey) return;
    autoLoadedRouteRef.current = routeKey;
    void loadRepoDetail(selectedRepoId, repos, branch);
  }, [page, repos, selectedRepoId, selectedBranch, contractAddress]);

  useEffect(() => {
    if (selectedPrId === null) {
      setPrDetailCommits([]);
      return;
    }
    const pr = pullRequests.find((item) => item.id === selectedPrId);
    if (!pr) return;
    void loadPullRequestDetail(pr);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedPrId, pullRequests]);

  async function loadRepos() {
    setLoadingRepos(true);
    setError("");
    try {
      const address = parseAddress(contractAddress);
      writeStoredValue("bit.rpcURL", rpcURL);
      writeStoredValue("bit.contract", contractAddress);
      writeStoredValue("bit.ipfsGateway", ipfsGateway);

      const count = (await publicClient.readContract({
        address,
        abi,
        functionName: "getRepoCount",
      })) as bigint;

      const nextRepos: RepoSummary[] = [];
      for (let repoId = 1n; repoId <= count; repoId++) {
        const [owner, metadataBytes] = (await publicClient.readContract({
          address,
          abi,
          functionName: "getRepo",
          args: [repoId],
        })) as [Address, Hex];
        const metadataCID = bytesHexToString(metadataBytes);
        const metadata = metadataCID ? await fetchJson<RepoMetadata>(ipfsURL(ipfsGateway, metadataCID)) : null;
        nextRepos.push({ id: repoId, owner, metadataCID, metadata });
      }

      setRepos(nextRepos);
      setPage("home");
      setSelectedRepoId(null);
      setSelectedBranch("");
      setSelectedPrId(null);
      setPrFilter("open");
      setActiveTab("commits");
      setCommits([]);
      setPullRequests([]);
      setBranches([]);
      setRepoRole("None");
      window.history.replaceState({}, "", "/");
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoadingRepos(false);
    }
  }

  async function openRepo(repoId: bigint) {
    const repo = repos.find((item) => item.id === repoId);
    const branch = repo?.metadata?.defaultBranch || "main";
    const nextPath = `/projects/${repoId.toString()}?branch=${encodeURIComponent(branch)}`;
    window.history.pushState({}, "", nextPath);
    setPage("project");
    setSelectedRepoId(repoId);
    setSelectedBranch(branch);
    setSelectedPrId(null);
    setPrFilter("open");
    setActiveTab("commits");
    setCopyState("idle");
    autoLoadedRouteRef.current = `${contractAddress}:${repoId.toString()}:${branch}`;
    await loadRepoDetail(repoId, repos, branch);
  }

  async function goHome() {
    window.history.pushState({}, "", "/");
    setPage("home");
    setSelectedRepoId(null);
    setSelectedBranch("");
    setSelectedPrId(null);
    setPrFilter("open");
    setActiveTab("commits");
    setCopyState("idle");
    setCommits([]);
    setPullRequests([]);
    setBranches([]);
    setRepoRole("None");
    setError("");
  }

  async function loadRepoDetail(repoId: bigint, repoLookup: RepoSummary[] = repos, branchName?: string) {
    setLoadingDetail(true);
    setError("");
    try {
      const address = parseAddress(contractAddress);
      const repo = repoLookup.find((item) => item.id === repoId) ?? selectedRepo;
      const branch = branchName || selectedBranch || repo?.metadata?.defaultBranch || "main";
      setSelectedBranch(branch);
      const branchKey = keccak256(stringToBytes(branch));

      const historyLength = (await publicClient.readContract({
        address,
        abi,
        functionName: "getBranchHistoryLength",
        args: [repoId, branchKey],
      })) as bigint;

      const pageSize = historyLength > 50n ? 50n : historyLength;
      const start = historyLength > pageSize ? historyLength - pageSize : 0n;
      const [hashes, treeHashes, manifestDigests] = (await publicClient.readContract({
        address,
        abi,
        functionName: "getBranchCommitsWithMetadata",
        args: [repoId, branchKey, start, pageSize],
      })) as [Hex[], Hex[], Hex[], Hex[]];

      const nextCommits = await Promise.all(
        hashes.map(async (hash, index) => {
          const [,,, updater, chainTimestamp] = (await publicClient.readContract({
            address,
            abi,
            functionName: "getCommit",
            args: [repoId, hash],
          })) as [Hex, Hex, Hex, Address, bigint];
          const manifestCID = cidV0FromDigest(manifestDigests[index]);
          const manifest = await fetchJson<Manifest>(ipfsURL(ipfsGateway, manifestCID));

          return {
            hash: bytes20HexToGitHash(hash),
            treeHash: bytes20HexToGitHash(treeHashes[index]),
            updater,
            chainTimestamp,
            message: manifest.message ?? "",
            authorName: manifest.author?.name ?? "",
            authorEmail: manifest.author?.email ?? "",
            authorDate: manifest.author?.date ?? "",
            committerName: manifest.committer?.name ?? "",
            committerEmail: manifest.committer?.email ?? "",
            committerDate: manifest.committer?.date ?? "",
            parents: manifest.parentCommits ?? [],
          };
        }),
      );

      setCommits(nextCommits.reverse());
      setRoleByTargetRepo(new Map());
      await loadRepoPullRequests(repoId);

      try {
        await loadRepoBranches(repoId, repoLookup, branch);
        if (walletAddress) {
          const role = (await publicClient.readContract({
            address,
            abi,
            functionName: "getRole",
            args: [repoId, walletAddress],
          })) as bigint;
          setRepoRole(roleToLabel(role));
        } else {
          setRepoRole("None");
        }
      } catch (err) {
        setError(errorMessage(err));
      }
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoadingDetail(false);
    }
  }

  async function loadRepoPullRequests(repoId: bigint, repoLookup: RepoSummary[] = repos) {
    try {
      const address = parseAddress(contractAddress);
      const prIds = new Set<string>();

      const prCount = (await publicClient.readContract({
        address,
        abi,
        functionName: "getRepoPullRequestCount",
        args: [repoId],
      })) as bigint;
      for (let index = 0n; index < prCount; index++) {
        const prId = (await publicClient.readContract({
          address,
          abi,
          functionName: "getRepoPullRequestAt",
          args: [repoId, index],
        })) as bigint;
        prIds.add(prId.toString());
      }

      const latestBlock = await publicClient.getBlockNumber({ cacheTime: 0 });
      for (let fromBlock = 0n; fromBlock <= latestBlock; fromBlock += LOG_BLOCK_RANGE) {
        const toBlock = fromBlock + LOG_BLOCK_RANGE - 1n > latestBlock ? latestBlock : fromBlock + LOG_BLOCK_RANGE - 1n;
        const logs = (await publicClient.getContractEvents({
          address,
          abi,
          eventName: "PullRequestCreated",
          args: { sourceRepoId: repoId },
          fromBlock,
          toBlock,
        })) as unknown as PullRequestCreatedLog[];
        for (const log of logs) {
          if (log.args.prId != null) prIds.add(log.args.prId.toString());
        }
      }

      const nextPullRequests: PullRequestSummary[] = [];
      for (const prId of prIds) {
        const pr = (await publicClient.readContract({
          address,
          abi,
          functionName: "getPullRequest",
          args: [BigInt(prId)],
        })) as PullRequestSummary;
        nextPullRequests.push({
          id: pr.id,
          targetRepoId: pr.targetRepoId,
          targetBranch: pr.targetBranch,
          sourceRepoId: pr.sourceRepoId,
          sourceBranch: pr.sourceBranch,
          baseCommit: pr.baseCommit,
          sourceHeadCommit: pr.sourceHeadCommit,
          author: pr.author,
          status: BigInt(pr.status as number | bigint),
          createdAt: pr.createdAt,
          updatedAt: pr.updatedAt,
          description: pr.description ? hexToString(pr.description as Hex) : "",
        });
      }
      nextPullRequests.sort((left, right) => (left.id < right.id ? 1 : left.id > right.id ? -1 : 0));
      setPullRequests(nextPullRequests);

      if (walletAddress) {
        const targetRepoIds = new Set(nextPullRequests.map((pr) => pr.targetRepoId.toString()));
        const roles = new Map<string, RoleLabel>();
        for (const targetRepoId of targetRepoIds) {
          const role = (await publicClient.readContract({
            address,
            abi,
            functionName: "getRole",
            args: [BigInt(targetRepoId), walletAddress],
          })) as bigint;
          roles.set(targetRepoId, roleToLabel(role));
        }
        setRoleByTargetRepo(roles);
      }
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  async function loadRepoBranches(repoId: bigint, repoLookup: RepoSummary[] = repos, defaultBranch = "main") {
    try {
      const repo = repoLookup.find((item) => item.id === repoId) ?? selectedRepo;
      const latestBlock = await publicClient.getBlockNumber({ cacheTime: 0 });
      const logs: CommitRecordedLog[] = [];
      for (let fromBlock = 0n; fromBlock <= latestBlock; fromBlock += LOG_BLOCK_RANGE) {
        const toBlock = fromBlock + LOG_BLOCK_RANGE - 1n > latestBlock ? latestBlock : fromBlock + LOG_BLOCK_RANGE - 1n;
        const page = (await publicClient.getContractEvents({
          address: parseAddress(contractAddress),
          abi,
          eventName: "CommitRecorded",
          args: { repoId },
          fromBlock,
          toBlock,
        })) as unknown as CommitRecordedLog[];
        logs.push(...page);
      }

      const nextBranches = new Map<string, BranchSummary>();
      for (const log of logs) {
        const manifestCID = cidV0FromDigest(log.args.manifestDigest!);
        const manifest = await getManifest(ipfsGateway, manifestCID);
        const branchName = manifest.branch || defaultBranch || repo?.metadata?.defaultBranch || "main";
        nextBranches.set(branchName, {
          name: branchName,
          branchHash: keccak256(stringToBytes(branchName)),
          commitCount: (nextBranches.get(branchName)?.commitCount ?? 0) + 1,
          headCommit: bytes20HexToGitHash(log.args.commitHash!),
        });
      }

      const sorted = Array.from(nextBranches.values()).sort((left, right) => {
        if (left.name === defaultBranch) return -1;
        if (right.name === defaultBranch) return 1;
        return left.name.localeCompare(right.name);
      });
      setBranches(sorted);
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  async function connectWallet() {
    try {
      if (!window.ethereum) {
        setError("MetaMask가 설치되어 있지 않습니다.");
        return;
      }
      const accounts = (await window.ethereum.request({ method: "eth_requestAccounts" })) as string[];
      const chainId = (await window.ethereum.request({ method: "eth_chainId" })) as string;
      if (!accounts?.[0]) {
        setError("지갑 계정을 가져오지 못했습니다.");
        return;
      }
      setWalletAddress(accounts[0] as Address);
      setWalletChainId(chainId);
      setError("");
      if (page === "project" && selectedRepoId) {
        void loadRepoDetail(selectedRepoId);
      }
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  async function runPullRequestAction(
    prId: bigint,
    functionName: "approvePullRequest" | "rejectPullRequest" | "closePullRequest",
  ) {
    if (!window.ethereum) {
      setError("MetaMask가 필요합니다.");
      return;
    }
    if (!walletAddress) {
      setError("먼저 MetaMask를 연결하세요.");
      return;
    }
    if (!selectedRepoId) {
      setError("선택된 프로젝트가 없습니다.");
      return;
    }

    setLoadingAction(`${functionName}-${prId.toString()}`);
    setError("");
    try {
      const chainId = (await window.ethereum.request({ method: "eth_chainId" })) as string;
      if (Number.parseInt(chainId, 16) !== configuredChain.id) {
        setError(`MetaMask network를 ${configuredChain.name}로 변경하세요.`);
        return;
      }
      const address = parseAddress(contractAddress);
      const walletClient = createWalletClient({
        account: walletAddress,
        chain: configuredChain,
        transport: custom(window.ethereum),
      });
      const txHash = await walletClient.writeContract({
        address,
        abi,
        functionName,
        args: [prId],
      });
      const receipt = await publicClient.waitForTransactionReceipt({ hash: txHash });
      if (receipt.status !== "success") {
        setError("트랜잭션이 revert 되었습니다. 대상 브랜치가 이동했거나 권한이 없는지 확인하세요.");
        return;
      }
      await loadRepoDetail(selectedRepoId);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoadingAction(null);
    }
  }

  async function createPullRequest() {
    if (!window.ethereum) {
      setError("MetaMask가 필요합니다.");
      return;
    }
    if (!walletAddress) {
      setError("먼저 MetaMask를 연결하세요.");
      return;
    }
    if (!selectedRepoId) {
      setError("선택된 프로젝트가 없습니다.");
      return;
    }
    const targetRepoId = parseOptionalBigInt(newPrTargetRepoId);
    if (targetRepoId === null || targetRepoId < 1n) {
      setError("대상 저장소를 선택하세요.");
      return;
    }
    const targetBranch = newPrTargetBranch.trim();
    const sourceBranch = newPrSourceBranch.trim();
    if (!targetBranch || !sourceBranch) {
      setError("대상 브랜치와 소스 브랜치를 입력하세요.");
      return;
    }
    if (newPrDescription.length > 2048) {
      setError("설명은 2048자 이하여야 합니다.");
      return;
    }

    setLoadingAction("create-pr");
    setError("");
    try {
      const chainId = (await window.ethereum.request({ method: "eth_chainId" })) as string;
      if (Number.parseInt(chainId, 16) !== configuredChain.id) {
        setError(`MetaMask network를 ${configuredChain.name}로 변경하세요.`);
        return;
      }
      const address = parseAddress(contractAddress);
      const walletClient = createWalletClient({
        account: walletAddress,
        chain: configuredChain,
        transport: custom(window.ethereum),
      });
      const txHash = await walletClient.writeContract({
        address,
        abi,
        functionName: "createPullRequest",
        args: [
          targetRepoId,
          keccak256(stringToBytes(targetBranch)),
          selectedRepoId,
          keccak256(stringToBytes(sourceBranch)),
          stringToHex(newPrDescription),
        ],
      });
      const receipt = await publicClient.waitForTransactionReceipt({ hash: txHash });
      if (receipt.status !== "success") {
        setError("PR 생성 트랜잭션이 revert 되었습니다. 대상 브랜치가 앞서 나갔거나 새 커밋이 없는지 확인하세요.");
        return;
      }
      setShowCreatePr(false);
      setNewPrDescription("");
      await loadRepoDetail(selectedRepoId);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoadingAction(null);
    }
  }

  function openPrDetail(pr: PullRequestSummary) {
    if (!selectedRepoId) return;
    const nextPath = `/projects/${selectedRepoId.toString()}/prs/${pr.id.toString()}?branch=${encodeURIComponent(selectedBranch)}`;
    window.history.pushState({}, "", nextPath);
    setSelectedPrId(pr.id);
    setActiveTab("prs");
  }

  function closePrDetail() {
    if (!selectedRepoId) return;
    const nextPath = `/projects/${selectedRepoId.toString()}?branch=${encodeURIComponent(selectedBranch)}`;
    window.history.pushState({}, "", nextPath);
    setSelectedPrId(null);
    setActiveTab("prs");
  }

  async function loadPullRequestDetail(pr: PullRequestSummary) {
    setLoadingDetail(true);
    setError("");
    try {
      const address = parseAddress(contractAddress);
      const historyLength = (await publicClient.readContract({
        address,
        abi,
        functionName: "getBranchHistoryLength",
        args: [pr.sourceRepoId, pr.sourceBranch],
      })) as bigint;
      if (historyLength === 0n) {
        setPrDetailCommits([]);
        return;
      }
      const [hashes, , manifestDigests] = (await publicClient.readContract({
        address,
        abi,
        functionName: "getBranchCommitsWithMetadata",
        args: [pr.sourceRepoId, pr.sourceBranch, 0n, historyLength],
      })) as [Hex[], Hex[], Hex[], Hex[]];

      const baseHex = pr.baseCommit.toLowerCase();
      let baseIndex = -1;
      for (let index = 0; index < hashes.length; index++) {
        if (hashes[index].toLowerCase() === baseHex) {
          baseIndex = index;
          break;
        }
      }
      const start = baseIndex + 1;
      if (start >= hashes.length) {
        setPrDetailCommits([]);
        return;
      }

      const nextCommits: CommitSummary[] = [];
      for (let index = start; index < hashes.length; index++) {
        const manifestCID = cidV0FromDigest(manifestDigests[index]);
        const manifest = await getManifest(ipfsGateway, manifestCID);
        nextCommits.push({
          hash: bytes20HexToGitHash(hashes[index]),
          treeHash: "",
          updater: "0x0000000000000000000000000000000000000000" as Address,
          chainTimestamp: 0n,
          message: manifest.message ?? "",
          authorName: manifest.author?.name ?? "",
          authorEmail: manifest.author?.email ?? "",
          authorDate: manifest.author?.date ?? "",
          committerName: manifest.committer?.name ?? "",
          committerEmail: manifest.committer?.email ?? "",
          committerDate: manifest.committer?.date ?? "",
          parents: manifest.parentCommits ?? [],
        });
      }
      setPrDetailCommits(nextCommits);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoadingDetail(false);
    }
  }

  async function copyForkCommand() {
    if (!detailRepo) return;

    const command = buildForkCommand({
      contractAddress,
      repoId: detailRepo.id,
      branch: selectedBranch || detailRepo.metadata?.defaultBranch || "main",
      rpcURL,
      ipfsAPI: defaultIpfsAPI,
    });

    try {
      await navigator.clipboard.writeText(command);
      setCopyState("copied");
      window.setTimeout(() => setCopyState("idle"), 1500);
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  function branchLabel(repoId: bigint, branchHash: Hex): string {
    const branchName = branchNameByHash.get(branchHash.toLowerCase());
    if (branchName) return branchName;
    return repoNameById.get(repoId.toString()) ?? `Repo #${repoId.toString()}`;
  }

  function canManagePullRequest(pr: PullRequestSummary): boolean {
    const role = roleByTargetRepo.get(pr.targetRepoId.toString()) ?? "None";
    return role === "Maintainer" || role === "Owner";
  }

  function canClosePullRequest(pr: PullRequestSummary): boolean {
    if (pr.author.toLowerCase() === walletAddress?.toLowerCase()) return true;
    return canManagePullRequest(pr);
  }

  const visiblePullRequests = useMemo(
    () => (prFilter === "open" ? pullRequests.filter((pr) => pr.status === 1n) : pullRequests),
    [prFilter, pullRequests],
  );

  const walletSummary = walletAddress ? `${shortAddress(walletAddress)} · ${formatChainId(walletChainId)}` : "";
  const contractSummary = contractAddress ? shortAddress(contractAddress) : "n/a";

  return (
    <main className="page">
      <header className="siteHeader">
        <a className="siteBrand" href="/" aria-label="Go to home">
          <div className="brandMark" aria-hidden="true">
            <span />
            <span />
            <span />
          </div>
          <div>
            <div className="brandName">BIT</div>
            <div className="brandTag">Blockchain-based version control</div>
          </div>
        </a>

        <div className="headerActions">
          <label className="headerField">
            <span>Contract</span>
            <input
              className="headerInput mono"
              value={contractAddress}
              onChange={(event) => setContractAddress(event.target.value)}
              placeholder="0x..."
            />
          </label>
          {walletAddress ? (
            <div className="headerChip headerWalletChip">
              <span>MetaMask</span>
              <strong className="mono">{walletSummary}</strong>
            </div>
          ) : (
            <button type="button" className="ghostButton headerButton" onClick={connectWallet}>
              Connect MetaMask
            </button>
          )}
        </div>
      </header>

      {error && <div className="errorBanner">{error}</div>}

      {page === "home" && (
        <>
          <section className="heroBand">
            <div className="heroCopy">
              <div className="heroKicker">01. Landing Page</div>
              <h1>BIT</h1>
              <p>Repository history and pull request metadata, recorded on-chain and surfaced read-only in the browser.</p>
              <div className="heroActions">
                <button type="button" className="primaryButton heroButton" onClick={loadRepos} disabled={loadingRepos}>
                  {loadingRepos ? "Loading..." : "Load repositories"}
                </button>
                <button
                  type="button"
                  className="ghostButton heroButton"
                  onClick={() => document.getElementById("projects-band")?.scrollIntoView({ behavior: "smooth", block: "start" })}
                >
                  View projects
                </button>
              </div>
            </div>

            <div className="heroVisual" aria-hidden="true">
              <div className="cube">
                <span />
                <span />
                <span />
              </div>
            </div>
          </section>

          <section className="projectsBand" id="projects-band">
            <div className="bandHeading">
              <div>
                <div className="eyebrow">Projects</div>
                <h2>Repository list</h2>
                <p>Each entry is a repository registered in the current BitRegistry contract.</p>
              </div>
              <div className="panelBadge">{repos.length} repositories</div>
            </div>

            <div className="projectList">
              {repos.length === 0 && <div className="emptyStage">Load repositories to see the list.</div>}
              {repos.map((repo) => (
                <button
                  key={repo.id.toString()}
                  type="button"
                  className="projectRow"
                  onClick={() => {
                    void openRepo(repo.id);
                  }}
                >
                  <div className="projectMain">
                    <div className="projectTitle">{repo.metadata?.name || `Repo #${repo.id}`}</div>
                    <div className="projectDescription">{repo.metadata?.description || "No description provided."}</div>
                  </div>
                  <div className="projectMeta">
                    <span className="mono">{shortAddress(repo.owner)}</span>
                    <span>{repo.metadata?.defaultBranch || "main"}</span>
                  </div>
                </button>
              ))}
            </div>
          </section>
        </>
      )}

      {page === "project" && detailRepo && (
        <section className="detailBand" id="detail-band">
          <header className="detailHeader">
            <div>
              <div className="eyebrow">Project</div>
              <h2>{detailRepo.metadata?.name || `Repo #${detailRepo.id}`}</h2>
              <p>{detailRepo.metadata?.description || "Metadata and pull request state for the selected repository."}</p>
            </div>
            <div className="detailHeaderActions">
              <button type="button" className="ghostButton copyButton" onClick={() => void copyForkCommand()} disabled={!detailRepo}>
                {copyState === "copied" ? "Copied fork command" : "Copy fork command"}
              </button>
            </div>
          </header>

          <div className="detailShell">
            <aside className="detailNav">
              <button type="button" className={activeTab === "commits" ? "tab active" : "tab"} onClick={() => setActiveTab("commits")}>
                Commits
              </button>
              <button type="button" className={activeTab === "prs" ? "tab active" : "tab"} onClick={() => setActiveTab("prs")}>
                Pull Requests
              </button>
            </aside>

            <div className="detailContent">
              {activeTab === "commits" && (
                <section className="detailGrid">
                  <div className="panel panelTall">
                    <div className="panelHeading">
                      <div>
                        <span className="eyebrow">Commit History</span>
                        <h3>Metadata only</h3>
                      </div>
                      <div className="panelBadge">{commits.length} commits</div>
                    </div>
                    <div className="timeline">
                      {loadingDetail && commits.length === 0 && <div className="emptyState">Loading commits...</div>}
                      {commits.map((commit) => (
                        <article className="timelineItem" key={commit.hash}>
                          <div className="timelineMark" />
                          <div className="timelineBody">
                            <div className="timelineTop">
                              <h4>{commit.message || "(no message)"}</h4>
                              <span className="mono commitHash">{shortHex(commit.hash)}</span>
                            </div>
                            <div className="timelineMeta">
                              <span>{commit.authorName || "Unknown author"}</span>
                              <span>{formatDate(commit.authorDate)}</span>
                              <span>{shortAddress(commit.updater)}</span>
                            </div>
                            <div className="timelineFoot">
                              <span className="mono">tree {shortHex(commit.treeHash)}</span>
                              <span className="mono">parents {commit.parents.length > 0 ? commit.parents.join(", ") : "none"}</span>
                            </div>
                          </div>
                        </article>
                      ))}
                    </div>
                  </div>

                  <aside className="panel stackPanel">
                    <div className="panelHeading">
                      <div>
                        <span className="eyebrow">Branches</span>
                        <h3>Branch list</h3>
                      </div>
                    </div>
                    <div className="branchList">
                      {branches.length === 0 && <div className="emptyState">No branches discovered yet.</div>}
                      {branches.map((branch) => (
                        <button
                          key={branch.branchHash}
                          type="button"
                          className={branch.name === selectedBranch ? "branchItem active" : "branchItem"}
                          onClick={() => {
                            const nextPath = `/projects/${detailRepo.id.toString()}?branch=${encodeURIComponent(branch.name)}`;
                            window.history.pushState({}, "", nextPath);
                            setSelectedBranch(branch.name);
                            setSelectedPrId(null);
                            setCopyState("idle");
                            autoLoadedRouteRef.current = `${contractAddress}:${detailRepo.id.toString()}:${branch.name}`;
                            void loadRepoDetail(detailRepo.id, repos, branch.name);
                          }}
                        >
                          <div className="branchItemTop">
                            <strong>{branch.name}</strong>
                            {branch.name === selectedBranch && <span className="statusChip ok">CURRENT</span>}
                          </div>
                          <div className="branchItemBottom">
                            <span className="mono">{branch.commitCount} commits</span>
                            <span className="mono">{shortHex(branch.headCommit)}</span>
                          </div>
                        </button>
                      ))}
                    </div>

                    <div className="panelHeading">
                      <div>
                        <span className="eyebrow">Repository</span>
                        <h3>{detailRepo.metadata?.name || `Repo #${detailRepo.id}`}</h3>
                      </div>
                    </div>
                    <div className="kvList">
                      <div>
                        <span>Owner</span>
                        <strong className="mono">{shortAddress(detailRepo.owner)}</strong>
                      </div>
                      <div>
                        <span>Default branch</span>
                        <strong>{selectedBranch || detailRepo.metadata?.defaultBranch || "main"}</strong>
                      </div>
                      <div>
                        <span>Metadata CID</span>
                        <strong className="mono">{detailRepo.metadataCID || "n/a"}</strong>
                      </div>
                    </div>
                  </aside>
                </section>
              )}

              {activeTab === "prs" && (
                <section className="detailGrid">
                  {selectedPrId !== null ? (
                    <PrDetailView
                      pr={pullRequests.find((item) => item.id === selectedPrId) ?? null}
                      commits={prDetailCommits}
                      loading={loadingDetail}
                      walletAddress={walletAddress}
                      loadingAction={loadingAction}
                      branchLabel={branchLabel}
                      canManage={canManagePullRequest}
                      canClose={canClosePullRequest}
                      onBack={() => closePrDetail()}
                      onAction={(pr, action) => void runPullRequestAction(pr.id, action)}
                    />
                  ) : (
                  <>
                  <div className="panel panelTall">
                    <div className="panelHeading">
                      <div>
                        <span className="eyebrow">Pull Requests</span>
                        <h3>PRs</h3>
                      </div>
                      <div className="panelHeadingActions">
                        <div className="chipGroup">
                          <button
                            type="button"
                            className={prFilter === "open" ? "filterChip active" : "filterChip"}
                            onClick={() => setPrFilter("open")}
                          >
                            Open
                          </button>
                          <button
                            type="button"
                            className={prFilter === "all" ? "filterChip active" : "filterChip"}
                            onClick={() => setPrFilter("all")}
                          >
                            All
                          </button>
                          <div className="panelBadge">
                            {prFilter === "open" ? pullRequests.filter((pr) => pr.status === 1n).length : pullRequests.length} shown
                          </div>
                        </div>
                        <button type="button" className="ghostButton" onClick={() => setShowCreatePr((value) => !value)}>
                          {showCreatePr ? "Cancel" : "New pull request"}
                        </button>
                      </div>
                    </div>

                    {showCreatePr && (
                      <div className="prForm">
                        <div className="prFormTitle">Create pull request</div>
                        <div className="prFormGrid">
                          <label className="prFormField">
                            <span>Source repo</span>
                            <input className="headerInput mono" value={detailRepo.metadata?.name || `Repo #${detailRepo.id.toString()}`} readOnly disabled />
                          </label>
                          <label className="prFormField">
                            <span>Source branch</span>
                            <select
                              className="headerInput mono"
                              value={newPrSourceBranch}
                              onChange={(event) => setNewPrSourceBranch(event.target.value)}
                            >
                              <option value="">Select branch...</option>
                              {branches.map((branch) => (
                                <option key={branch.branchHash} value={branch.name}>
                                  {branch.name}
                                </option>
                              ))}
                            </select>
                          </label>
                          <label className="prFormField">
                            <span>Target repo</span>
                            <select
                              className="headerInput mono"
                              value={newPrTargetRepoId}
                              onChange={(event) => {
                                setNewPrTargetRepoId(event.target.value);
                                if (event.target.value) {
                                  const target = repos.find((repo) => repo.id === BigInt(event.target.value));
                                  setNewPrTargetBranch(target?.metadata?.defaultBranch || "main");
                                }
                              }}
                            >
                              <option value="">Select repo...</option>
                              {repos.map((repo) => (
                                <option key={repo.id.toString()} value={repo.id.toString()}>
                                  {repo.metadata?.name || `Repo #${repo.id.toString()}`} (#{repo.id.toString()})
                                </option>
                              ))}
                            </select>
                          </label>
                          <label className="prFormField">
                            <span>Target branch</span>
                            <input
                              className="headerInput mono"
                              value={newPrTargetBranch}
                              onChange={(event) => setNewPrTargetBranch(event.target.value)}
                              placeholder="main"
                            />
                          </label>
                        </div>
                        <label className="prFormField">
                          <span>Description</span>
                          <textarea
                            className="headerInput prDescriptionInput"
                            value={newPrDescription}
                            onChange={(event) => setNewPrDescription(event.target.value)}
                            placeholder="What does this pull request change?"
                            rows={4}
                          />
                        </label>
                        <div className="prFormActions">
                          <button
                            type="button"
                            className="primaryButton"
                            disabled={!walletAddress || loadingAction === "create-pr"}
                            onClick={() => void createPullRequest()}
                          >
                            {loadingAction === "create-pr" ? "Creating..." : "Create pull request"}
                          </button>
                          {!walletAddress && <span className="helperText">Connect MetaMask to sign.</span>}
                        </div>
                      </div>
                    )}

                    <div className="prList">
                      {loadingDetail && visiblePullRequests.length === 0 && <div className="emptyState">Loading pull requests...</div>}
                      {visiblePullRequests.length === 0 && !loadingDetail && <div className="emptyState">No pull requests yet.</div>}
                      {visiblePullRequests.map((pr) => (
                        <article className="prCard" key={pr.id.toString()}>
                          <div className="prHeader">
                            <button type="button" className="prOpenButton" onClick={() => openPrDetail(pr)}>
                              <div className="prTitle">#{pr.id.toString()}</div>
                              <div className="prSubtitle">
                                <span className="mono">
                                  from {branchLabel(pr.sourceRepoId, pr.sourceBranch)}
                                </span>
                                <span className="mono">
                                  to {branchLabel(pr.targetRepoId, pr.targetBranch)}
                                </span>
                              </div>
                            </button>
                            <span className={`statusChip ${prStatusChipClass(pr.status)}`}>{prStatusLabel(pr.status)}</span>
                          </div>
                          {pr.description && <div className="prDescription">{pr.description}</div>}
                          <div className="prMetaGrid">
                            <div>
                              <span>Author</span>
                              <strong className="mono">{shortAddress(pr.author)}</strong>
                            </div>
                            <div>
                              <span>Created</span>
                              <strong>{formatUnix(pr.createdAt)}</strong>
                            </div>
                            <div>
                              <span>Base</span>
                              <strong className="mono">{shortHex(pr.baseCommit)}</strong>
                            </div>
                            <div>
                              <span>Head</span>
                              <strong className="mono">{shortHex(pr.sourceHeadCommit)}</strong>
                            </div>
                          </div>
                          {pr.status === 1n && (
                            <div className="prActions">
                              {canManagePullRequest(pr) && (
                                <button
                                  type="button"
                                  className="primaryButton"
                                  disabled={!walletAddress || loadingAction === `approvePullRequest-${pr.id.toString()}`}
                                  onClick={() => void runPullRequestAction(pr.id, "approvePullRequest")}
                                >
                                  {loadingAction === `approvePullRequest-${pr.id.toString()}` ? "Approving..." : "Approve"}
                                </button>
                              )}
                              {canManagePullRequest(pr) && (
                                <button
                                  type="button"
                                  className="dangerButton"
                                  disabled={!walletAddress || loadingAction === `rejectPullRequest-${pr.id.toString()}`}
                                  onClick={() => void runPullRequestAction(pr.id, "rejectPullRequest")}
                                >
                                  {loadingAction === `rejectPullRequest-${pr.id.toString()}` ? "Rejecting..." : "Reject"}
                                </button>
                              )}
                              {canClosePullRequest(pr) && (
                                <button
                                  type="button"
                                  className="ghostButton"
                                  disabled={!walletAddress || loadingAction === `closePullRequest-${pr.id.toString()}`}
                                  onClick={() => void runPullRequestAction(pr.id, "closePullRequest")}
                                >
                                  {loadingAction === `closePullRequest-${pr.id.toString()}` ? "Closing..." : "Close"}
                                </button>
                              )}
                              {!walletAddress && <span className="helperText">Connect MetaMask to sign.</span>}
                              {walletAddress && !canManagePullRequest(pr) && !canClosePullRequest(pr) && (
                                <span className="helperText">Owner or Maintainer of the target repo can manage this PR.</span>
                              )}
                            </div>
                          )}
                        </article>
                      ))}
                    </div>
                  </div>

                  <aside className="panel stackPanel">
                    <div className="panelHeading">
                      <div>
                        <span className="eyebrow">Authorization</span>
                        <h3>Signer</h3>
                      </div>
                    </div>
                    <div className="kvList">
                      <div>
                        <span>Wallet</span>
                        <strong>{walletAddress ? shortAddress(walletAddress) : "Not connected"}</strong>
                      </div>
                      <div>
                        <span>Role on repo</span>
                        <strong>{repoRole}</strong>
                      </div>
                      <div>
                        <span>Approval rule</span>
                        <strong>Owner or Maintainer</strong>
                      </div>
                      <div>
                        <span>Chain</span>
                        <strong>{formatChainId(walletChainId) || "unknown"}</strong>
                      </div>
                    </div>
                    <p className="helperText">Create, approve, reject and close open MetaMask and submit the transaction for the connected account.</p>
                  </aside>
                  </>
                )}
                </section>
              )}
            </div>
          </div>
        </section>
      )}

      <footer className="siteFooter">
        <span className="mono">v{APP_VERSION}</span>
      </footer>
    </main>
  );
}

function PrDetailView(props: {
  pr: PullRequestSummary | null;
  commits: CommitSummary[];
  loading: boolean;
  walletAddress: Address | null;
  loadingAction: string | null;
  branchLabel: (repoId: bigint, branchHash: Hex) => string;
  canManage: (pr: PullRequestSummary) => boolean;
  canClose: (pr: PullRequestSummary) => boolean;
  onBack: () => void;
  onAction: (pr: PullRequestSummary, action: "approvePullRequest" | "rejectPullRequest" | "closePullRequest") => void;
}) {
  const { pr, commits, loading, walletAddress, loadingAction, branchLabel, canManage, canClose, onBack, onAction } = props;
  if (!pr) {
    return (
      <div className="panel panelTall">
        <button type="button" className="ghostButton backButton" onClick={onBack}>
          &larr; All pull requests
        </button>
        <div className="emptyState">Pull request not found.</div>
      </div>
    );
  }

  const titleLine = pr.description.split("\n")[0].trim() || "(no description)";

  return (
    <div className="panel panelTall">
      <button type="button" className="ghostButton backButton" onClick={onBack}>
        &larr; All pull requests
      </button>

      <div className="prDetailHeader">
        <div>
          <div className="prDetailTitleRow">
            <span className={`statusChip ${prStatusChipClass(pr.status)}`}>{prStatusLabel(pr.status)}</span>
            <h3 className="prDetailTitle">#{pr.id.toString()} · {titleLine}</h3>
          </div>
          <div className="prDetailAuthor">
            <span className="mono">{shortAddress(pr.author)}</span>
            <span>opened {formatUnix(pr.createdAt)}</span>
            <span>updated {formatUnix(pr.updatedAt)}</span>
          </div>
        </div>
        {pr.status === 1n && (
          <div className="prActions">
            {canManage(pr) && (
              <button
                type="button"
                className="primaryButton"
                disabled={!walletAddress || loadingAction === `approvePullRequest-${pr.id.toString()}`}
                onClick={() => onAction(pr, "approvePullRequest")}
              >
                {loadingAction === `approvePullRequest-${pr.id.toString()}` ? "Approving..." : "Approve"}
              </button>
            )}
            {canManage(pr) && (
              <button
                type="button"
                className="dangerButton"
                disabled={!walletAddress || loadingAction === `rejectPullRequest-${pr.id.toString()}`}
                onClick={() => onAction(pr, "rejectPullRequest")}
              >
                {loadingAction === `rejectPullRequest-${pr.id.toString()}` ? "Rejecting..." : "Reject"}
              </button>
            )}
            {canClose(pr) && (
              <button
                type="button"
                className="ghostButton"
                disabled={!walletAddress || loadingAction === `closePullRequest-${pr.id.toString()}`}
                onClick={() => onAction(pr, "closePullRequest")}
              >
                {loadingAction === `closePullRequest-${pr.id.toString()}` ? "Closing..." : "Close"}
              </button>
            )}
            {!walletAddress && <span className="helperText">Connect MetaMask to sign.</span>}
          </div>
        )}
      </div>

      {pr.description && <div className="prDetailDescription">{pr.description}</div>}

      <div className="prMetaGrid prDetailMetaGrid">
        <div>
          <span>Source repo</span>
          <strong>{branchLabel(pr.sourceRepoId, pr.sourceBranch)}</strong>
        </div>
        <div>
          <span>Target repo</span>
          <strong>{branchLabel(pr.targetRepoId, pr.targetBranch)}</strong>
        </div>
        <div>
          <span>Base</span>
          <strong className="mono">{shortHex(pr.baseCommit)}</strong>
        </div>
        <div>
          <span>Head</span>
          <strong className="mono">{shortHex(pr.sourceHeadCommit)}</strong>
        </div>
      </div>

      <div className="panelHeading prDetailCommitsHeading">
        <div>
          <span className="eyebrow">Commits</span>
          <h3>Included changes</h3>
        </div>
        <div className="panelBadge">{commits.length} commits</div>
      </div>
      <div className="timeline">
        {loading && commits.length === 0 && <div className="emptyState">Loading commits...</div>}
        {!loading && commits.length === 0 && <div className="emptyState">No new commits in this pull request.</div>}
        {commits.map((commit) => (
          <article className="timelineItem" key={commit.hash}>
            <div className="timelineMark" />
            <div className="timelineBody">
              <div className="timelineTop">
                <h4>{commit.message || "(no message)"}</h4>
                <span className="mono commitHash">{shortHex(commit.hash)}</span>
              </div>
              <div className="timelineMeta">
                <span>{commit.authorName || "Unknown author"}</span>
                <span>{formatDate(commit.authorDate)}</span>
              </div>
            </div>
          </article>
        ))}
      </div>
    </div>
  );
}

function routeFromLocation(
  pathname: string,
  search: string,
): { page: PageState; repoId: bigint | null; branch: string | null; prId: bigint | null } {
  const prMatch = pathname.match(/^\/projects\/(\d+)\/prs\/(\d+)\/?$/);
  if (prMatch) {
    const params = new URLSearchParams(search);
    return { page: "project", repoId: BigInt(prMatch[1]), branch: params.get("branch"), prId: BigInt(prMatch[2]) };
  }
  const match = pathname.match(/^\/projects\/(\d+)\/?$/);
  if (!match) {
    return { page: "home", repoId: null, branch: null, prId: null };
  }
  const params = new URLSearchParams(search);
  return { page: "project", repoId: BigInt(match[1]), branch: params.get("branch"), prId: null };
}

function buildForkCommand(options: {
  contractAddress: string;
  repoId: bigint;
  branch: string;
  rpcURL: string;
  ipfsAPI: string;
}): string {
  const contract = options.contractAddress || "0xYourContractAddress";
  return [
    "bit fork",
    `bit://local/${contract}/${options.repoId.toString()}`,
    `--rpc ${options.rpcURL}`,
    `--contract ${contract}`,
    `--key <YOUR_PRIVATE_KEY>`,
    `--ipfs ${options.ipfsAPI}`,
    `--branch ${options.branch}`,
  ].join(" ");
}

function parseOptionalBigInt(value: string): bigint | null {
  if (!/^\d+$/.test(value.trim())) return null;
  try {
    return BigInt(value.trim());
  } catch {
    return null;
  }
}

function prStatusLabel(status: bigint): string {
  const label = PR_STATUS_LABELS[Number(status)];
  return label || "Unknown";
}

function prStatusChipClass(status: bigint): string {
  switch (Number(status)) {
    case 1:
      return "open";
    case 2:
      return "approved";
    case 3:
      return "rejected";
    case 4:
      return "closed";
    default:
      return "";
  }
}

function parseAddress(value: string): Address {
  const normalized = value.trim();
  if (!/^0x[a-fA-F0-9]{40}$/.test(normalized)) {
    throw new Error("Contract address must be a 20-byte hex address.");
  }
  return normalized as Address;
}

function bytesHexToString(value: Hex): string {
  const hex = value.startsWith("0x") ? value.slice(2) : value;
  let out = "";
  for (let index = 0; index < hex.length; index += 2) {
    const code = Number.parseInt(hex.slice(index, index + 2), 16);
    if (code !== 0) out += String.fromCharCode(code);
  }
  return out;
}

function bytes20HexToGitHash(value: Hex): string {
  return value.startsWith("0x") ? value.slice(2) : value;
}

function ipfsURL(gateway: string, cid: string): string {
  return `${gateway.replace(/\/$/, "")}/${cid}`;
}

async function fetchJson<T>(url: string): Promise<T> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`IPFS fetch failed (${response.status})`);
  }
  return response.json() as Promise<T>;
}

const manifestCache = new Map<string, Manifest>();

async function getManifest(gateway: string, cid: string): Promise<Manifest> {
  const cacheKey = `${gateway}|${cid}`;
  const cached = manifestCache.get(cacheKey);
  if (cached) return cached;
  const manifest = await fetchJson<Manifest>(ipfsURL(gateway, cid));
  manifestCache.set(cacheKey, manifest);
  return manifest;
}

function cidV0FromDigest(digestHex: Hex): string {
  const digest = hexToBytes(digestHex);
  return base58btcEncode(new Uint8Array([0x12, 0x20, ...digest]));
}

function hexToBytes(value: Hex): Uint8Array {
  const hex = value.startsWith("0x") ? value.slice(2) : value;
  const out = new Uint8Array(hex.length / 2);
  for (let index = 0; index < out.length; index += 1) {
    out[index] = Number.parseInt(hex.slice(index * 2, index * 2 + 2), 16);
  }
  return out;
}

function base58btcEncode(bytes: Uint8Array): string {
  const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
  let value = 0n;
  for (const byte of bytes) {
    value = (value << 8n) + BigInt(byte);
  }
  let encoded = "";
  while (value > 0n) {
    const mod = value % 58n;
    encoded = alphabet[Number(mod)] + encoded;
    value /= 58n;
  }
  for (const byte of bytes) {
    if (byte !== 0) break;
    encoded = alphabet[0] + encoded;
  }
  return encoded;
}

function formatDate(value: string): string {
  if (!value) return "unknown";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function formatUnix(value: bigint): string {
  return new Date(Number(value) * 1000).toLocaleString();
}

function formatChainId(chainId: string): string {
  if (!chainId) return "";
  try {
    return String(Number.parseInt(chainId, 16));
  } catch {
    return chainId;
  }
}

function shortAddress(value: string): string {
  if (!value) return "n/a";
  return `${value.slice(0, 6)}...${value.slice(-4)}`;
}

function shortHex(value: string): string {
  return `${value.slice(0, 8)}...${value.slice(-6)}`;
}

function roleToLabel(value: bigint): RoleLabel {
  const index = Number(value);
  return ROLE_LABELS[index] ?? "None";
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

function readStoredValue(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function writeStoredValue(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Ignore storage failures in private mode or restricted environments.
  }
}

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);

import React, { useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import { createPublicClient, createWalletClient, custom, hexToString, http, keccak256, stringToBytes, stringToHex, type Address, type Hex } from "viem";
import { foundry, sepolia } from "viem/chains";
import bitRegistryArtifact from "../../internal/chain/artifacts/BitRegistry.json";
import { ForkRepositoryPage } from "./components/ForkRepositoryPage";
import { PrDetailView } from "./components/PrDetailView";
import { RepositoryList } from "./components/RepositoryList";
import { APP_VERSION, MAX_REPOSITORY_NAME_LENGTH, WORKFLOW_STEPS, type RoleLabel } from "./constants";
import "./styles.css";
import type { BranchSummary, CommitSummary, Manifest, PageState, PullRequestSummary, RepoMetadata, RepoSummary } from "./types";
import {
  assertManifestMatchesRecord,
  bytes20HexToGitHash,
  bytesHexToString,
  cidV0FromDigest,
  errorMessage,
  fetchJson,
  formatChainId,
  formatDate,
  formatUnix,
  getManifest,
  ipfsURL,
  parseAddress,
  parseOptionalBigInt,
  prStatusChipClass,
  prStatusLabel,
  readStoredValue,
  repoIdFromCreateReceipt,
  roleToLabel,
  routeFromLocation,
  shortAddress,
  shortHex,
  uploadJsonToIPFS,
  writeStoredValue,
} from "./utils";

const abi = bitRegistryArtifact.abi;
const defaultIpfsAPI = import.meta.env.VITE_BIT_IPFS_API ?? "/ipfs-api";
const configuredChain = Number(import.meta.env.VITE_BIT_CHAIN_ID ?? sepolia.id) === foundry.id ? foundry : sepolia;
const defaultRpcURL = import.meta.env.VITE_BIT_RPC_URL ?? readStoredValue("bit.rpcURL") ?? "https://ethereum-sepolia-rpc.publicnode.com";
const defaultContract = import.meta.env.VITE_BIT_CONTRACT ?? readStoredValue("bit.contract") ?? "0x34B9D83E03E2E7BF646E2452E0620E2F39cDbeE3";
const defaultGateway = import.meta.env.VITE_BIT_IPFS_GATEWAY ?? readStoredValue("bit.ipfsGateway") ?? "https://ipfs.sugang.click/ipfs";

function providerErrorCode(err: unknown): number | null {
  if (typeof err !== "object" || err === null || !("code" in err)) return null;
  return typeof err.code === "number" ? err.code : null;
}

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
  const [workflowVisible, setWorkflowVisible] = useState(false);
  const [activeWorkflowStep, setActiveWorkflowStep] = useState(0);
  const [typedWorkflowCommand, setTypedWorkflowCommand] = useState("");
  const [error, setError] = useState("");
  const [showCreatePr, setShowCreatePr] = useState(false);
  const [newPrTargetRepoId, setNewPrTargetRepoId] = useState("");
  const [newPrTargetBranch, setNewPrTargetBranch] = useState("");
  const [newPrSourceBranch, setNewPrSourceBranch] = useState("");
  const [newPrDescription, setNewPrDescription] = useState("");
  const [roleAddress, setRoleAddress] = useState("");
  const [roleValue, setRoleValue] = useState("1");
  const [roleByTargetRepo, setRoleByTargetRepo] = useState<Map<string, RoleLabel>>(new Map());
  const [prDetailCommits, setPrDetailCommits] = useState<CommitSummary[]>([]);

  const publicClient = useMemo(() => createPublicClient({ transport: http(rpcURL) }), [rpcURL]);
  const selectedRepo = repos.find((repo) => repo.id === selectedRepoId) ?? null;
  const autoLoadedRouteRef = useRef<string | null>(null);
  const autoLoadedReposRef = useRef(false);
  const workflowRef = useRef<HTMLElement | null>(null);
  const detailRepo =
    selectedRepo ??
    (selectedRepoId
      ? {
          id: selectedRepoId,
          owner: "0x0000000000000000000000000000000000000000" as Address,
          metadataCID: "",
          metadata: null,
        }
      : null);
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

  function repoRouteKey(repoId: bigint, branch: string): string {
    return `${contractAddress}:${repoId.toString()}:${branch}:${walletAddress?.toLowerCase() ?? "anonymous"}`;
  }

  useEffect(() => {
    const onPopState = () => {
      const nextRoute = routeFromLocation(window.location.pathname, window.location.search);
      setPage(nextRoute.page);
      setSelectedRepoId(nextRoute.repoId);
      setSelectedBranch(nextRoute.branch ?? "");
      setSelectedPrId(nextRoute.prId);
      setActiveTab(nextRoute.prId ? "prs" : "commits");
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
    const provider = window.ethereum;
    if (!provider) return;
    const handleAccounts = (...args: unknown[]) => {
      const accounts = Array.isArray(args[0]) ? (args[0] as string[]) : [];
      setWalletAddress(accounts[0] ? (accounts[0] as Address) : null);
    };
    const handleChain = (...args: unknown[]) => {
      setWalletChainId(typeof args[0] === "string" ? args[0] : "");
    };
    provider.on?.("accountsChanged", handleAccounts);
    provider.on?.("chainChanged", handleChain);
    void Promise.all([provider.request({ method: "eth_accounts" }), provider.request({ method: "eth_chainId" })])
      .then(([accounts, chainId]) => {
        handleAccounts(accounts);
        handleChain(chainId);
      })
      .catch(() => undefined);
    return () => {
      provider.removeListener?.("accountsChanged", handleAccounts);
      provider.removeListener?.("chainChanged", handleChain);
    };
  }, []);

  useEffect(() => {
    if (page === "home") {
      autoLoadedRouteRef.current = null;
      return;
    }
    if (!selectedRepoId) return;
    if (!/^0x[a-fA-F0-9]{40}$/.test(contractAddress)) return;
    const repo = repos.find((item) => item.id === selectedRepoId);
    const branch = selectedBranch || repo?.metadata?.defaultBranch || "";
    const routeKey = repoRouteKey(selectedRepoId, branch || "__default__");
    if (autoLoadedRouteRef.current === routeKey) return;
    autoLoadedRouteRef.current = routeKey;
    void loadRepoDetail(selectedRepoId, repos, branch || undefined);
  }, [page, repos, selectedRepoId, selectedBranch, contractAddress, walletAddress]);

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

  useEffect(() => {
    const target = workflowRef.current;
    if (!target || workflowVisible) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setWorkflowVisible(true);
          observer.disconnect();
        }
      },
      { threshold: 0.28 },
    );
    observer.observe(target);
    return () => observer.disconnect();
  }, [workflowVisible]);

  useEffect(() => {
    if (!workflowVisible) return;
    const command = WORKFLOW_STEPS[activeWorkflowStep].command;
    if (typedWorkflowCommand.length < command.length) {
      const timer = window.setTimeout(() => {
        setTypedWorkflowCommand(command.slice(0, typedWorkflowCommand.length + 1));
      }, 22);
      return () => window.clearTimeout(timer);
    }

    const timer = window.setTimeout(() => {
      setActiveWorkflowStep((index) => (index + 1) % WORKFLOW_STEPS.length);
      setTypedWorkflowCommand("");
    }, 2400);
    return () => window.clearTimeout(timer);
  }, [activeWorkflowStep, typedWorkflowCommand, workflowVisible]);

  useEffect(() => {
    if (page !== "home" || autoLoadedReposRef.current) return;
    autoLoadedReposRef.current = true;
    void loadRepos();
    // Repository settings are deliberately refreshed through the Refresh button.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page]);

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
      const protocolVersion = (await publicClient.readContract({
        address,
        abi,
        functionName: "PROTOCOL_VERSION",
      })) as bigint;
      if (protocolVersion !== 2n) {
        throw new Error(`Unsupported BitRegistry protocol v${protocolVersion.toString()}; this client requires v2.`);
      }

      const nextRepos: RepoSummary[] = [];
      const batchSize = 32n;
      for (let start = 0n; start < count; start += batchSize) {
        const [repoIds, owners, metadataValues] = (await publicClient.readContract({
          address,
          abi,
          functionName: "getRepos",
          args: [start, batchSize],
        })) as [bigint[], Address[], Hex[]];
        const batch = await Promise.all(
          repoIds.map(async (repoId, index): Promise<RepoSummary> => {
            const owner = owners[index];
            const metadataBytes = metadataValues[index];
            const metadataCID = bytesHexToString(metadataBytes);
            try {
              const metadata = metadataCID ? await fetchJson<RepoMetadata>(ipfsURL(ipfsGateway, metadataCID)) : null;
              return { id: repoId, owner, metadataCID, metadata };
            } catch (err) {
              return {
                id: repoId,
                owner,
                metadataCID,
                metadata: null,
                metadataError: errorMessage(err),
              };
            }
          }),
        );
        nextRepos.push(...batch);
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
    autoLoadedRouteRef.current = repoRouteKey(repoId, branch);
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
      let repo = repoLookup.find((item) => item.id === repoId) ?? selectedRepo;
      if (!repo) {
        const [owner, metadataBytes] = (await publicClient.readContract({
          address,
          abi,
          functionName: "getRepo",
          args: [repoId],
        })) as [Address, Hex];
        const metadataCID = bytesHexToString(metadataBytes);
        let metadata: RepoMetadata | null = null;
        let metadataError: string | undefined;
        try {
          metadata = metadataCID ? await fetchJson<RepoMetadata>(ipfsURL(ipfsGateway, metadataCID)) : null;
        } catch (err) {
          metadataError = errorMessage(err);
        }
        repo = { id: repoId, owner, metadataCID, metadata, metadataError };
        setRepos((current) => (current.some((item) => item.id === repoId) ? current : [...current, repo!]));
      }
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
      const [hashes, treeHashes, manifestDigests, diffDigests] = (await publicClient.readContract({
        address,
        abi,
        functionName: "getBranchCommitsWithMetadata",
        args: [repoId, branchKey, start, pageSize],
      })) as [Hex[], Hex[], Hex[], Hex[]];

      const nextCommits = await Promise.all(
        hashes.map(async (hash, index) => {
          const [, , , updater, chainTimestamp] = (await publicClient.readContract({
            address,
            abi,
            functionName: "getCommit",
            args: [repoId, hash],
          })) as [Hex, Hex, Hex, Address, bigint];
          const manifestCID = cidV0FromDigest(manifestDigests[index]);
          let manifest: Manifest | null = null;
          let metadataError: string | undefined;
          try {
            manifest = await getManifest(ipfsGateway, manifestCID);
            assertManifestMatchesRecord(manifest, {
              gitCommit: bytes20HexToGitHash(hash),
              treeHash: bytes20HexToGitHash(treeHashes[index]),
              diffCID: cidV0FromDigest(diffDigests[index]),
              branchHash: branchKey,
            });
          } catch (err) {
            metadataError = errorMessage(err);
          }

          return {
            hash: bytes20HexToGitHash(hash),
            treeHash: bytes20HexToGitHash(treeHashes[index]),
            updater,
            chainTimestamp,
            message: manifest?.message ?? "",
            authorName: manifest?.author?.name ?? "",
            authorEmail: manifest?.author?.email ?? "",
            authorDate: manifest?.author?.date ?? "",
            committerName: manifest?.committer?.name ?? "",
            committerEmail: manifest?.committer?.email ?? "",
            committerDate: manifest?.committer?.date ?? "",
            parents: manifest?.parentCommits ?? [],
            metadataError,
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

  async function loadRepoPullRequests(repoId: bigint) {
    try {
      const address = parseAddress(contractAddress);
      const [targetCount, sourceCount] = (await Promise.all([
        publicClient.readContract({
          address,
          abi,
          functionName: "getRepoPullRequestCount",
          args: [repoId],
        }),
        publicClient.readContract({
          address,
          abi,
          functionName: "getSourceRepoPullRequestCount",
          args: [repoId],
        }),
      ])) as [bigint, bigint];

      async function readPullRequestIds(functionName: "getRepoPullRequestIds" | "getSourceRepoPullRequestIds", total: bigint): Promise<bigint[]> {
        const ids: bigint[] = [];
        const pageSize = 100n;
        for (let start = 0n; start < total; start += pageSize) {
          const page = (await publicClient.readContract({
            address,
            abi,
            functionName,
            args: [repoId, start, pageSize],
          })) as bigint[];
          ids.push(...page);
        }
        return ids;
      }

      const [targetIds, sourceIds] = await Promise.all([
        readPullRequestIds("getRepoPullRequestIds", targetCount),
        readPullRequestIds("getSourceRepoPullRequestIds", sourceCount),
      ]);
      const prIds = Array.from(new Set([...targetIds, ...sourceIds].map((prId) => prId.toString())), BigInt);

      const nextPullRequests = await Promise.all(
        prIds.map(async (prId): Promise<PullRequestSummary> => {
          const pr = (await publicClient.readContract({
            address,
            abi,
            functionName: "getPullRequest",
            args: [prId],
          })) as PullRequestSummary;
          return {
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
            sourceStart: pr.sourceStart,
            sourceEnd: pr.sourceEnd,
          };
        }),
      );
      nextPullRequests.sort((left, right) => (left.id < right.id ? 1 : left.id > right.id ? -1 : 0));
      setPullRequests(nextPullRequests);

      if (walletAddress) {
        const targetRepoIds = new Set(nextPullRequests.map((pr) => pr.targetRepoId.toString()));
        const roleEntries = await Promise.all(
          Array.from(targetRepoIds, async (targetRepoId): Promise<[string, RoleLabel]> => {
            const role = (await publicClient.readContract({
              address,
              abi,
              functionName: "getRole",
              args: [BigInt(targetRepoId), walletAddress],
            })) as bigint;
            return [targetRepoId, roleToLabel(role)];
          }),
        );
        const roles = new Map<string, RoleLabel>(roleEntries);
        setRoleByTargetRepo(roles);
      }
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  async function loadRepoBranches(repoId: bigint, repoLookup: RepoSummary[] = repos, defaultBranch = "main") {
    try {
      const address = parseAddress(contractAddress);
      const repo = repoLookup.find((item) => item.id === repoId) ?? selectedRepo;
      const branchCount = (await publicClient.readContract({
        address,
        abi,
        functionName: "getRepoBranchCount",
        args: [repoId],
      })) as bigint;

      const branchKeys: Hex[] = [];
      const headCommits: Hex[] = [];
      const historyLengths: bigint[] = [];
      const headManifestDigests: Hex[] = [];
      const pageSize = 100n;
      for (let start = 0n; start < branchCount; start += pageSize) {
        const [pageKeys, pageHeads, pageLengths, pageDigests] = (await publicClient.readContract({
          address,
          abi,
          functionName: "getRepoBranches",
          args: [repoId, start, pageSize],
        })) as [Hex[], Hex[], bigint[], Hex[]];
        branchKeys.push(...pageKeys);
        headCommits.push(...pageHeads);
        historyLengths.push(...pageLengths);
        headManifestDigests.push(...pageDigests);
      }

      const configuredDefaultBranch = defaultBranch || repo?.metadata?.defaultBranch || "main";
      const configuredDefaultKey = keccak256(stringToBytes(configuredDefaultBranch));
      const nextBranches = await Promise.all(
        branchKeys.map(async (branchKey, index): Promise<BranchSummary> => {
          const manifestCID = cidV0FromDigest(headManifestDigests[index]);
          let manifest: Manifest | null = null;
          try {
            manifest = await getManifest(ipfsGateway, manifestCID);
            if (keccak256(stringToBytes(manifest.branch)).toLowerCase() !== branchKey.toLowerCase()) {
              throw new Error(`Manifest branch does not match ${branchKey}`);
            }
          } catch {
            // A single unavailable manifest must not hide every other branch.
          }
          const isDefault = branchKey.toLowerCase() === configuredDefaultKey.toLowerCase();
          const branchName = manifest?.branch || (isDefault ? configuredDefaultBranch : `Unavailable ${shortHex(branchKey)}`);
          return {
            name: branchName,
            branchHash: branchKey,
            commitCount: Number(historyLengths[index]),
            headCommit: bytes20HexToGitHash(headCommits[index]),
            resolvedName: Boolean(manifest?.branch) || isDefault,
          };
        }),
      );

      const sorted = nextBranches.sort((left, right) => {
        if (left.name === configuredDefaultBranch) return -1;
        if (right.name === configuredDefaultBranch) return 1;
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
        setError("MetaMask is not installed.");
        return;
      }
      const accounts = (await window.ethereum.request({
        method: "eth_requestAccounts",
      })) as string[];
      const chainId = (await window.ethereum.request({
        method: "eth_chainId",
      })) as string;
      if (!accounts?.[0]) {
        setError("No wallet account was returned by MetaMask.");
        return;
      }
      setWalletAddress(accounts[0] as Address);
      setWalletChainId(chainId);
      setError("");
      await ensureWalletNetwork();
    } catch (err) {
      setError(errorMessage(err));
    }
  }

  async function ensureWalletNetwork(): Promise<boolean> {
    if (!window.ethereum) {
      setError("MetaMask is required.");
      return false;
    }
    const chainId = (await window.ethereum.request({
      method: "eth_chainId",
    })) as string;
    const expectedChainId = `0x${configuredChain.id.toString(16)}`;
    if (chainId.toLowerCase() === expectedChainId.toLowerCase()) {
      setWalletChainId(chainId);
      return true;
    }
    try {
      await window.ethereum.request({
        method: "wallet_switchEthereumChain",
        params: [{ chainId: expectedChainId }],
      });
    } catch (err) {
      if (providerErrorCode(err) !== 4902) {
        setError(`Could not switch MetaMask to ${configuredChain.name}: ${errorMessage(err)}`);
        return false;
      }
      try {
        await window.ethereum.request({
          method: "wallet_addEthereumChain",
          params: [
            {
              chainId: expectedChainId,
              chainName: configuredChain.name,
              nativeCurrency: configuredChain.nativeCurrency,
              rpcUrls: [rpcURL],
            },
          ],
        });
      } catch (addError) {
        setError(`Could not add ${configuredChain.name} to MetaMask: ${errorMessage(addError)}`);
        return false;
      }
    }
    setWalletChainId(expectedChainId);
    return true;
  }

  async function runPullRequestAction(prId: bigint, functionName: "approvePullRequest" | "rejectPullRequest" | "closePullRequest") {
    if (!window.ethereum) {
      setError("MetaMask is required.");
      return;
    }
    if (!walletAddress) {
      setError("Connect MetaMask first.");
      return;
    }
    if (!selectedRepoId) {
      setError("No repository is selected.");
      return;
    }

    setLoadingAction(`${functionName}-${prId.toString()}`);
    setError("");
    try {
      if (!(await ensureWalletNetwork())) return;
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
      const receipt = await publicClient.waitForTransactionReceipt({
        hash: txHash,
      });
      if (receipt.status !== "success") {
        setError("The transaction reverted. Check your role and whether the target branch moved.");
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
      setError("MetaMask is required.");
      return;
    }
    if (!walletAddress) {
      setError("Connect MetaMask first.");
      return;
    }
    if (!selectedRepoId) {
      setError("No repository is selected.");
      return;
    }
    const targetRepoId = parseOptionalBigInt(newPrTargetRepoId);
    if (targetRepoId === null || targetRepoId < 1n) {
      setError("Select a target repository.");
      return;
    }
    const targetBranch = newPrTargetBranch.trim();
    const sourceBranch = newPrSourceBranch.trim();
    if (!targetBranch || !sourceBranch) {
      setError("Enter both target and source branches.");
      return;
    }
    if (newPrDescription.length > 2048) {
      setError("The description must be 2,048 characters or fewer.");
      return;
    }

    setLoadingAction("create-pr");
    setError("");
    try {
      if (!(await ensureWalletNetwork())) return;
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
        args: [targetRepoId, keccak256(stringToBytes(targetBranch)), selectedRepoId, keccak256(stringToBytes(sourceBranch)), stringToHex(newPrDescription)],
      });
      const receipt = await publicClient.waitForTransactionReceipt({
        hash: txHash,
      });
      if (receipt.status !== "success") {
        setError("PR creation reverted. The target may have moved, the source may have no new commits, or the PR may exceed the commit limit.");
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

  async function updateRepositoryRole() {
    if (!window.ethereum || !walletAddress || !selectedRepoId) {
      setError("Connect MetaMask and open a repository first.");
      return;
    }
    if (repoRole !== "Owner") {
      setError("Only an Owner can update repository roles.");
      return;
    }
    setLoadingAction("set-role");
    setError("");
    try {
      const user = parseAddress(roleAddress);
      if (user.toLowerCase() === "0x0000000000000000000000000000000000000000") {
        throw new Error("The zero address cannot receive a role.");
      }
      if (!(await ensureWalletNetwork())) return;
      const walletClient = createWalletClient({
        account: walletAddress,
        chain: configuredChain,
        transport: custom(window.ethereum),
      });
      const txHash = await walletClient.writeContract({
        address: parseAddress(contractAddress),
        abi,
        functionName: "setRole",
        args: [selectedRepoId, user, Number(roleValue)],
      });
      const receipt = await publicClient.waitForTransactionReceipt({
        hash: txHash,
      });
      if (receipt.status !== "success") throw new Error("Role update reverted.");
      setRoleAddress("");
      await loadRepoDetail(selectedRepoId);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoadingAction(null);
    }
  }

  function openForkPage() {
    if (!selectedRepoId) return;
    const branch = selectedBranch || detailRepo?.metadata?.defaultBranch || "main";
    const nextPath = `/projects/${selectedRepoId.toString()}/fork?branch=${encodeURIComponent(branch)}`;
    window.history.pushState({}, "", nextPath);
    setPage("fork");
    setSelectedPrId(null);
    setError("");
  }

  function closeForkPage() {
    if (!selectedRepoId) return;
    const branch = selectedBranch || detailRepo?.metadata?.defaultBranch || "main";
    const nextPath = `/projects/${selectedRepoId.toString()}?branch=${encodeURIComponent(branch)}`;
    window.history.pushState({}, "", nextPath);
    setPage("project");
    setError("");
  }

  async function forkRepository(forkName: string) {
    if (!window.ethereum) {
      setError("MetaMask is required.");
      return;
    }
    if (!walletAddress) {
      setError("Connect MetaMask first.");
      return;
    }
    if (!selectedRepoId) {
      setError("No repository is selected.");
      return;
    }

    const branch = selectedBranch || detailRepo?.metadata?.defaultBranch || "main";
    const name = forkName.trim();
    if (!name) {
      setError("Fork repository name is required.");
      return;
    }
    if (name.length > MAX_REPOSITORY_NAME_LENGTH) {
      setError(`Fork repository name must be ${MAX_REPOSITORY_NAME_LENGTH} characters or fewer.`);
      return;
    }

    setLoadingAction("fork");
    setError("");

    try {
      if (!(await ensureWalletNetwork())) return;

      const address = parseAddress(contractAddress);
      const branchHash = keccak256(stringToBytes(branch));
      const historyLength = (await publicClient.readContract({
        address,
        abi,
        functionName: "getBranchHistoryLength",
        args: [selectedRepoId, branchHash],
      })) as bigint;
      if (historyLength === 0n) {
        setError(`Branch '${branch}' has no commits to fork.`);
        return;
      }
      const maxForkCommits = (await publicClient.readContract({
        address,
        abi,
        functionName: "MAX_FORK_COMMITS",
      })) as bigint;
      if (historyLength > maxForkCommits) {
        setError(`Fork is limited to ${maxForkCommits.toString()} commits; this branch has ${historyLength.toString()}.`);
        return;
      }

      const sourceMetadataCID =
        detailRepo?.metadataCID ||
        bytesHexToString(
          (
            (await publicClient.readContract({
              address,
              abi,
              functionName: "getRepo",
              args: [selectedRepoId],
            })) as [Address, Hex]
          )[1],
        );
      const sourceMetadata = detailRepo?.metadata ?? (sourceMetadataCID ? await fetchJson<RepoMetadata>(ipfsURL(ipfsGateway, sourceMetadataCID)) : null);
      const forkMetadata: RepoMetadata = {
        version: sourceMetadata?.version ?? 1,
        name,
        description: sourceMetadata?.description,
        defaultBranch: branch,
      };
      const forkMetadataCID = await uploadJsonToIPFS(defaultIpfsAPI, forkMetadata);

      const walletClient = createWalletClient({
        account: walletAddress,
        chain: configuredChain,
        transport: custom(window.ethereum),
      });
      const createTxHash = await walletClient.writeContract({
        address,
        abi,
        functionName: "forkRepo",
        args: [selectedRepoId, branchHash, stringToHex(forkMetadataCID)],
      });
      const createReceipt = await publicClient.waitForTransactionReceipt({
        hash: createTxHash,
      });
      if (createReceipt.status !== "success") {
        throw new Error("The atomic fork transaction reverted.");
      }
      const forkRepoId = repoIdFromCreateReceipt(createReceipt.logs);

      const forkRepo: RepoSummary = {
        id: forkRepoId,
        owner: walletAddress,
        metadataCID: forkMetadataCID,
        metadata: forkMetadata,
      };
      const nextRepos = [...repos, forkRepo];
      setRepos(nextRepos);
      const nextPath = `/projects/${forkRepoId.toString()}?branch=${encodeURIComponent(branch)}`;
      window.history.pushState({}, "", nextPath);
      setPage("project");
      setSelectedRepoId(forkRepoId);
      setSelectedBranch(branch);
      setSelectedPrId(null);
      setActiveTab("commits");
      autoLoadedRouteRef.current = repoRouteKey(forkRepoId, branch);
      await loadRepoDetail(forkRepoId, nextRepos, branch);
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
      const rangeLength = pr.sourceEnd - pr.sourceStart;
      if (rangeLength === 0n) {
        setPrDetailCommits([]);
        return;
      }
      const [hashes, treeHashes, manifestDigests, diffDigests] = (await publicClient.readContract({
        address,
        abi,
        functionName: "getBranchCommitsWithMetadata",
        args: [pr.sourceRepoId, pr.sourceBranch, pr.sourceStart, rangeLength],
      })) as [Hex[], Hex[], Hex[], Hex[]];

      const nextCommits = await Promise.all(
        hashes.map(async (hash, index): Promise<CommitSummary> => {
          const manifestCID = cidV0FromDigest(manifestDigests[index]);
          let manifest: Manifest | null = null;
          let metadataError: string | undefined;
          try {
            manifest = await getManifest(ipfsGateway, manifestCID);
            assertManifestMatchesRecord(manifest, {
              gitCommit: bytes20HexToGitHash(hash),
              treeHash: bytes20HexToGitHash(treeHashes[index]),
              diffCID: cidV0FromDigest(diffDigests[index]),
              branchHash: pr.sourceBranch,
            });
          } catch (err) {
            metadataError = errorMessage(err);
          }
          return {
            hash: bytes20HexToGitHash(hash),
            treeHash: bytes20HexToGitHash(treeHashes[index]),
            updater: "0x0000000000000000000000000000000000000000" as Address,
            chainTimestamp: 0n,
            message: manifest?.message ?? "",
            authorName: manifest?.author?.name ?? "",
            authorEmail: manifest?.author?.email ?? "",
            authorDate: manifest?.author?.date ?? "",
            committerName: manifest?.committer?.name ?? "",
            committerEmail: manifest?.committer?.email ?? "",
            committerDate: manifest?.committer?.date ?? "",
            parents: manifest?.parentCommits ?? [],
            metadataError,
          };
        }),
      );
      setPrDetailCommits(nextCommits);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setLoadingDetail(false);
    }
  }

  function branchLabel(repoId: bigint, branchHash: Hex): string {
    const repoName = repoNameById.get(repoId.toString()) ?? `Repo #${repoId.toString()}`;
    const branchName = branchNameByHash.get(branchHash.toLowerCase()) ?? shortHex(branchHash);
    return `${repoName}:${branchName}`;
  }

  function canManagePullRequest(pr: PullRequestSummary): boolean {
    const role = roleByTargetRepo.get(pr.targetRepoId.toString()) ?? "None";
    return role === "Maintainer" || role === "Owner";
  }

  function canClosePullRequest(pr: PullRequestSummary): boolean {
    if (pr.author.toLowerCase() === walletAddress?.toLowerCase()) return true;
    return canManagePullRequest(pr);
  }

  const visiblePullRequests = useMemo(() => (prFilter === "open" ? pullRequests.filter((pr) => pr.status === 1n) : pullRequests), [prFilter, pullRequests]);
  const canCreatePullRequest = repoRole === "Owner" || repoRole === "Maintainer";

  const walletSummary = walletAddress ? `${shortAddress(walletAddress)} · ${formatChainId(walletChainId)}` : "";
  const workflowStep = WORKFLOW_STEPS[activeWorkflowStep];

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
          <details className="connectionSettings">
            <summary className="ghostButton headerButton">Connection</summary>
            <div className="connectionPopover">
              <div className="connectionHeading">
                <strong>Data connection</strong>
                <span>{configuredChain.name}</span>
              </div>
              <label className="headerField">
                <span>RPC URL</span>
                <input className="headerInput mono" value={rpcURL} onChange={(event) => setRpcURL(event.target.value)} />
              </label>
              <label className="headerField">
                <span>Contract</span>
                <input className="headerInput mono" value={contractAddress} onChange={(event) => setContractAddress(event.target.value)} placeholder="0x..." />
              </label>
              <label className="headerField">
                <span>IPFS gateway</span>
                <input className="headerInput mono" value={ipfsGateway} onChange={(event) => setIpfsGateway(event.target.value)} />
              </label>
              <button type="button" className="primaryButton" onClick={() => void loadRepos()} disabled={loadingRepos}>
                {loadingRepos ? "Refreshing…" : "Apply and refresh"}
              </button>
            </div>
          </details>
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

      {error && (
        <div className="errorBanner" role="alert">
          <span>{error}</span>
          <button type="button" onClick={() => setError("")} aria-label="Dismiss error">
            ×
          </button>
        </div>
      )}

      {page === "home" && (
        <>
          <section className="heroBand">
            <div className="heroCopy">
              <div className="heroKicker">Open protocol for source history</div>
              <h1>
                Git history,
                <br />
                verified.
              </h1>
              <p>Bit records repository state on Ethereum and keeps commit data addressable through IPFS — without a central Git host.</p>
              <div className="heroSignals" aria-label="Bit protocol components">
                <span>Git-compatible</span>
                <span>IPFS-backed</span>
                <span>Ethereum-verified</span>
              </div>
              <div className="heroActions">
                <button
                  type="button"
                  className="primaryButton heroButton"
                  onClick={() => document.getElementById("projects-band")?.scrollIntoView({ behavior: "smooth", block: "start" })}
                >
                  Explore repositories
                </button>
                <button
                  type="button"
                  className="ghostButton heroButton"
                  onClick={() => document.getElementById("workflow")?.scrollIntoView({ behavior: "smooth", block: "start" })}
                >
                  Read the workflow
                </button>
              </div>
            </div>

            <div className="heroVisual" aria-label="Repository protocol preview">
              <div className="repoPreview">
                <div className="repoPreviewBar">
                  <div className="windowDots" aria-hidden="true">
                    <span />
                    <span />
                    <span />
                  </div>
                  <span className="mono">bit / protocol-overview.md</span>
                  <span className="previewBranch">
                    <i /> main
                  </span>
                </div>
                <div className="repoPreviewBody">
                  <div className="repoTree mono" aria-hidden="true">
                    <span className="treeRoot">bit/</span>
                    <span>├─ contracts/</span>
                    <span>│ └─ BitRegistry.sol</span>
                    <span>├─ internal/</span>
                    <span>│ └─ ipfs/</span>
                    <span>└─ web/</span>
                  </div>
                  <div className="repoCode mono">
                    <span>
                      <b>01</b>
                      <em>## source, without a host</em>
                    </span>
                    <span>
                      <b>02</b>commit.diff <strong>→</strong> <mark>IPFS</mark>
                    </span>
                    <span>
                      <b>03</b>branch.head <strong>→</strong> <mark>Ethereum</mark>
                    </span>
                    <span>
                      <b>04</b>pull request <strong>→</strong> <mark>MetaMask</mark>
                    </span>
                    <span>
                      <b>05</b>
                    </span>
                    <span>
                      <b>06</b>
                      <em># every ref is independently verifiable</em>
                    </span>
                  </div>
                </div>
                <div className="repoPreviewFooter">
                  <span>
                    <i /> synced to registry
                  </span>
                  <span className="mono">commit 72b24c6</span>
                </div>
              </div>
            </div>
          </section>

          <section className="workflowBand" id="workflow" ref={workflowRef}>
            <div className="workflowHeading">
              <div>
                <div className="eyebrow">Quick start</div>
                <h2>Publish verifiable history in four commands.</h2>
              </div>
              <p>Initialize a repository, point it to BitRegistry, then publish and restore the same commit history from any machine.</p>
            </div>

            <div className="workflowShell">
              <aside className="workflowSteps" aria-label="Bit command workflow">
                {WORKFLOW_STEPS.map((step, index) => (
                  <button
                    key={step.label}
                    type="button"
                    className={index === activeWorkflowStep ? "workflowStep active" : "workflowStep"}
                    onClick={() => {
                      setWorkflowVisible(true);
                      setActiveWorkflowStep(index);
                      setTypedWorkflowCommand("");
                    }}
                  >
                    <span>{String(index + 1).padStart(2, "0")}</span>
                    <strong>{step.label}</strong>
                  </button>
                ))}
              </aside>

              <div className="commandStudio">
                <div className="studioTabs">
                  <div className="studioTab">
                    <span className="fileDot" /> {workflowStep.fileName}
                  </div>
                  <div className="studioStatus">
                    <i /> live example
                  </div>
                </div>
                <div className="codeEditor mono">
                  <div className="editorLine">
                    <span>1</span>
                    <code># {workflowStep.label.toLowerCase()} a Bit repository</code>
                  </div>
                  <div className="editorLine">
                    <span>2</span>
                    <code>git status --short</code>
                  </div>
                  <div className="editorLine active">
                    <span>3</span>
                    <code>
                      <mark>$</mark> {typedWorkflowCommand}
                      <i className="typingCursor" />
                    </code>
                  </div>
                  <div className="editorLine">
                    <span>4</span>
                    <code className="comment"># provenance is stored, not hosted</code>
                  </div>
                </div>
                <div className="terminalOutput mono" aria-live="polite">
                  <div className="terminalTitle">
                    <span>terminal</span>
                    <span>zsh</span>
                  </div>
                  {typedWorkflowCommand.length === workflowStep.command.length ? (
                    workflowStep.output.map((line) => (
                      <div className="terminalLine" key={line}>
                        <i>›</i>
                        {line}
                      </div>
                    ))
                  ) : (
                    <div className="terminalLine muted">
                      <i>›</i>waiting for command…
                    </div>
                  )}
                </div>
              </div>
            </div>
          </section>

          <section className="protocolBand">
            <div className="protocolIntro">
              <div>
                <div className="eyebrow">Built for collaborative code</div>
                <h2>Clear ownership. Portable history.</h2>
              </div>
              <p>Bit keeps the Git workflow familiar while moving repository state and review authority into a public, verifiable protocol.</p>
            </div>

            <div className="benefitGrid">
              <article className="benefitCard">
                <span className="benefitIndex">01</span>
                <h3>History you can verify</h3>
                <p>Branch heads and commit digests are recorded on-chain, so a client can independently validate the history it receives.</p>
              </article>
              <article className="benefitCard">
                <span className="benefitIndex">02</span>
                <h3>Data without a host</h3>
                <p>Diffs and manifests are content-addressed on IPFS. A repository is not dependent on one central Git service.</p>
              </article>
              <article className="benefitCard">
                <span className="benefitIndex">03</span>
                <h3>Review at the protocol layer</h3>
                <p>Pull request state and fast-forward approval live in BitRegistry, with MetaMask signing every write.</p>
              </article>
            </div>

            <div className="rolesPanel">
              <div className="rolesHeading">
                <div className="eyebrow">Roles</div>
                <h3>Who can do what?</h3>
              </div>
              <div className="roleGrid">
                <article className="roleCard owner">
                  <span>Creator</span>
                  <strong>Govern access</strong>
                  <p>Assign and revoke repository roles.</p>
                </article>
                <article className="roleCard maintainer">
                  <span>Maintainer</span>
                  <strong>Publish and merge</strong>
                  <p>Record commits and approve or reject pull requests.</p>
                </article>
                <article className="roleCard contributor">
                  <span>Contributor</span>
                  <strong>Fork and propose</strong>
                  <p>Fork public history, push to a fork, and open a pull request for review.</p>
                </article>
              </div>
            </div>
          </section>

          <RepositoryList repos={repos} loading={loadingRepos} onRefresh={() => void loadRepos()} onOpen={openRepo} />
        </>
      )}

      {page === "fork" && detailRepo && (
        <ForkRepositoryPage
          sourceRepo={detailRepo}
          branch={selectedBranch || detailRepo.metadata?.defaultBranch || "main"}
          commitCount={branches.find((branch) => branch.name === (selectedBranch || detailRepo.metadata?.defaultBranch || "main"))?.commitCount ?? null}
          loading={loadingAction === "fork"}
          walletAddress={walletAddress ?? ""}
          onCancel={closeForkPage}
          onSubmit={(name) => void forkRepository(name)}
        />
      )}

      {page === "project" && detailRepo && (
        <section className="detailBand" id="detail-band">
          <header className="detailHeader">
            <div>
              <div className="eyebrow">Project</div>
              <h2>{detailRepo.metadata?.name || `Repo #${detailRepo.id}`}</h2>
              <p>{detailRepo.metadata?.description || "Metadata and pull request state for the selected repository."}</p>
            </div>
          </header>
          <div className="detailPrimaryAction">
            <button
              type="button"
              className="primaryButton forkButton"
              onClick={openForkPage}
              disabled={!walletAddress || loadingAction === "fork" || loadingDetail}
            >
              Fork {selectedBranch || detailRepo.metadata?.defaultBranch || "main"}
            </button>
          </div>

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
                              <h4 title={commit.metadataError}>{commit.metadataError ? "Commit metadata unavailable" : commit.message || "(no message)"}</h4>
                              <span className="mono commitHash">{shortHex(commit.hash)}</span>
                            </div>
                            <div className="timelineMeta">
                              <span>{commit.metadataError ? "IPFS unavailable" : commit.authorName || "Unknown author"}</span>
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
                          disabled={!branch.resolvedName}
                          title={branch.resolvedName ? undefined : "Branch name is unavailable because its IPFS manifest could not be loaded."}
                          onClick={() => {
                            const nextPath = `/projects/${detailRepo.id.toString()}?branch=${encodeURIComponent(branch.name)}`;
                            window.history.pushState({}, "", nextPath);
                            setSelectedBranch(branch.name);
                            setSelectedPrId(null);
                            autoLoadedRouteRef.current = repoRouteKey(detailRepo.id, branch.name);
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
                    {repoRole === "Owner" && (
                      <div className="roleManager">
                        <div>
                          <span className="eyebrow">Access</span>
                          <h3>Manage role</h3>
                        </div>
                        <label className="prFormField">
                          <span>Wallet address</span>
                          <input
                            className="headerInput mono"
                            value={roleAddress}
                            onChange={(event) => setRoleAddress(event.target.value)}
                            placeholder="0x..."
                          />
                        </label>
                        <label className="prFormField">
                          <span>Role</span>
                          <select className="headerInput" value={roleValue} onChange={(event) => setRoleValue(event.target.value)}>
                            <option value="0">None</option>
                            <option value="1">Contributor</option>
                            <option value="2">Maintainer</option>
                            <option value="3">Owner</option>
                          </select>
                        </label>
                        <button
                          type="button"
                          className="primaryButton"
                          disabled={!roleAddress.trim() || loadingAction === "set-role"}
                          onClick={() => void updateRepositoryRole()}
                        >
                          {loadingAction === "set-role" ? "Updating…" : "Update role"}
                        </button>
                        <p className="helperText">The contract prevents removal of the final Owner.</p>
                      </div>
                    )}
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
                              <button type="button" className={prFilter === "open" ? "filterChip active" : "filterChip"} onClick={() => setPrFilter("open")}>
                                Open
                              </button>
                              <button type="button" className={prFilter === "all" ? "filterChip active" : "filterChip"} onClick={() => setPrFilter("all")}>
                                All
                              </button>
                              <div className="panelBadge">
                                {prFilter === "open" ? pullRequests.filter((pr) => pr.status === 1n).length : pullRequests.length} shown
                              </div>
                            </div>
                            <button
                              type="button"
                              className="ghostButton"
                              disabled={!canCreatePullRequest}
                              title={canCreatePullRequest ? undefined : "You must maintain the source repository to open a PR."}
                              onClick={() => setShowCreatePr((value) => !value)}
                            >
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
                                <input
                                  className="headerInput mono"
                                  value={detailRepo.metadata?.name || `Repo #${detailRepo.id.toString()}`}
                                  readOnly
                                  disabled
                                />
                              </label>
                              <label className="prFormField">
                                <span>Source branch</span>
                                <select className="headerInput mono" value={newPrSourceBranch} onChange={(event) => setNewPrSourceBranch(event.target.value)}>
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
                                    <span className="mono">from {branchLabel(pr.sourceRepoId, pr.sourceBranch)}</span>
                                    <span className="mono">to {branchLabel(pr.targetRepoId, pr.targetBranch)}</span>
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

createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);

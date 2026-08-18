import type { Address, Hex } from "viem";

export type RepoMetadata = {
  version?: number;
  name?: string;
  description?: string;
  defaultBranch?: string;
};

export type RepoSummary = {
  id: bigint;
  owner: Address;
  metadataCID: string;
  metadata: RepoMetadata | null;
  metadataError?: string;
};

export type BranchSummary = {
  name: string;
  branchHash: Hex;
  commitCount: number;
  headCommit: string;
  resolvedName: boolean;
};

export type CommitSummary = {
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
  metadataError?: string;
};

export type PullRequestSummary = {
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
  sourceStart: bigint;
  sourceEnd: bigint;
};

export type WorkflowStep = {
  label: string;
  fileName: string;
  command: string;
  output: string[];
};

export type Manifest = {
  version: number;
  storage: string;
  diffAlgorithm: string;
  gitCommit: string;
  treeHash: string;
  branch: string;
  diffCID: string;
  parentCommits: string[];
  author: Identity;
  committer: Identity;
  message: string;
  bundleCid?: string;
};

export type Identity = {
  name: string;
  email: string;
  date: string;
};

export type EthereumProvider = {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
  on?: (event: "accountsChanged" | "chainChanged", listener: (...args: unknown[]) => void) => void;
  removeListener?: (event: "accountsChanged" | "chainChanged", listener: (...args: unknown[]) => void) => void;
};

export type PageState = "home" | "project" | "fork";

declare global {
  interface Window {
    ethereum?: EthereumProvider;
  }
}

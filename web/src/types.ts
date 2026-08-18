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
};

export type BranchSummary = {
  name: string;
  branchHash: Hex;
  commitCount: number;
  headCommit: string;
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
};

export type ForkCommitRecord = {
  commitHash: Hex;
  treeHash: Hex;
  manifestDigest: Hex;
  diffDigest: Hex;
};

export type WorkflowStep = {
  label: string;
  fileName: string;
  command: string;
  output: string[];
};

export type Manifest = {
  gitCommit: string;
  treeHash: string;
  branch?: string;
  parentCommits?: string[];
  author?: Identity;
  committer?: Identity;
  message?: string;
};

export type Identity = {
  name?: string;
  email?: string;
  date?: string;
};

export type EthereumProvider = {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
};

export type PageState = "home" | "project" | "fork";

declare global {
  interface Window {
    ethereum?: EthereumProvider;
  }
}

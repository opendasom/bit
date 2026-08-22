import { readFile } from "node:fs/promises";
import {
  createPublicClient,
  createWalletClient,
  http,
  keccak256,
  stringToBytes,
  stringToHex,
} from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { foundry } from "viem/chains";

const rpcURL = process.env.BIT_RPC_URL ?? "http://127.0.0.1:8545";
const contractAddress = process.env.BIT_REGISTRY ?? "0x5FbDB2315678afecb367f032d93F642f64180aa3";
const ipfsAPI = process.env.BIT_IPFS_API ?? "http://127.0.0.1:5001";

const privateKeys = [
  "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80",
  "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d",
  "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a",
  "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6",
];

const artifactURL = new URL("../../internal/chain/artifacts/BitRegistry.json", import.meta.url);
const artifact = JSON.parse(await readFile(artifactURL, "utf8"));
const abi = artifact.abi;
const publicClient = createPublicClient({ chain: foundry, transport: http(rpcURL) });
const accounts = privateKeys.map(privateKeyToAccount);
const wallets = accounts.map((account) =>
  createWalletClient({ account, chain: foundry, transport: http(rpcURL) }),
);

const zeroCommit = "0x0000000000000000000000000000000000000000";
const mainBranch = keccak256(stringToBytes("main"));
const docsBranch = keccak256(stringToBytes("docs"));

async function upload(name, value, contentType = "application/json") {
  const body = new FormData();
  const contents = typeof value === "string" ? value : JSON.stringify(value);
  body.append("file", new Blob([contents], { type: contentType }), name);
  const response = await fetch(`${ipfsAPI.replace(/\/$/, "")}/api/v0/add?pin=true&cid-version=0`, {
    method: "POST",
    body,
  });
  const text = await response.text();
  if (!response.ok) throw new Error(`IPFS add failed (${response.status}): ${text}`);
  const lastLine = text.trim().split("\n").filter(Boolean).at(-1);
  const result = lastLine ? JSON.parse(lastLine) : null;
  if (!result?.Hash) throw new Error("IPFS add response did not include a CID");
  return result.Hash;
}

function cidDigest(cid) {
  const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
  let value = 0n;
  for (const character of cid) {
    const digit = alphabet.indexOf(character);
    if (digit < 0) throw new Error(`Invalid base58 CID: ${cid}`);
    value = value * 58n + BigInt(digit);
  }
  let hex = value.toString(16);
  if (hex.length % 2 !== 0) hex = `0${hex}`;
  const leadingZeroes = cid.match(/^1*/)?.[0].length ?? 0;
  hex = `${"00".repeat(leadingZeroes)}${hex}`;
  if (!hex.startsWith("1220") || hex.length !== 68) throw new Error(`Expected a CIDv0 sha2-256 multihash: ${cid}`);
  return `0x${hex.slice(4)}`;
}

async function transact(wallet, functionName, args) {
  const hash = await wallet.writeContract({ address: contractAddress, abi, functionName, args });
  const receipt = await publicClient.waitForTransactionReceipt({ hash });
  if (receipt.status !== "success") throw new Error(`${functionName} reverted: ${hash}`);
  return receipt;
}

async function createRepo(wallet, metadata) {
  const metadataCID = await upload(`${metadata.name}.json`, metadata);
  await transact(wallet, "createRepo", [stringToHex(metadataCID)]);
  const repoId = await publicClient.readContract({ address: contractAddress, abi, functionName: "getRepoCount" });
  return { id: repoId, metadataCID, metadata };
}

const commitSpecs = {
  A: {
    hash: "0x1111111111111111111111111111111111111111",
    tree: "0x9111111111111111111111111111111111111111",
    parent: null,
    branch: "main",
    message: "chore: initialize the protocol workspace",
    author: "Mina Park",
  },
  B: {
    hash: "0x2222222222222222222222222222222222222222",
    tree: "0x9222222222222222222222222222222222222222",
    parent: "A",
    branch: "main",
    message: "feat: add repository and commit registries",
    author: "Mina Park",
  },
  C: {
    hash: "0x3333333333333333333333333333333333333333",
    tree: "0x9333333333333333333333333333333333333333",
    parent: "B",
    branch: "main",
    message: "feat: add pull request review workflow",
    author: "Alice Kim",
  },
  D: {
    hash: "0x4444444444444444444444444444444444444444",
    tree: "0x9444444444444444444444444444444444444444",
    parent: "C",
    branch: "main",
    message: "refactor: use indexed repository discovery",
    author: "Bob Lee",
  },
  E: {
    hash: "0x5555555555555555555555555555555555555555",
    tree: "0x9555555555555555555555555555555555555555",
    parent: "C",
    branch: "main",
    message: "feat: add compact activity summaries",
    author: "Carol Choi",
  },
  J: {
    hash: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    tree: "0x9aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    parent: null,
    branch: "docs",
    message: "docs: add local development guide",
    author: "Mina Park",
  },
  K: {
    hash: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    tree: "0x9bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    parent: "J",
    branch: "docs",
    message: "docs: explain fork and pull request flow",
    author: "Mina Park",
  },
};

async function uploadCommit(key, index) {
  const spec = commitSpecs[key];
  const parent = spec.parent ? commitSpecs[spec.parent] : null;
  const diffCID = await upload(
    `${key}.patch`,
    `diff --git a/${key.toLowerCase()}.txt b/${key.toLowerCase()}.txt\n+mock commit ${key}\n`,
    "text/plain",
  );
  const timestamp = new Date(Date.UTC(2026, 7, 18, 1 + index, 0, 0)).toISOString();
  const manifest = {
    version: 1,
    storage: "git-diff",
    diffAlgorithm: "git diff --binary --full-index",
    branch: spec.branch,
    diffCid: diffCID,
    gitCommit: spec.hash.slice(2),
    treeHash: spec.tree.slice(2),
    parentCommits: parent ? [parent.hash.slice(2)] : [],
    author: { name: spec.author, email: `${spec.author.toLowerCase().replaceAll(" ", ".")}@example.test`, date: timestamp },
    committer: { name: spec.author, email: `${spec.author.toLowerCase().replaceAll(" ", ".")}@example.test`, date: timestamp },
    message: spec.message,
  };
  const manifestCID = await upload(`${key}.manifest.json`, manifest);
  return { ...spec, manifestCID, manifestDigest: cidDigest(manifestCID), diffCID, diffDigest: cidDigest(diffCID) };
}

async function recordSeries(wallet, repoId, branch, keys, commits) {
  let expectedOldCommit = zeroCommit;
  for (const key of keys) {
    const commit = commits[key];
    const parents = commit.parent ? [commits[commit.parent].hash] : [];
    await transact(wallet, "recordCommit", [
      repoId,
      branch,
      expectedOldCommit,
      commit.hash,
      commit.tree,
      parents,
      commit.manifestDigest,
      commit.diffDigest,
    ]);
    expectedOldCommit = commit.hash;
  }
}

const code = await publicClient.getCode({ address: contractAddress });
if (!code || code === "0x") throw new Error(`No contract deployed at ${contractAddress}`);
const existingRepoCount = await publicClient.readContract({ address: contractAddress, abi, functionName: "getRepoCount" });
if (existingRepoCount !== 0n) throw new Error(`Seed requires an empty registry; found ${existingRepoCount} repositories`);

const commitEntries = await Promise.all(Object.keys(commitSpecs).map(async (key, index) => [key, await uploadCommit(key, index)]));
const commits = Object.fromEntries(commitEntries);

const repos = [];
repos.push(await createRepo(wallets[0], {
  version: 1,
  name: "bit-core",
  description: "Core protocol registry with indexed branches and pull requests.",
  defaultBranch: "main",
}));
repos.push(await createRepo(wallets[1], {
  version: 1,
  name: "bit-core-alice",
  description: "Alice's fork for the pull request workflow.",
  defaultBranch: "main",
}));
repos.push(await createRepo(wallets[2], {
  version: 1,
  name: "bit-core-bob",
  description: "Bob's renamed fork with an open architecture proposal.",
  defaultBranch: "main",
}));
repos.push(await createRepo(wallets[3], {
  version: 1,
  name: "bit-core-carol",
  description: "Carol's renamed fork used for review-state examples.",
  defaultBranch: "main",
}));

await recordSeries(wallets[0], repos[0].id, mainBranch, ["A", "B"], commits);
await recordSeries(wallets[0], repos[0].id, docsBranch, ["J", "K"], commits);
await recordSeries(wallets[1], repos[1].id, mainBranch, ["A", "B", "C"], commits);
await recordSeries(wallets[2], repos[2].id, mainBranch, ["A", "B", "C", "D"], commits);
await recordSeries(wallets[3], repos[3].id, mainBranch, ["A", "B", "C", "E"], commits);

for (let index = 1; index < accounts.length; index++) {
  await transact(wallets[0], "setRole", [repos[0].id, accounts[index].address, 1]);
}

await transact(wallets[1], "createPullRequest", [
  repos[0].id,
  mainBranch,
  repos[1].id,
  mainBranch,
  stringToHex("Add the first protocol-level pull request workflow."),
]);
await transact(wallets[0], "approvePullRequest", [1n]);

await transact(wallets[2], "createPullRequest", [
  repos[0].id,
  mainBranch,
  repos[2].id,
  mainBranch,
  stringToHex("Replace historical event scans with bounded indexed reads."),
]);

await transact(wallets[3], "createPullRequest", [
  repos[0].id,
  mainBranch,
  repos[3].id,
  mainBranch,
  stringToHex("Add an alternative activity summary to the explorer."),
]);
await transact(wallets[0], "rejectPullRequest", [3n]);

await transact(wallets[3], "createPullRequest", [
  repos[0].id,
  mainBranch,
  repos[3].id,
  mainBranch,
  stringToHex("Draft follow-up for the activity summary."),
]);
await transact(wallets[3], "closePullRequest", [4n]);

const summary = {
  rpcURL,
  contractAddress,
  accounts: accounts.map((account) => account.address),
  repositories: repos.map((repo) => ({ id: repo.id.toString(), name: repo.metadata.name, metadataCID: repo.metadataCID })),
  pullRequests: [
    { id: "1", status: "Approved" },
    { id: "2", status: "Open" },
    { id: "3", status: "Rejected" },
    { id: "4", status: "Closed" },
  ],
};

console.log(JSON.stringify(summary, null, 2));

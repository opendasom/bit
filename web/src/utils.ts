import { decodeEventLog, type Address, type Hex } from "viem";
import bitRegistryArtifact from "../../internal/chain/artifacts/BitRegistry.json";
import { PR_STATUS_LABELS, ROLE_LABELS, type RoleLabel } from "./constants";
import type { Manifest, PageState } from "./types";

const abi = bitRegistryArtifact.abi;
const manifestCache = new Map<string, Manifest>();

export function routeFromLocation(
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

export function repoIdFromCreateReceipt(logs: readonly { data: Hex; topics: readonly Hex[] }[]): bigint {
  for (const log of logs) {
    try {
      const decoded = decodeEventLog({ abi, data: log.data, topics: [...log.topics] as [Hex, ...Hex[]] });
      if (decoded.eventName === "RepoCreated") {
        const repoId = (decoded.args as { repoId?: bigint }).repoId;
        if (repoId !== undefined) return repoId;
      }
    } catch {
      // The receipt can contain unrelated logs. Only RepoCreated is relevant here.
    }
  }
  throw new Error("RepoCreated event was not found in the transaction receipt.");
}

export function parseOptionalBigInt(value: string): bigint | null {
  if (!/^\d+$/.test(value.trim())) return null;
  try {
    return BigInt(value.trim());
  } catch {
    return null;
  }
}

export function prStatusLabel(status: bigint): string {
  const label = PR_STATUS_LABELS[Number(status)];
  return label || "Unknown";
}

export function prStatusChipClass(status: bigint): string {
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

export function parseAddress(value: string): Address {
  const normalized = value.trim();
  if (!/^0x[a-fA-F0-9]{40}$/.test(normalized)) {
    throw new Error("Contract address must be a 20-byte hex address.");
  }
  return normalized as Address;
}

export function bytesHexToString(value: Hex): string {
  const hex = value.startsWith("0x") ? value.slice(2) : value;
  let out = "";
  for (let index = 0; index < hex.length; index += 2) {
    const code = Number.parseInt(hex.slice(index, index + 2), 16);
    if (code !== 0) out += String.fromCharCode(code);
  }
  return out;
}

export function bytes20HexToGitHash(value: Hex): string {
  return value.startsWith("0x") ? value.slice(2) : value;
}

export function ipfsURL(gateway: string, cid: string): string {
  return `${gateway.replace(/\/$/, "")}/${cid}`;
}

export async function fetchJson<T>(url: string): Promise<T> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`IPFS fetch failed (${response.status})`);
  }
  return response.json() as Promise<T>;
}

export async function getManifest(gateway: string, cid: string): Promise<Manifest> {
  const cacheKey = `${gateway}|${cid}`;
  const cached = manifestCache.get(cacheKey);
  if (cached) return cached;
  const manifest = await fetchJson<Manifest>(ipfsURL(gateway, cid));
  manifestCache.set(cacheKey, manifest);
  return manifest;
}

export function cidV0FromDigest(digestHex: Hex): string {
  const digest = hexToBytes(digestHex);
  return base58btcEncode(new Uint8Array([0x12, 0x20, ...digest]));
}

export function buildPagination(totalPages: number, currentPage: number): Array<number | string> {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, index) => index + 1);
  }

  const visiblePages = Array.from(
    new Set([1, currentPage - 1, currentPage, currentPage + 1, totalPages].filter((page) => page >= 1 && page <= totalPages)),
  ).sort((left, right) => left - right);

  return visiblePages.flatMap((page, index) => {
    const previous = visiblePages[index - 1];
    return index > 0 && page - previous > 1 ? [`ellipsis-${previous}`, page] : [page];
  });
}

export function formatDate(value: string): string {
  if (!value) return "unknown";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function formatUnix(value: bigint): string {
  return new Date(Number(value) * 1000).toLocaleString();
}

export function formatChainId(chainId: string): string {
  if (!chainId) return "";
  try {
    return String(Number.parseInt(chainId, 16));
  } catch {
    return chainId;
  }
}

export function shortAddress(value: string): string {
  if (!value) return "n/a";
  return `${value.slice(0, 6)}...${value.slice(-4)}`;
}

export function shortHex(value: string): string {
  return `${value.slice(0, 8)}...${value.slice(-6)}`;
}

export function roleToLabel(value: bigint): RoleLabel {
  const index = Number(value);
  return ROLE_LABELS[index] ?? "None";
}

export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

export function readStoredValue(key: string): string | null {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

export function writeStoredValue(key: string, value: string) {
  try {
    localStorage.setItem(key, value);
  } catch {
    // Ignore storage failures in private mode or restricted environments.
  }
}

function hexToBytes(value: Hex): Uint8Array {
  const hex = value.startsWith("0x") ? value.slice(2) : value;
  const out = new Uint8Array(hex.length / 2);
  for (let index = 0; index < hex.length; index += 1) {
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

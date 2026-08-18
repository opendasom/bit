import { afterEach, describe, expect, it, vi } from "vitest";
import {
  assertManifestMatchesRecord,
  buildPagination,
  bytesHexToString,
  cidV0FromDigest,
  fetchJson,
  formatChainId,
  parseAddress,
  parseOptionalBigInt,
  routeFromLocation,
} from "./utils";
import type { Manifest } from "./types";

describe("routeFromLocation", () => {
  it("parses repository, fork, and PR deep links", () => {
    expect(routeFromLocation("/projects/7", "?branch=dev")).toEqual({
      page: "project",
      repoId: 7n,
      branch: "dev",
      prId: null,
    });
    expect(routeFromLocation("/projects/7/fork", "?branch=main")).toEqual({
      page: "fork",
      repoId: 7n,
      branch: "main",
      prId: null,
    });
    expect(routeFromLocation("/projects/7/prs/3", "")).toEqual({
      page: "project",
      repoId: 7n,
      branch: null,
      prId: 3n,
    });
  });
});

describe("input parsing", () => {
  it("rejects malformed addresses and bigint values", () => {
    expect(() => parseAddress("0x1234")).toThrow(/20-byte/);
    expect(parseOptionalBigInt(" 42 ")).toBe(42n);
    expect(parseOptionalBigInt("-1")).toBeNull();
  });

  it("preserves malformed chain IDs instead of showing NaN", () => {
    expect(formatChainId("0xaa36a7")).toBe("11155111");
    expect(formatChainId("unknown")).toBe("unknown");
  });
});

describe("content helpers", () => {
  it("decodes byte strings and creates a CIDv0", () => {
    expect(bytesHexToString("0x626974" as `0x${string}`)).toBe("bit");
    const cid = cidV0FromDigest(`0x${"00".repeat(32)}` as `0x${string}`);
    expect(cid).toMatch(/^Qm[1-9A-HJ-NP-Za-km-z]{44}$/);
  });

  it("creates compact pagination", () => {
    expect(buildPagination(10, 5)).toEqual([1, "ellipsis-1", 4, 5, 6, "ellipsis-6", 10]);
  });

  it("rejects oversized IPFS JSON before reading the body", async () => {
    vi.stubGlobal("window", { setTimeout, clearTimeout });
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response("{}", {
            status: 200,
            headers: { "content-length": String(2 * 1024 * 1024 + 1) },
          }),
      ),
    );
    await expect(fetchJson("http://ipfs.test/object")).rejects.toThrow(/exceeds/);
  });

  it("rejects manifests that disagree with the on-chain record", () => {
    const manifest = {
      gitCommit: "11".repeat(20),
      treeHash: "22".repeat(20),
      diffCID: cidV0FromDigest(`0x${"33".repeat(32)}` as `0x${string}`),
      branch: "main",
    } as Manifest;
    const expected = {
      gitCommit: manifest.gitCommit,
      treeHash: manifest.treeHash,
      diffCID: manifest.diffCID,
      branchHash: `0x${"44".repeat(32)}` as `0x${string}`,
    };
    expect(() => assertManifestMatchesRecord(manifest, expected)).toThrow(/branch mismatch/);
  });
});

afterEach(() => {
  vi.unstubAllGlobals();
});

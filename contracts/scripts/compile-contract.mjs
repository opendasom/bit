import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";

const repositoryRoot = process.cwd();
const candidates = [process.env.FOUNDRY_FORGE, "forge", path.join(os.homedir(), ".foundry", "bin", "forge")].filter(Boolean);

let buildError = "forge executable was not found";
let built = false;
for (const executable of candidates) {
  const result = spawnSync(executable, ["build"], {
    cwd: repositoryRoot,
    encoding: "utf8",
  });
  if (result.status === 0) {
    built = true;
    break;
  }
  buildError = [result.error?.message, result.stdout, result.stderr].filter(Boolean).join("\n").trim();
}
if (!built) {
  throw new Error(`Foundry build failed. Install Foundry or set FOUNDRY_FORGE.\n${buildError}`);
}

const foundryPath = path.join(repositoryRoot, "out", "BitRegistry.sol", "BitRegistry.json");
const outputPath = path.join(repositoryRoot, "internal", "chain", "artifacts", "BitRegistry.json");
const foundryArtifact = JSON.parse(fs.readFileSync(foundryPath, "utf8"));
const abiTypeOrder = new Map([
  ["error", 0],
  ["event", 1],
  ["function", 2],
  ["constructor", 3],
]);
const abi = [...foundryArtifact.abi].sort((left, right) => {
  const typeOrder = (abiTypeOrder.get(left.type) ?? 99) - (abiTypeOrder.get(right.type) ?? 99);
  return typeOrder || (left.name ?? "").localeCompare(right.name ?? "");
});
function normalizeParameter(parameter) {
  const normalized = {};
  if (parameter.indexed !== undefined) normalized.indexed = parameter.indexed;
  if (parameter.components) normalized.components = parameter.components.map(normalizeParameter);
  if (parameter.internalType !== undefined) normalized.internalType = parameter.internalType;
  if (parameter.name !== undefined) normalized.name = parameter.name;
  if (parameter.type !== undefined) normalized.type = parameter.type;
  return normalized;
}

function normalizeABIEntry(entry) {
  if (entry.type === "error") {
    return { inputs: entry.inputs.map(normalizeParameter), name: entry.name, type: entry.type };
  }
  if (entry.type === "event") {
    return { anonymous: entry.anonymous, inputs: entry.inputs.map(normalizeParameter), name: entry.name, type: entry.type };
  }
  if (entry.type === "function") {
    return {
      inputs: entry.inputs.map(normalizeParameter),
      name: entry.name,
      outputs: entry.outputs.map(normalizeParameter),
      stateMutability: entry.stateMutability,
      type: entry.type,
    };
  }
  return entry;
}
const artifact = {
  contractName: "BitRegistry",
  abi: abi.map(normalizeABIEntry),
  bytecode: foundryArtifact.bytecode.object,
};

fs.mkdirSync(path.dirname(outputPath), { recursive: true });
fs.writeFileSync(outputPath, `${JSON.stringify(artifact, null, 2)}\n`);
console.log(outputPath);

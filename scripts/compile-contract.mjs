import fs from "node:fs";
import path from "node:path";
import solc from "solc";

const root = process.cwd();
const srcDir = path.join(root, "contracts/src");
const outDir = path.join(root, "internal/chain/artifacts");
const outPath = path.join(outDir, "BitRegistry.json");

const sources = Object.fromEntries(
  fs
    .readdirSync(srcDir)
    .filter((name) => name.endsWith(".sol"))
    .map((name) => [name, { content: fs.readFileSync(path.join(srcDir, name), "utf8") }])
);

const input = {
  language: "Solidity",
  sources,
  settings: {
    evmVersion: "paris",
    viaIR: true,
    optimizer: { enabled: true, runs: 200 },
    outputSelection: {
      "*": {
        "*": ["abi", "evm.bytecode.object"]
      }
    }
  }
};

const output = JSON.parse(solc.compile(JSON.stringify(input)));
const errors = output.errors ?? [];
const fatal = errors.filter((err) => err.severity === "error");
for (const err of errors) {
  console.error(err.formattedMessage);
}
if (fatal.length > 0) {
  process.exit(1);
}

const contract = output.contracts["BitRegistry.sol"].BitRegistry;
const artifact = {
  contractName: "BitRegistry",
  abi: contract.abi,
  bytecode: `0x${contract.evm.bytecode.object}`
};

fs.mkdirSync(outDir, { recursive: true });
fs.writeFileSync(outPath, `${JSON.stringify(artifact, null, 2)}\n`);
console.log(outPath);

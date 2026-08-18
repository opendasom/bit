import type { WorkflowStep } from "./types";

export const APP_VERSION = "1.1.0";
export const REPOSITORIES_PER_PAGE = 6;
export const MAX_REPOSITORY_NAME_LENGTH = 100;
export const ROLE_LABELS = ["None", "Contributor", "Maintainer", "Owner"] as const;
export type RoleLabel = (typeof ROLE_LABELS)[number];
export const PR_STATUS_LABELS = ["", "Open", "Approved", "Rejected", "Closed"] as const;

export const WORKFLOW_STEPS: WorkflowStep[] = [
  {
    label: "Initialize",
    fileName: "01-init.sh",
    command: "BIT_PRIVATE_KEY=$BIT_PRIVATE_KEY bit init --rpc $BIT_RPC_URL --contract $BIT_REGISTRY",
    output: ["uploading repository metadata to IPFS…", "repository created on-chain  ·  repoId: 1", "saved .bit/config.json"],
  },
  {
    label: "Connect",
    fileName: "02-remote.sh",
    command: "bit remote add origin bit://sepolia/$BIT_REGISTRY/1",
    output: ["origin registered", "network: sepolia", "branch: main"],
  },
  {
    label: "Publish",
    fileName: "03-push.sh",
    command: "bit push origin",
    output: ["packing commit diff…", "pinned manifest and diff to IPFS", "main → 72b24c6  ·  verified on-chain"],
  },
  {
    label: "Collaborate",
    fileName: "04-pull.sh",
    command: "bit pull origin main",
    output: ["resolving branch head from BitRegistry…", "restoring verified commits", "working tree is up to date"],
  },
];

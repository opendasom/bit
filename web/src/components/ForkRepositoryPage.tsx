import { useEffect, useState, type FormEvent } from "react";
import type { RepoSummary } from "../types";
import { shortAddress } from "../utils";

type ForkRepositoryPageProps = {
  sourceRepo: RepoSummary;
  branch: string;
  commitCount: number | null;
  loading: boolean;
  progress: { copied: number; total: number } | null;
  walletAddress: string;
  onCancel: () => void;
  onSubmit: (name: string) => void;
};

export function ForkRepositoryPage({
  sourceRepo,
  branch,
  commitCount,
  loading,
  progress,
  walletAddress,
  onCancel,
  onSubmit,
}: ForkRepositoryPageProps) {
  const sourceName = sourceRepo.metadata?.name || `Repo #${sourceRepo.id.toString()}`;
  const [name, setName] = useState(sourceName);
  const trimmedName = name.trim();

  useEffect(() => {
    setName(sourceName);
  }, [sourceName, sourceRepo.id]);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!trimmedName || loading) return;
    onSubmit(trimmedName);
  }

  const transactionCount = commitCount === null ? null : commitCount + 1;
  const actionLabel = progress
    ? `Copying commits ${progress.copied}/${progress.total}`
    : loading
      ? "Preparing fork…"
      : "Create fork";

  return (
    <section className="forkPage">
      <button type="button" className="ghostButton backButton" onClick={onCancel} disabled={loading}>
        ← Back to repository
      </button>

      <div className="forkPageHeader">
        <div>
          <div className="eyebrow">Fork repository</div>
          <h2>Create your copy</h2>
          <p>Choose a name for the fork before its metadata is pinned to your local IPFS node.</p>
        </div>
        <span className="panelBadge">Source #{sourceRepo.id.toString()}</span>
      </div>

      <form className="forkSetup" onSubmit={submit}>
        <div className="forkFormPanel">
          <label className="forkNameField">
            <span>Repository name</span>
            <input
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="Repository name"
              autoFocus
              disabled={loading}
              aria-describedby="fork-name-help"
            />
          </label>
          <p id="fork-name-help" className="helperText">
            The description is copied from the source. The selected branch becomes the fork's default branch.
          </p>

          <div className="forkSummaryGrid">
            <div>
              <span>Source</span>
              <strong>{sourceName}</strong>
            </div>
            <div>
              <span>Branch</span>
              <strong className="mono">{branch}</strong>
            </div>
            <div>
              <span>Commits</span>
              <strong>{commitCount ?? "Loading…"}</strong>
            </div>
            <div>
              <span>Owner</span>
              <strong className="mono">{shortAddress(walletAddress)}</strong>
            </div>
          </div>
        </div>

        <aside className="forkReviewPanel">
          <div>
            <div className="eyebrow">Review</div>
            <h3>{trimmedName || "Unnamed repository"}</h3>
            <p>
              Metadata will be uploaded once to local IPFS, then the repository and each commit will be written on-chain.
            </p>
          </div>

          <div className="forkTransactionNote">
            <span>Wallet confirmations</span>
            <strong>{transactionCount ?? "Calculating…"}</strong>
            <small>One repository transaction plus one transaction per commit.</small>
          </div>

          <div className="forkFormActions">
            <button type="button" className="ghostButton" onClick={onCancel} disabled={loading}>
              Cancel
            </button>
            <button type="submit" className="primaryButton" disabled={!trimmedName || loading || commitCount === null}>
              {actionLabel}
            </button>
          </div>
        </aside>
      </form>
    </section>
  );
}

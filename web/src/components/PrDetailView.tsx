import type { Address, Hex } from "viem";
import type { CommitSummary, PullRequestSummary } from "../types";
import { formatDate, formatUnix, prStatusChipClass, prStatusLabel, shortAddress, shortHex } from "../utils";

type PullRequestAction = "approvePullRequest" | "rejectPullRequest" | "closePullRequest";

type PrDetailViewProps = {
  pr: PullRequestSummary | null;
  commits: CommitSummary[];
  loading: boolean;
  walletAddress: Address | null;
  loadingAction: string | null;
  branchLabel: (repoId: bigint, branchHash: Hex) => string;
  canManage: (pr: PullRequestSummary) => boolean;
  canClose: (pr: PullRequestSummary) => boolean;
  onBack: () => void;
  onAction: (pr: PullRequestSummary, action: PullRequestAction) => void;
};

export function PrDetailView(props: PrDetailViewProps) {
  const { pr, commits, loading, walletAddress, loadingAction, branchLabel, canManage, canClose, onBack, onAction } = props;
  if (!pr) {
    return (
      <div className="panel panelTall">
        <button type="button" className="ghostButton backButton" onClick={onBack}>
          &larr; All pull requests
        </button>
        <div className="emptyState">Pull request not found.</div>
      </div>
    );
  }

  const titleLine = pr.description.split("\n")[0].trim() || "(no description)";

  return (
    <div className="panel panelTall">
      <button type="button" className="ghostButton backButton" onClick={onBack}>
        &larr; All pull requests
      </button>

      <div className="prDetailHeader">
        <div>
          <div className="prDetailTitleRow">
            <span className={`statusChip ${prStatusChipClass(pr.status)}`}>{prStatusLabel(pr.status)}</span>
            <h3 className="prDetailTitle">#{pr.id.toString()} · {titleLine}</h3>
          </div>
          <div className="prDetailAuthor">
            <span className="mono">{shortAddress(pr.author)}</span>
            <span>opened {formatUnix(pr.createdAt)}</span>
            <span>updated {formatUnix(pr.updatedAt)}</span>
          </div>
        </div>
        {pr.status === 1n && (
          <div className="prActions">
            {canManage(pr) && (
              <button
                type="button"
                className="primaryButton"
                disabled={!walletAddress || loadingAction === `approvePullRequest-${pr.id.toString()}`}
                onClick={() => onAction(pr, "approvePullRequest")}
              >
                {loadingAction === `approvePullRequest-${pr.id.toString()}` ? "Approving..." : "Approve"}
              </button>
            )}
            {canManage(pr) && (
              <button
                type="button"
                className="dangerButton"
                disabled={!walletAddress || loadingAction === `rejectPullRequest-${pr.id.toString()}`}
                onClick={() => onAction(pr, "rejectPullRequest")}
              >
                {loadingAction === `rejectPullRequest-${pr.id.toString()}` ? "Rejecting..." : "Reject"}
              </button>
            )}
            {canClose(pr) && (
              <button
                type="button"
                className="ghostButton"
                disabled={!walletAddress || loadingAction === `closePullRequest-${pr.id.toString()}`}
                onClick={() => onAction(pr, "closePullRequest")}
              >
                {loadingAction === `closePullRequest-${pr.id.toString()}` ? "Closing..." : "Close"}
              </button>
            )}
            {!walletAddress && <span className="helperText">Connect MetaMask to sign.</span>}
          </div>
        )}
      </div>

      {pr.description && <div className="prDetailDescription">{pr.description}</div>}

      <div className="prMetaGrid prDetailMetaGrid">
        <div>
          <span>Source repo</span>
          <strong>{branchLabel(pr.sourceRepoId, pr.sourceBranch)}</strong>
        </div>
        <div>
          <span>Target repo</span>
          <strong>{branchLabel(pr.targetRepoId, pr.targetBranch)}</strong>
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

      <div className="panelHeading prDetailCommitsHeading">
        <div>
          <span className="eyebrow">Commits</span>
          <h3>Included changes</h3>
        </div>
        <div className="panelBadge">{commits.length} commits</div>
      </div>
      <div className="timeline">
        {loading && commits.length === 0 && <div className="emptyState">Loading commits...</div>}
        {!loading && commits.length === 0 && <div className="emptyState">No new commits in this pull request.</div>}
        {commits.map((commit) => (
          <article className="timelineItem" key={commit.hash}>
            <div className="timelineMark" />
            <div className="timelineBody">
              <div className="timelineTop">
                <h4>{commit.message || "(no message)"}</h4>
                <span className="mono commitHash">{shortHex(commit.hash)}</span>
              </div>
              <div className="timelineMeta">
                <span>{commit.authorName || "Unknown author"}</span>
                <span>{formatDate(commit.authorDate)}</span>
              </div>
            </div>
          </article>
        ))}
      </div>
    </div>
  );
}

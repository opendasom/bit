import { useEffect, useMemo, useState } from "react";
import { REPOSITORIES_PER_PAGE } from "../constants";
import type { RepoSummary } from "../types";
import { buildPagination, shortAddress } from "../utils";

type RepositoryListProps = {
  repos: RepoSummary[];
  loading: boolean;
  onRefresh: () => void;
  onOpen: (repoId: bigint) => void;
};

export function RepositoryList({ repos, loading, onRefresh, onOpen }: RepositoryListProps) {
  const [page, setPage] = useState(1);
  const pageCount = Math.max(1, Math.ceil(repos.length / REPOSITORIES_PER_PAGE));
  const currentPage = Math.min(Math.max(page, 1), pageCount);
  const visibleRepos = useMemo(() => {
    const start = (currentPage - 1) * REPOSITORIES_PER_PAGE;
    return repos.slice(start, start + REPOSITORIES_PER_PAGE);
  }, [currentPage, repos]);
  const pagination = useMemo(() => buildPagination(pageCount, currentPage), [currentPage, pageCount]);

  useEffect(() => {
    setPage(1);
  }, [repos]);

  return (
    <section className="projectsBand" id="projects-band">
      <div className="bandHeading">
        <div>
          <div className="eyebrow">Projects</div>
          <h2>Repository list</h2>
          <p>Each entry is a repository registered in the current BitRegistry contract.</p>
        </div>
        <div className="projectsActions">
          <div className="panelBadge">{repos.length} repositories</div>
          <button
            type="button"
            className="ghostButton repositoryRefreshButton"
            onClick={onRefresh}
            disabled={loading}
            aria-label="Refresh repositories"
            title="Refresh repositories"
          >
            ↻
          </button>
        </div>
      </div>

      <div className="repositoryPanel">
        <div className="projectList">
          {repos.length === 0 && <div className="emptyStage">Load repositories to see the list.</div>}
          {visibleRepos.map((repo) => (
            <button key={repo.id.toString()} type="button" className="projectRow" onClick={() => onOpen(repo.id)}>
              <div className="projectMain">
                <div className="projectTitle">{repo.metadata?.name || `Repo #${repo.id}`}</div>
                <div className="projectDescription">
                  {repo.metadataError ? "Metadata is temporarily unavailable." : repo.metadata?.description || "No description provided."}
                </div>
              </div>
              <div className="projectMeta">
                <span className="mono">{shortAddress(repo.owner)}</span>
                <span>{repo.metadata?.defaultBranch || "main"}</span>
              </div>
            </button>
          ))}
        </div>
        {repos.length > 0 && (
          <nav className="repositoryPagination" aria-label="Repository pages">
            <button
              type="button"
              className="paginationButton"
              onClick={() => setPage(currentPage - 1)}
              disabled={currentPage === 1}
              aria-label="Previous repository page"
            >
              ←
            </button>
            {pagination.map((item) =>
              typeof item === "number" ? (
                <button
                  key={item}
                  type="button"
                  className={`paginationButton ${item === currentPage ? "active" : ""}`}
                  aria-current={item === currentPage ? "page" : undefined}
                  onClick={() => setPage(item)}
                >
                  {item}
                </button>
              ) : (
                <span key={item} className="paginationEllipsis" aria-hidden="true">
                  …
                </span>
              ),
            )}
            <button
              type="button"
              className="paginationButton"
              onClick={() => setPage(currentPage + 1)}
              disabled={currentPage === pageCount}
              aria-label="Next repository page"
            >
              →
            </button>
          </nav>
        )}
      </div>
    </section>
  );
}

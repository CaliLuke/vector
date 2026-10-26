/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment } from "react";
// components
import { cn } from "@plane/utils/common";
// hooks
import { useWorkspaceDraftList } from "@/hooks/use-workspace-drafts";
import { useWorkspaceIssueProperties } from "@/hooks/use-workspace-issue-properties";
// components
import { DraftIssueBlock } from "./draft-issue-block";
import { WorkspaceDraftEmptyState } from "./empty-state";
import { WorkspaceDraftIssuesLoader } from "./loader";

type TWorkspaceDraftIssuesRoot = {
  workspaceSlug: string;
};

export function WorkspaceDraftIssuesRoot(props: TWorkspaceDraftIssuesRoot) {
  const { workspaceSlug } = props;
  const query = useWorkspaceDraftList(workspaceSlug);
  const issues = query.issues;
  const paginationInfo = query.paginationInfo;
  // Query for fetching issue properties
  useWorkspaceIssueProperties(workspaceSlug);
  if (query.isPending) {
    return <WorkspaceDraftIssuesLoader items={14} />;
  }

  if (query.isError || query.accessDenied) {
    return (
      <div className="flex items-center gap-2 p-4 text-13 text-danger-primary">
        <span>{query.accessDenied ? "You do not have access to these drafts." : "Drafts could not be loaded."}</span>
        {!query.accessDenied && (
          <button type="button" className="underline" onClick={() => void query.retry()}>
            Retry
          </button>
        )}
      </div>
    );
  }

  if (issues.length === 0) return <WorkspaceDraftEmptyState />;

  return (
    <div className="relative">
      <div className="relative">
        {issues.map((issue) => (
          <DraftIssueBlock key={issue.id} workspaceSlug={workspaceSlug} issue={issue} />
        ))}
      </div>

      {paginationInfo?.next_page_results && (
        <Fragment>
          {query.isFetchingNextPage ? (
            <WorkspaceDraftIssuesLoader items={1} />
          ) : (
            <div
              className={cn("h-11 border-b border-subtle bg-surface-1 p-3 pl-6 text-13 font-medium transition-all", {
                "cursor-pointer text-accent-primary underline-offset-2 hover:text-accent-secondary hover:underline":
                  paginationInfo?.next_page_results,
              })}
              onClick={() => void query.fetchNextPage()}
            >
              Load More &darr;
            </div>
          )}
        </Fragment>
      )}
    </div>
  );
}

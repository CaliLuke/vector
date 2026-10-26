/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { CancelledError, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { TProjectIssuesSearchParams } from "@plane/types";
import { ProjectService } from "@/services/project";
import type { ProjectCatalogueScope, WorkspaceCatalogueScope } from "./catalogue-scope";
import { captureParentAccess, parentScopeKey, parentScopeReady } from "./cycle-module-scope";

const service = new ProjectService();

export function projectIssueSearchOptions(
  client: QueryClient,
  project: ProjectCatalogueScope,
  workspace: WorkspaceCatalogueScope,
  params: TProjectIssuesSearchParams,
  enabled = true
) {
  const queryKey = ["project-issue-search", parentScopeKey(project), parentScopeKey(workspace), params] as const;
  return queryOptions({
    queryKey,
    retry: false,
    queryFn:
      enabled && parentScopeReady(project) && parentScopeReady(workspace)
        ? async ({ signal }) => {
            const projectAccess = captureParentAccess(client, project);
            const workspaceAccess = captureParentAccess(client, workspace);
            const query = client.getQueryCache().find({ queryKey, exact: true });
            const isCurrent = () =>
              !signal.aborted &&
              projectAccess.isVerified() &&
              workspaceAccess.isVerified() &&
              client.getQueryCache().find({ queryKey, exact: true }) === query;
            if (!isCurrent() || !project.workspaceSlug || !project.projectId) throw new CancelledError();
            const rows = await service.projectIssuesSearch(project.workspaceSlug, project.projectId, params, signal);
            if (!isCurrent()) throw new CancelledError();
            if (
              rows.some(
                (row) =>
                  !row.id || !row.project_id || (!params.workspace_search && row.project_id !== project.projectId)
              )
            )
              throw new Error("Work item search returned an unexpected project.");
            return rows;
          }
        : skipToken,
  });
}

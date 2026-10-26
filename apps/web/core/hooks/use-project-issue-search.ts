/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { TProjectIssuesSearchParams } from "@plane/types";
import { useParentCatalogueScope } from "./use-parent-catalogue-scope";
import { projectIssueSearchOptions } from "@/queries/project-issue-search";

export function useProjectIssueSearch(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  params: TProjectIssuesSearchParams,
  enabled = true
) {
  const client = useQueryClient();
  const access = useParentCatalogueScope(workspaceSlug, projectId);
  const query = useQuery(
    projectIssueSearchOptions(client, access.projectScope, access.workspaceScope, params, enabled && access.isReady)
  );
  return {
    ...query,
    accessReady: enabled && access.isReady,
    data: enabled && access.isReady && query.isSuccess ? query.data : undefined,
    isError: access.isError || access.isAccessDenied || query.isError,
    error: access.isError || access.isAccessDenied ? new Error("Work item search access is unavailable.") : query.error,
    isPending: !access.isAccessDenied && (access.isPending || query.isPending),
    retry: !access.isReady ? access.retry : query.refetch,
  };
}

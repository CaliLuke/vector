/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useOptionalParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import type { TIssueDetailScope } from "@/queries/issue-details";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { currentIssueListMutationOwner, issueListMutationOptions } from "@/queries/issue-list-mutations";
import { captureProjectAccessScope } from "@/queries/project-access-scope";

/** Detail lifecycle writes use their item scope, including outside an archive list. */
export function useIssueDetailLifecycle(scope: TIssueDetailScope, issueId: string | undefined, ready: boolean) {
  const client = useQueryClient();
  const parent = useOptionalParentIssuePreferences()?.requestInput;
  const { mutateAsync } = useMutation(issueListMutationOptions(client));
  const mounted = useCommittedScope(JSON.stringify([scope, issueId, ready]));
  const run = useCallback(
    async (kind: "delete" | "archive" | "restore", workspaceSlug: string, projectId: string, targetIssueId: string) => {
      const access = captureProjectAccessScope(client, scope);
      const isCurrent = () => mounted() && access.isCurrent();
      if (
        !ready ||
        !isCurrent() ||
        scope.workspaceSlug !== workspaceSlug ||
        scope.projectId !== projectId ||
        issueId !== targetIssueId
      )
        throw new Error("The work item or account session has changed.");
      const detail = captureIssueDetailReconciliation(client, scope, [targetIssueId]);
      const statistics =
        parent?.workspaceSlug === workspaceSlug && parent.projectId === projectId && parent.isScopeCurrent()
          ? parent.parentStatistics
          : undefined;
      const owner = currentIssueListMutationOwner(
        client,
        workspaceSlug,
        projectId,
        scope.serviceType,
        isCurrent,
        statistics
      );
      await mutateAsync({ owner, mutation: { kind, issueId: targetIssueId } });
      if (kind === "delete") detail.remove();
      else void detail.refresh();
      return { isCurrent };
    },
    [client, scope, issueId, ready, mounted, mutateAsync, parent]
  );
  return useMemo(
    () => ({
      remove: (workspaceSlug: string, projectId: string, targetIssueId: string) =>
        run("delete", workspaceSlug, projectId, targetIssueId),
      archive: (workspaceSlug: string, projectId: string, targetIssueId: string) =>
        run("archive", workspaceSlug, projectId, targetIssueId),
      restore: (workspaceSlug: string, projectId: string, targetIssueId: string) =>
        run("restore", workspaceSlug, projectId, targetIssueId),
    }),
    [run]
  );
}

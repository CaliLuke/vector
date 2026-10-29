/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { TIssue } from "@plane/types";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import type { IssueActions } from "@/hooks/use-issues-actions";
import type { TIssueDetailScope } from "@/queries/issue-details";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { captureIssueActivityRefresh } from "@/queries/issue-discussion";
import { refreshItemSchema } from "@/queries/work-item-detail-schema";

export function useIssueDetailWrites(
  scope: TIssueDetailScope,
  issueId: string | undefined,
  enabled: boolean,
  actions: IssueActions
) {
  const client = useQueryClient();
  const isCurrent = useCommittedScope(JSON.stringify([scope, issueId, enabled]));
  const capture = useCallback(
    (workspaceSlug: string, projectId: string, targetIssueId: string) => {
      if (
        !enabled ||
        !isCurrent() ||
        workspaceSlug !== scope.workspaceSlug ||
        projectId !== scope.projectId ||
        targetIssueId !== issueId
      )
        throw new Error("The work item or account session has changed.");
      return {
        detail: captureIssueDetailReconciliation(client, scope, [targetIssueId]),
        refreshActivity: captureIssueActivityRefresh(client, {
          workspaceSlug,
          projectId,
          issueId: targetIssueId,
          serviceType: scope.serviceType,
        }),
      };
    },
    [client, scope, issueId, enabled, isCurrent]
  );

  const update = useCallback(
    async (workspaceSlug: string, projectId: string, targetIssueId: string, data: Partial<TIssue>) => {
      const owner = capture(workspaceSlug, projectId, targetIssueId);
      if (!actions.updateIssue) throw new Error("Work item editing is unavailable.");
      const receipt = await actions.updateIssue(projectId, targetIssueId, data);
      await owner.detail.reconcile(receipt);
      if (owner.detail.isCurrent())
        void Promise.allSettled([refreshItemSchema(client, workspaceSlug, targetIssueId), owner.refreshActivity()]);
    },
    [actions, capture, client]
  );

  const assign = useCallback(
    async (workspaceSlug: string, projectId: string, targetIssueId: string, write: () => Promise<void>) => {
      const owner = capture(workspaceSlug, projectId, targetIssueId);
      await write();
      void Promise.allSettled([owner.detail.refresh(), owner.refreshActivity()]);
    },
    [capture]
  );

  return useMemo(
    () => ({
      update,
      addCycleToIssue: (workspaceSlug: string, projectId: string, cycleId: string, targetIssueId: string) =>
        assign(workspaceSlug, projectId, targetIssueId, () =>
          actions.addCycleToIssue(projectId, cycleId, targetIssueId)
        ),
      removeIssueFromCycle: (workspaceSlug: string, projectId: string, cycleId: string, targetIssueId: string) =>
        assign(workspaceSlug, projectId, targetIssueId, () =>
          actions.removeCycleFromIssue(projectId, cycleId, targetIssueId)
        ),
      changeModulesInIssue: (
        workspaceSlug: string,
        projectId: string,
        targetIssueId: string,
        added: string[],
        removed: string[]
      ) =>
        assign(workspaceSlug, projectId, targetIssueId, () =>
          actions.changeModulesInIssue(projectId, targetIssueId, added, removed)
        ),
    }),
    [actions, assign, update]
  );
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { TIssue } from "@plane/types";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import type { IssueActions } from "@/hooks/use-issues-actions";
import type { TIssueDetailScope } from "@/queries/issue-details";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { captureIssueActivityRefresh } from "@/queries/issue-discussion";
import { refreshItemSchema } from "@/queries/work-item-detail-schema";

export function useIssueDetailUpdate(
  scope: TIssueDetailScope,
  issueId: string | undefined,
  enabled: boolean,
  updateIssue: IssueActions["updateIssue"]
) {
  const client = useQueryClient();
  const isCurrent = useCommittedScope(JSON.stringify([scope, issueId, enabled]));
  return useCallback(
    async (workspaceSlug: string, projectId: string, targetIssueId: string, data: Partial<TIssue>) => {
      if (
        !enabled ||
        !isCurrent() ||
        workspaceSlug !== scope.workspaceSlug ||
        projectId !== scope.projectId ||
        targetIssueId !== issueId
      )
        throw new Error("The work item or account session has changed.");
      if (!updateIssue) throw new Error("Work item editing is unavailable.");
      const detailOwner = captureIssueDetailReconciliation(client, scope, [targetIssueId]);
      const refreshActivity = captureIssueActivityRefresh(client, {
        workspaceSlug,
        projectId,
        issueId: targetIssueId,
        serviceType: scope.serviceType,
      });
      const receipt = await updateIssue(projectId, targetIssueId, data);
      await detailOwner.reconcile(receipt);
      if (detailOwner.isCurrent())
        void Promise.allSettled([refreshItemSchema(client, workspaceSlug, targetIssueId), refreshActivity()]);
    },
    [client, scope, issueId, enabled, isCurrent, updateIssue]
  );
}

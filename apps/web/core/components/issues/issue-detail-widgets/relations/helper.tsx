/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { TIssue, TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { useWorkItemDetailScope } from "@/hooks/use-work-item-detail";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { currentIssueListMutationOwner, issueListMutation, patchIssueMutation } from "@/queries/issue-list-mutations";

export type TRelationIssueOperations = {
  update: (data: Partial<TIssue>) => Promise<void>;
  remove: () => Promise<void>;
  captureAcceptedUpdate: () => (record: TIssue) => Promise<void>;
};

export const useRelationOperations = (
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  issueServiceType: TIssueServiceType = EIssueServiceType.ISSUES
): TRelationIssueOperations => {
  const client = useQueryClient();
  const access = useWorkItemDetailScope(workspaceSlug, projectId, issueServiceType, issueId);
  const mutationScopeIsCurrent = useCommittedScope(JSON.stringify([access.scope, issueId]));
  return useMemo(() => {
    const capture = () => {
      if (!access.isReady || !mutationScopeIsCurrent())
        throw new Error("The related work item or account session has changed.");
      return {
        detail: captureIssueDetailReconciliation(client, access.scope, [issueId]),
        list: currentIssueListMutationOwner(client, workspaceSlug, projectId, issueServiceType, mutationScopeIsCurrent),
      };
    };
    return {
      update: async (data) => {
        const owner = capture();
        const receipt = await patchIssueMutation(client, owner.list, issueId, data);
        await owner.detail.reconcile(receipt);
      },
      remove: async () => {
        const owner = capture();
        await issueListMutation(client, owner.list, { kind: "delete", issueId });
        await owner.detail.refresh();
        owner.detail.remove();
      },
      captureAcceptedUpdate: () => {
        const detail = captureIssueDetailReconciliation(client, access.scope, [issueId]);
        return async (record) => {
          if (!mutationScopeIsCurrent() || record.id !== issueId || record.project_id !== projectId) return;
          await detail.reconcile(record);
        };
      },
    };
  }, [
    access.isReady,
    access.scope,
    client,
    issueId,
    issueServiceType,
    mutationScopeIsCurrent,
    projectId,
    workspaceSlug,
  ]);
};

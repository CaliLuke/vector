/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { useQueryClient } from "@tanstack/react-query";
// plane imports
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TIssue, TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// hooks
import { useWorkItemDetailScope } from "@/hooks/use-work-item-detail";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { currentIssueListMutationOwner, issueListMutation, patchIssueMutation } from "@/queries/issue-list-mutations";

export type TRelationIssueOperations = {
  update: (workspaceSlug: string, projectId: string, issueId: string, data: Partial<TIssue>) => Promise<void>;
  remove: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
};

export const useRelationOperations = (
  workspaceSlug: string,
  projectId: string,
  issueId: string,
  issueServiceType: TIssueServiceType = EIssueServiceType.ISSUES
): TRelationIssueOperations => {
  const client = useQueryClient();
  const access = useWorkItemDetailScope(workspaceSlug, projectId, issueServiceType, issueId);
  const mutationScopeIsCurrent = useCommittedScope(JSON.stringify(access.scope));
  const { t } = useTranslation();
  // derived values
  const entityName = issueServiceType === EIssueServiceType.ISSUES ? "Work item" : "Epic";

  const issueOperations: TRelationIssueOperations = useMemo(
    () => ({
      update: async (targetWorkspaceSlug, targetProjectId, targetIssueId, data) => {
        try {
          const detailOwner = captureIssueDetailReconciliation(client, access.scope, [targetIssueId]);
          const listOwner = currentIssueListMutationOwner(
            client,
            targetWorkspaceSlug,
            targetProjectId,
            issueServiceType,
            mutationScopeIsCurrent
          );
          const receipt = await patchIssueMutation(client, listOwner, targetIssueId, data);
          await detailOwner.reconcile(receipt);
          setToast({
            title: t("toast.success"),
            type: TOAST_TYPE.SUCCESS,
            message: t("entity.update.success", { entity: entityName }),
          });
        } catch (_error) {
          setToast({
            title: t("toast.error"),
            type: TOAST_TYPE.ERROR,
            message: t("entity.update.failed", { entity: entityName }),
          });
        }
      },
      remove: async (targetWorkspaceSlug, targetProjectId, targetIssueId) => {
        const detailOwner = captureIssueDetailReconciliation(client, access.scope, [targetIssueId]);
        const listOwner = currentIssueListMutationOwner(
          client,
          targetWorkspaceSlug,
          targetProjectId,
          issueServiceType,
          mutationScopeIsCurrent
        );
        await issueListMutation(client, listOwner, { kind: "delete", issueId: targetIssueId });
        detailOwner.remove();
      },
    }),
    [access.scope, client, entityName, issueServiceType, mutationScopeIsCurrent, t]
  );

  return issueOperations;
};

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";

import { useQueryClient } from "@tanstack/react-query";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IIssueLabel, TIssue, TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// components
// hooks
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { currentIssueListMutationOwner, patchIssueMutation } from "@/queries/issue-list-mutations";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { useProjectLabelMutations } from "@/hooks/use-labels";
// ui
// types
import { LabelList, IssueLabelSelectRoot } from "./";
// TODO: Fix this import statement, as core should not import from ee

export type TIssueLabel = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  disabled: boolean;
  isInboxIssue?: boolean;
  labelIds?: string[];
  onLabelUpdate?: (labelIds: string[]) => void | Promise<void>;
  issueServiceType?: TIssueServiceType;
};

export type TLabelOperations = {
  updateIssue: (workspaceSlug: string, projectId: string, issueId: string, data: Partial<TIssue>) => Promise<void>;
  createLabel: (data: Partial<IIssueLabel>) => Promise<IIssueLabel>;
};

export const IssueLabel = function IssueLabel(props: TIssueLabel) {
  const {
    workspaceSlug,
    projectId,
    issueId,
    disabled = false,
    isInboxIssue = false,
    onLabelUpdate,
    issueServiceType = EIssueServiceType.ISSUES,
  } = props;
  const { t } = useTranslation();
  // hooks
  const client = useQueryClient();
  const { createLabel } = useProjectLabelMutations(workspaceSlug, projectId);
  const detail = useWorkItemDetail(workspaceSlug, projectId, issueId, issueServiceType);
  const mutationScopeIsCurrent = useCommittedScope(JSON.stringify(detail.scope));
  const labelIds = props.labelIds ?? detail.issue?.label_ids ?? [];

  const labelOperations: TLabelOperations = useMemo(
    () => ({
      updateIssue: async (targetWorkspaceSlug, targetProjectId, targetIssueId, data) => {
        try {
          if (onLabelUpdate) await onLabelUpdate(data.label_ids || []);
          else {
            const detailOwner = captureIssueDetailReconciliation(client, detail.scope, [targetIssueId]);
            const listOwner = currentIssueListMutationOwner(
              client,
              targetWorkspaceSlug,
              targetProjectId,
              issueServiceType,
              mutationScopeIsCurrent
            );
            const receipt = await patchIssueMutation(client, listOwner, targetIssueId, data);
            await detailOwner.reconcile(receipt);
          }
        } catch (error) {
          setToast({
            title: t("toast.error"),
            type: TOAST_TYPE.ERROR,
            message: t("entity.update.failed", { entity: t("issue.label", { count: 1 }) }),
          });
          throw error;
        }
      },
      createLabel: async (data: Partial<IIssueLabel>) => {
        try {
          const labelResponse = await createLabel(data);
          if (!isInboxIssue)
            setToast({
              title: t("toast.success"),
              type: TOAST_TYPE.SUCCESS,
              message: t("label.create.success"),
            });
          return labelResponse;
        } catch (error) {
          let errMessage = t("label.create.failed");
          if (
            typeof error === "object" &&
            error !== null &&
            "error" in error &&
            error.error === "Label with the same name already exists in the project"
          )
            errMessage = t("label.create.already_exists");

          setToast({
            title: t("toast.error"),
            type: TOAST_TYPE.ERROR,
            message: errMessage,
          });
          throw error;
        }
      },
    }),
    [client, createLabel, detail.scope, isInboxIssue, issueServiceType, mutationScopeIsCurrent, onLabelUpdate, t]
  );

  return (
    <div className="relative flex min-h-7.5 w-full flex-wrap items-center gap-1 px-2">
      <LabelList
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        values={labelIds}
        labelOperations={labelOperations}
        disabled={disabled}
      />

      {!disabled && (
        <IssueLabelSelectRoot
          workspaceSlug={workspaceSlug}
          projectId={projectId}
          issueId={issueId}
          values={labelIds}
          labelOperations={labelOperations}
        />
      )}
    </div>
  );
};

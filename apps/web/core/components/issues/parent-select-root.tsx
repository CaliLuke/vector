/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

// plane imports
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EIssueServiceType } from "@plane/types";
import type { TIssue } from "@plane/types";
import { useQueryClient } from "@tanstack/react-query";
// components
import type { TIssueOperations } from "@/components/issues/issue-detail";
import { IssueParentSelect } from "@/components/issues/issue-detail/parent-select";
// hooks
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import { useWorkItemDetail, useWorkItemDetailScope } from "@/hooks/use-work-item-detail";
import { captureIssueSubItemsReconciliation } from "@/queries/issue-details";
import { useSubIssueOperations } from "./issue-detail-widgets/sub-issues/helper";

type TIssueParentSelect = {
  className?: string;
  disabled?: boolean;
  issueId: string;
  issueOperations: TIssueOperations;
  projectId: string;
  workspaceSlug: string;
};

type TParentSelectionData = {
  issue: TIssue | undefined;
  parentId: string | undefined;
  parentProjectId: string;
  parentServiceType: EIssueServiceType;
  parentIssue: Partial<TIssue> | undefined;
  hasParentRelation: boolean;
};

type TParentIssueWithIdentity = Partial<TIssue> & { id: string; project_id: string };

function issueForRoute(
  record: ReturnType<typeof useWorkItemDetailRecord>,
  { workspaceSlug, projectId, issueId }: TIssueParentSelect
) {
  if (record?.workspaceSlug === workspaceSlug && record.projectId === projectId && record.issueId === issueId)
    return record.issue;
  return undefined;
}

function parentReadTarget(issue: TIssue | undefined, currentProjectId: string) {
  const parentSummary = issue?.parent;
  const parentId = parentSummary?.id ?? issue?.parent_id ?? undefined;
  const parentProjectId = parentSummary?.project_id ?? currentProjectId;
  const parentServiceType = parentSummary?.is_epic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES;
  const needsParentRead = Boolean(parentId && !hasParentSummaryIdentity(parentSummary));
  return { parentSummary, parentId, parentProjectId, parentServiceType, needsParentRead };
}

function hasParentSummaryIdentity(issue: Partial<TIssue> | undefined) {
  return Boolean(issue?.id && issue.project_id && issue.sequence_id !== undefined);
}

function resolvedParentIssue(
  parentId: string | undefined,
  shouldRead: boolean,
  parentSummary: Partial<TIssue> | undefined,
  readResult: TIssue | undefined
) {
  if (!parentId) return undefined;
  return shouldRead ? (readResult ?? parentSummary) : parentSummary;
}

function useParentSelectionData(props: TIssueParentSelect): TParentSelectionData {
  const record = useWorkItemDetailRecord();
  const issue = issueForRoute(record, props);
  const target = parentReadTarget(issue, props.projectId);
  const parentRead = useWorkItemDetail(
    props.workspaceSlug,
    target.parentProjectId,
    target.parentId,
    target.parentServiceType,
    target.needsParentRead
  );
  return {
    issue,
    parentId: target.parentId,
    parentProjectId: target.parentProjectId,
    parentServiceType: target.parentServiceType,
    parentIssue: resolvedParentIssue(target.parentId, target.needsParentRead, target.parentSummary, parentRead.issue),
    hasParentRelation: Boolean(target.parentId),
  };
}

function hasParentOperationIdentity(issue: Partial<TIssue> | undefined): issue is TParentIssueWithIdentity {
  return Boolean(issue?.id && issue.project_id);
}

function parentWorkItemLink(workspaceSlug: string, issue: Partial<TIssue> | undefined) {
  if (!hasParentOperationIdentity(issue)) return undefined;
  return `/${workspaceSlug}/projects/${issue.project_id}/issues/${issue.id}`;
}

function useParentSelectionActions(props: TIssueParentSelect, data: TParentSelectionData) {
  const { issueId, issueOperations, projectId, workspaceSlug } = props;
  const { t } = useTranslation();
  const client = useQueryClient();
  const detailAccess = useWorkItemDetailScope(workspaceSlug, projectId, EIssueServiceType.ISSUES, issueId);
  const parentScopeProject = data.parentIssue?.project_id ?? data.parentProjectId;
  const parentScopeId = data.parentIssue?.id ?? data.parentId ?? "";
  const parentScopeType = data.parentIssue?.is_epic ? EIssueServiceType.EPICS : data.parentServiceType;
  const subIssueOperations = useSubIssueOperations(workspaceSlug, parentScopeProject, parentScopeId, parentScopeType);

  const handleParentIssue = async (targetParentId: string) => {
    const parentQueries = captureIssueSubItemsReconciliation(
      client,
      detailAccess.scope,
      [data.parentId, targetParentId].filter((id): id is string => Boolean(id))
    );
    await issueOperations.update(workspaceSlug, projectId, issueId, { parent_id: targetParentId });
    await Promise.all([parentQueries.refresh(), issueOperations.fetch(workspaceSlug, projectId, issueId, false)]);
  };

  const handleRemoveSubIssue = async () => {
    if (!hasParentOperationIdentity(data.parentIssue)) return;
    try {
      await subIssueOperations.removeSubIssue(workspaceSlug, projectId, data.parentIssue.id, issueId);
    } catch (_error) {
      setToast({
        type: TOAST_TYPE.ERROR,
        title: t("common.error.label"),
        message: t("common.something_went_wrong"),
      });
    }
  };
  return { handleParentIssue, handleRemoveSubIssue };
}

export const IssueParentSelectRoot = function IssueParentSelectRoot(props: TIssueParentSelect) {
  const data = useParentSelectionData(props);
  const actions = useParentSelectionActions(props, data);
  if (!data.issue) return null;
  return (
    <IssueParentSelect
      {...props}
      parentIssue={data.parentIssue}
      hasParentRelation={data.hasParentRelation}
      {...actions}
      workItemLink={parentWorkItemLink(props.workspaceSlug, data.parentIssue)}
    />
  );
};

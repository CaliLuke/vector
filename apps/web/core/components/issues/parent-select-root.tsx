/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";

// plane imports
import { useTranslation } from "@plane/i18n";
import { EIssueServiceType } from "@plane/types";
import type { TIssue } from "@plane/types";
import { useQueryClient } from "@tanstack/react-query";
// components
import type { TIssueOperations } from "@/components/issues/issue-detail";
import { IssueParentSelect } from "@/components/issues/issue-detail/parent-select";
// hooks
import { useWorkItemDetailRecord } from "@/providers/work-item-detail-record";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { currentParentCatalogueScope } from "@/queries/cycle-module-scope";
import { captureIssueSubItemsReconciliation } from "@/queries/issue-details";
import { useIssuePropertyChange } from "@/hooks/use-issue-property-change";
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
  const parentScopeProject = data.parentIssue?.project_id ?? data.parentProjectId;
  const parentScopeId = data.parentIssue?.id ?? data.parentId ?? "";
  const parentScopeType = data.parentIssue?.is_epic ? EIssueServiceType.EPICS : data.parentServiceType;
  const subIssueOperations = useSubIssueOperations(workspaceSlug, parentScopeProject, parentScopeId, parentScopeType);

  const handleParentIssue = async (targetParentId: string, targetParentProjectId: string) => {
    if (targetParentId === data.parentId && targetParentProjectId === parentScopeProject) return;
    const parents = [
      { id: data.parentId, projectId: parentScopeProject, serviceType: parentScopeType },
      { id: targetParentId, projectId: targetParentProjectId, serviceType: EIssueServiceType.ISSUES },
    ];
    const readers = parents.flatMap((parent) => {
      if (!parent.id) return [];
      const { workspaceRole, ...scope } = currentParentCatalogueScope(client, workspaceSlug, parent.projectId).project;
      return [
        captureIssueSubItemsReconciliation(
          client,
          { ...scope, membershipRole: workspaceRole, serviceType: parent.serviceType },
          [parent.id]
        ),
      ];
    });
    await issueOperations.update(workspaceSlug, projectId, issueId, { parent_id: targetParentId });
    await Promise.all(readers.map((reader) => reader.refresh()));
  };

  const entity =
    parentScopeType === EIssueServiceType.ISSUES ? t("common.sub_work_items") : t("issue.label", { count: 1 });
  const removal = useIssuePropertyChange(
    { workspaceSlug, projectId, issueId, context: `parent:${parentScopeProject}:${parentScopeId}:${parentScopeType}` },
    !props.disabled && hasParentOperationIdentity(data.parentIssue),
    {
      success: t("entity.remove.success", { entity }),
      error: t("entity.remove.failed", { entity }),
    }
  );
  const handleRemoveSubIssue = () => {
    const parent = data.parentIssue;
    if (hasParentOperationIdentity(parent))
      removal.run(() => subIssueOperations.removeSubIssue(workspaceSlug, projectId, parent.id, issueId));
  };
  return { handleParentIssue, handleRemoveSubIssue, isRemovingParent: removal.pending };
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

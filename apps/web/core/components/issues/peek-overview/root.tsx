/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo, useCallback } from "react";
import { useCommittedScope } from "@/hooks/use-committed-scope";

// Vector imports
import { EUserPermissions } from "@plane/constants";
import type { IWorkItemPeekOverview } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
import { useIssueDetailLifecycle } from "@/hooks/use-issue-detail-lifecycle";
import { useIssueDetailWrites } from "@/hooks/use-issue-detail-writes";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useIssueListType } from "@/hooks/use-issue-list-type";
import { useWorkItemProperties } from "@/hooks/use-issue-properties";
import { useIssuesActions } from "@/hooks/use-issues-actions";
// local imports
import type { TIssueOperations } from "../issue-detail";
import { IssueView } from "./view";
import { WorkItemDetailRecordProvider } from "@/providers/work-item-detail-record";

export type TIssuePeekOperations = TIssueOperations & {
  remove: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  archive: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  restore: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
};

export function IssuePeekOverview(props: IWorkItemPeekOverview) {
  const { embedIssue = false, embedRemoveCurrentNotification, storeType: issueStoreFromProps } = props;
  // router
  const { peekIssue, setPeekIssue } = useIssueDetailUiState();
  const { workspaceSlug: peekWorkspaceSlug, projectId: peekProjectId, issueId: peekIssueId } = peekIssue ?? {};
  const { can: canProject } = useProjectPermissions(peekWorkspaceSlug, peekProjectId);
  const issueStoreType = useIssueListType();
  const storeType = issueStoreFromProps ?? issueStoreType;
  const actions = useIssuesActions(storeType);
  const serviceType = storeType === EIssuesStoreType.EPIC ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES;
  const detail = useWorkItemDetail(peekWorkspaceSlug, peekProjectId, peekIssueId, serviceType);
  const lifecycle = useIssueDetailLifecycle(detail.scope, peekIssueId, detail.isReady);
  const isPeekScopeCurrent = useCommittedScope(
    JSON.stringify([detail.scope, peekWorkspaceSlug, peekProjectId, peekIssueId, storeType, !!peekIssue?.isArchived])
  );

  useWorkItemProperties(
    peekProjectId,
    peekWorkspaceSlug,
    peekIssueId,
    storeType === EIssuesStoreType.EPIC ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES
  );
  const removeRoutePeekId = useCallback(() => {
    setPeekIssue(undefined);
    if (embedIssue && embedRemoveCurrentNotification)
      void Promise.resolve()
        .then(embedRemoveCurrentNotification)
        .catch((error) => console.error("Could not dismiss the work item notification", error));
  }, [embedIssue, embedRemoveCurrentNotification, setPeekIssue]);

  const isPeekIssueArchived = peekIssue?.isArchived;
  const writes = useIssueDetailWrites(detail.scope, peekIssueId, detail.isReady && !isPeekIssueArchived, actions);
  const issueOperations: TIssuePeekOperations = useMemo(
    () => ({
      ...writes,
      updateText: writes.update,
      remove: async (workspaceSlug, projectId, issueId) => {
        const receipt = await lifecycle.remove(workspaceSlug, projectId, issueId);
        if (isPeekScopeCurrent() && receipt.isCurrent()) removeRoutePeekId();
      },
      archive: async (workspaceSlug, projectId, issueId) => {
        const receipt = await lifecycle.archive(workspaceSlug, projectId, issueId);
        if (isPeekScopeCurrent() && receipt.isCurrent()) removeRoutePeekId();
      },
      restore: async (workspaceSlug, projectId, issueId) => {
        const receipt = await lifecycle.restore(workspaceSlug, projectId, issueId);
        if (isPeekScopeCurrent() && receipt.isCurrent()) removeRoutePeekId();
      },
    }),
    [writes, lifecycle, isPeekScopeCurrent, removeRoutePeekId]
  );

  const isLoading = detail.isPending;
  const isError = detail.isError || detail.isDenied;

  if (!peekIssue || !peekWorkspaceSlug || !peekProjectId || !peekIssueId) return <></>;

  // Check if issue is editable, based on user role
  const isEditable = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const view = (
    <IssueView
      workspaceSlug={peekIssue.workspaceSlug}
      projectId={peekIssue.projectId}
      issueId={peekIssue.issueId}
      isLoading={isLoading}
      isError={isError}
      is_archived={!!peekIssue.isArchived}
      disabled={!isEditable}
      embedIssue={embedIssue}
      embedRemoveCurrentNotification={embedRemoveCurrentNotification}
      issueOperations={issueOperations}
    />
  );
  if (!detail.issue) return view;
  return (
    <WorkItemDetailRecordProvider
      key={`${peekWorkspaceSlug}/${peekProjectId}/${peekIssueId}/${serviceType}`}
      value={{
        workspaceSlug: peekWorkspaceSlug,
        projectId: peekProjectId,
        issueId: peekIssueId,
        serviceType,
        issue: detail.issue,
      }}
    >
      {view}
    </WorkItemDetailRecordProvider>
  );
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { useMemo, useCallback } from "react";
import { useCommittedScope } from "@/hooks/use-committed-scope";

// Vector imports
import { useQueryClient } from "@tanstack/react-query";
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IWorkItemPeekOverview } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
// hooks
import { useIssueDetailUiState } from "@/providers/issue-detail-ui-state";
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
  archive?: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  restore?: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
};

export function IssuePeekOverview(props: IWorkItemPeekOverview) {
  const { embedIssue = false, embedRemoveCurrentNotification, storeType: issueStoreFromProps } = props;
  const { t } = useTranslation();
  // router
  const { peekIssue, setPeekIssue } = useIssueDetailUiState();
  const { workspaceSlug: peekWorkspaceSlug, projectId: peekProjectId, issueId: peekIssueId } = peekIssue ?? {};
  const { can: canProject } = useProjectPermissions(peekWorkspaceSlug, peekProjectId);
  const issueStoreType = useIssueListType();
  const storeType = issueStoreFromProps ?? issueStoreType;
  const actions = useIssuesActions(storeType);
  const archivedActions = useIssuesActions(EIssuesStoreType.ARCHIVED);
  const serviceType = storeType === EIssuesStoreType.EPIC ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES;
  const detail = useWorkItemDetail(peekWorkspaceSlug, peekProjectId, peekIssueId, serviceType);
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
    if (embedIssue) embedRemoveCurrentNotification?.();
  }, [embedIssue, embedRemoveCurrentNotification, setPeekIssue]);

  const queryClient = useQueryClient();
  const isPeekIssueArchived = peekIssue?.isArchived;
  const writes = useIssueDetailWrites(detail.scope, peekIssueId, detail.isReady && !isPeekIssueArchived, actions);
  const issueOperations: TIssuePeekOperations = useMemo(
    () => ({
      ...writes,
      updateText: writes.update,
      fetch: async (workspaceSlug: string, projectId: string, issueId: string) => {
        if (workspaceSlug === peekWorkspaceSlug && projectId === peekProjectId && issueId === peekIssueId)
          await detail.query.refetch();
      },
      remove: async (workspaceSlug: string, projectId: string, issueId: string) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [issueId]);
        try {
          if (isPeekIssueArchived) await archivedActions.removeIssue(projectId, issueId);
          else await actions.removeIssue(projectId, issueId);
          detailOwner.remove();
          if (isPeekScopeCurrent()) removeRoutePeekId();
        } catch (_error) {
          setToast({
            title: t("toast.error"),
            type: TOAST_TYPE.ERROR,
            message: t("entity.delete.failed", { entity: t("issue.label", { count: 1 }) }),
          });
        }
      },
      archive: async (workspaceSlug: string, projectId: string, issueId: string) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [issueId]);
        if (!actions.archiveIssue) throw new Error("Issue archive action is unavailable");
        await actions.archiveIssue(projectId, issueId);
        void detailOwner.refresh();
      },
      restore: async (workspaceSlug: string, projectId: string, issueId: string) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [issueId]);
        try {
          if (!archivedActions.restoreIssue) throw new Error("Issue restore action is unavailable");
          await archivedActions.restoreIssue(projectId, issueId);
          void detailOwner.refresh();
          setToast({
            type: TOAST_TYPE.SUCCESS,
            title: t("issue.restore.success.title"),
            message: t("issue.restore.success.message"),
          });
        } catch (_error) {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("toast.error"),
            message: t("issue.restore.failed.message"),
          });
        }
      },
    }),
    [
      writes,
      queryClient,
      detail.query,
      detail.scope,
      actions,
      archivedActions,
      isPeekIssueArchived,
      peekIssueId,
      peekProjectId,
      peekWorkspaceSlug,
      removeRoutePeekId,
      isPeekScopeCurrent,
      t,
    ]
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

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useMemo } from "react";
// plane imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setPromiseToast, setToast } from "@plane/propel/toast";
import type { TIssue } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
// assets
import emptyIssue from "@/app/assets/empty-state/issue.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
// hooks
import { useIssueDetailSidebarCollapsed } from "@/providers/sidebar-state";
import { useIssueDetailUpdate } from "@/hooks/use-issue-detail-update";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { captureIssueDetailReconciliation } from "@/queries/issue-details";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useIssuesActions } from "@/hooks/use-issues-actions";

// local components
import { IssuePeekOverview } from "../peek-overview";
import { IssueMainContent } from "./main-content";
import { IssueDetailsSidebar } from "./sidebar";
import { WorkItemDetailRecordProvider } from "@/providers/work-item-detail-record";

export type TIssueOperations = {
  fetch: (workspaceSlug: string, projectId: string, issueId: string, loader?: boolean) => Promise<void>;
  updateText: (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: Pick<Partial<TIssue>, "name" | "description_html"> & { skip_activity?: string }
  ) => Promise<void>;
  update: (workspaceSlug: string, projectId: string, issueId: string, data: Partial<TIssue>) => Promise<void>;
  remove: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  archive?: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  restore?: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  addCycleToIssue?: (workspaceSlug: string, projectId: string, cycleId: string, issueId: string) => Promise<void>;
  addIssueToCycle?: (workspaceSlug: string, projectId: string, cycleId: string, issueIds: string[]) => Promise<void>;
  removeIssueFromCycle?: (workspaceSlug: string, projectId: string, cycleId: string, issueId: string) => Promise<void>;
  removeIssueFromModule?: (
    workspaceSlug: string,
    projectId: string,
    moduleId: string,
    issueId: string
  ) => Promise<void>;
  changeModulesInIssue?: (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    addModuleIds: string[],
    removeModuleIds: string[]
  ) => Promise<void>;
};

export type TIssueDetailRoot = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  is_archived?: boolean;
};

export function IssueDetailRoot(props: TIssueDetailRoot) {
  const { t } = useTranslation();
  const { workspaceSlug, projectId, issueId, is_archived = false } = props;
  // router
  const router = useRouter();
  // hooks
  const detail = useWorkItemDetail(workspaceSlug, projectId, issueId);
  const projectActions = useIssuesActions(EIssuesStoreType.PROJECT);
  const archivedActions = useIssuesActions(EIssuesStoreType.ARCHIVED);
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const issueDetailSidebarCollapsed = useIssueDetailSidebarCollapsed();

  const queryClient = useQueryClient();
  const update = useIssueDetailUpdate(
    detail.scope,
    issueId,
    detail.isReady && !is_archived,
    projectActions.updateIssue
  );
  const issueOperations: TIssueOperations = useMemo(
    () => ({
      updateText: update,
      fetch: async (targetWorkspaceSlug, targetProjectId, targetIssueId) => {
        if (targetWorkspaceSlug !== workspaceSlug || targetProjectId !== projectId || targetIssueId !== issueId) return;
        await detail.query.refetch();
      },
      update,
      remove: async (_targetWorkspaceSlug, targetProjectId, targetIssueId) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [targetIssueId]);
        try {
          if (is_archived) await archivedActions.removeIssue(targetProjectId, targetIssueId);
          else await projectActions.removeIssue(targetProjectId, targetIssueId);
          detailOwner.remove();
          setToast({
            title: t("common.success"),
            type: TOAST_TYPE.SUCCESS,
            message: t("entity.delete.success", { entity: t("issue.label") }),
          });
        } catch (_error) {
          setToast({
            title: t("common.error.label"),
            type: TOAST_TYPE.ERROR,
            message: t("entity.delete.failed", { entity: t("issue.label") }),
          });
        }
      },
      archive: async (_targetWorkspaceSlug, targetProjectId, targetIssueId) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [targetIssueId]);
        try {
          if (!projectActions.archiveIssue) throw new Error("Issue archive action is unavailable.");
          await projectActions.archiveIssue(targetProjectId, targetIssueId);
          void detailOwner.refresh();
        } catch (_error) {
          setToast({
            title: t("common.error.label"),
            type: TOAST_TYPE.ERROR,
            message: t("entity.update.failed", { entity: t("issue.label") }),
          });
        }
      },
      addCycleToIssue: async (_targetWorkspaceSlug, targetProjectId, cycleId, targetIssueId) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [targetIssueId]);
        try {
          await projectActions.addCycleToIssue(targetProjectId, cycleId, targetIssueId);
          void detailOwner.refresh();
        } catch (_error) {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("common.error.label"),
            message: t("issue.add.cycle.failed"),
          });
        }
      },
      addIssueToCycle: async (_targetWorkspaceSlug, targetProjectId, cycleId, issueIds) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, issueIds);
        try {
          await projectActions.addIssuesToCycle(targetProjectId, cycleId, issueIds);
          void detailOwner.refresh();
        } catch (_error) {
          setToast({
            type: TOAST_TYPE.ERROR,
            title: t("common.error.label"),
            message: t("issue.add.cycle.failed"),
          });
        }
      },
      removeIssueFromCycle: async (_targetWorkspaceSlug, targetProjectId, cycleId, targetIssueId) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [targetIssueId]);
        try {
          const removeFromCyclePromise = projectActions.removeCycleFromIssue(targetProjectId, cycleId, targetIssueId);
          setPromiseToast(removeFromCyclePromise, {
            loading: t("issue.remove.cycle.loading"),
            success: {
              title: t("common.success"),
              message: () => t("issue.remove.cycle.success"),
            },
            error: {
              title: t("common.error.label"),
              message: () => t("issue.remove.cycle.failed"),
            },
          });
          await removeFromCyclePromise;
          void detailOwner.refresh();
        } catch (_error) {
          // The promise toast above reports the failure.
        }
      },
      removeIssueFromModule: async (_targetWorkspaceSlug, targetProjectId, moduleId, targetIssueId) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [targetIssueId]);
        try {
          const removeFromModulePromise = projectActions.removeIssueFromModule(
            targetProjectId,
            moduleId,
            targetIssueId
          );
          setPromiseToast(removeFromModulePromise, {
            loading: t("issue.remove.module.loading"),
            success: {
              title: t("common.success"),
              message: () => t("issue.remove.module.success"),
            },
            error: {
              title: t("common.error.label"),
              message: () => t("issue.remove.module.failed"),
            },
          });
          await removeFromModulePromise;
          void detailOwner.refresh();
        } catch (_error) {
          // The promise toast above reports the failure.
        }
      },
      changeModulesInIssue: async (
        _targetWorkspaceSlug,
        targetProjectId,
        targetIssueId,
        addModuleIds,
        removeModuleIds
      ) => {
        const detailOwner = captureIssueDetailReconciliation(queryClient, detail.scope, [targetIssueId]);
        const promise = await projectActions.changeModulesInIssue(
          targetProjectId,
          targetIssueId,
          addModuleIds,
          removeModuleIds
        );
        void detailOwner.refresh();
        return promise;
      },
    }),
    [
      update,
      queryClient,
      is_archived,
      detail.query,
      detail.scope,
      projectActions,
      archivedActions,
      t,
      workspaceSlug,
      projectId,
      issueId,
    ]
  );

  // issue details
  const issue = detail.issue;
  // checking if issue is editable, based on user role
  const isEditable = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  return (
    <>
      {!issue ? (
        <EmptyState
          image={emptyIssue}
          title={t("issue.empty_state.issue_detail.title")}
          description={t("issue.empty_state.issue_detail.description")}
          primaryButton={{
            text: t("issue.empty_state.issue_detail.primary_button.text"),
            onClick: () => void router.navigate({ href: `/${workspaceSlug}/projects/${projectId}/issues` }),
          }}
        />
      ) : (
        <WorkItemDetailRecordProvider
          key={`${workspaceSlug}/${projectId}/${issueId}/issues`}
          value={{ workspaceSlug, projectId, issueId, serviceType: EIssueServiceType.ISSUES, issue }}
        >
          <div className="flex h-full w-full overflow-hidden">
            <div className="h-full w-full space-y-6 overflow-y-auto px-9 py-5">
              <IssueMainContent
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                issueId={issueId}
                issueOperations={issueOperations}
                isEditable={isEditable}
                isArchived={is_archived}
              />
            </div>
            <div
              className="fixed right-0 z-[5] h-full w-full min-w-[300px] border-l border-subtle bg-surface-1 sm:w-1/2 md:relative md:w-1/4 lg:min-w-80 xl:min-w-96"
              style={issueDetailSidebarCollapsed ? { right: `-${window?.innerWidth || 0}px` } : {}}
            >
              <IssueDetailsSidebar
                workspaceSlug={workspaceSlug}
                projectId={projectId}
                issueId={issueId}
                issueOperations={issueOperations}
                isEditable={!is_archived && isEditable}
              />
            </div>
          </div>
        </WorkItemDetailRecordProvider>
      )}

      {/* peek overview */}
      <IssuePeekOverview />
    </>
  );
}

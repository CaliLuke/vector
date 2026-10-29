/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRouter } from "@tanstack/react-router";
import { useMemo } from "react";
// plane imports
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import type { TIssue } from "@plane/types";
import { EIssueServiceType, EIssuesStoreType } from "@plane/types";
// assets
import emptyIssue from "@/app/assets/empty-state/issue.svg?url";
// components
import { EmptyState } from "@/components/common/empty-state";
// hooks
import { useIssueDetailSidebarCollapsed } from "@/providers/sidebar-state";
import { useIssueDetailWrites } from "@/hooks/use-issue-detail-writes";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useIssuesActions } from "@/hooks/use-issues-actions";

// local components
import { IssuePeekOverview } from "../peek-overview";
import { IssueMainContent } from "./main-content";
import { IssueDetailsSidebar } from "./sidebar";
import { WorkItemDetailRecordProvider } from "@/providers/work-item-detail-record";

export type TIssueOperations = {
  updateText: (
    workspaceSlug: string,
    projectId: string,
    issueId: string,
    data: Pick<Partial<TIssue>, "name" | "description_html"> & { skip_activity?: string }
  ) => Promise<void>;
  update: (workspaceSlug: string, projectId: string, issueId: string, data: Partial<TIssue>) => Promise<void>;
  addCycleToIssue?: (workspaceSlug: string, projectId: string, cycleId: string, issueId: string) => Promise<void>;
  removeIssueFromCycle?: (workspaceSlug: string, projectId: string, cycleId: string, issueId: string) => Promise<void>;
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
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const issueDetailSidebarCollapsed = useIssueDetailSidebarCollapsed();

  const writes = useIssueDetailWrites(detail.scope, issueId, detail.isReady && !is_archived, projectActions);
  const issueOperations: TIssueOperations = useMemo(
    () => ({
      ...writes,
      updateText: writes.update,
    }),
    [writes]
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

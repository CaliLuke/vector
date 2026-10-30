/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useRouter } from "@tanstack/react-router";
import { useEffect } from "react";
import { useTheme } from "next-themes";
// plane imports
import { useTranslation } from "@plane/i18n";
import { EIssueServiceType } from "@plane/types";
import { Loader } from "@plane/ui";
// assets
import emptyIssueDark from "@/app/assets/empty-state/search/issues-dark.webp?url";
import emptyIssueLight from "@/app/assets/empty-state/search/issues-light.webp?url";
// components
import { EmptyState } from "@/components/common/empty-state";
import { PageHead } from "@/components/core/page-title";
// hooks
import { useSidebarActions, useIssueDetailSidebarCollapsed } from "@/providers/sidebar-state";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkItemByIdentifier } from "@/hooks/use-work-item-detail";

// layouts
import { ProjectAuthWrapper } from "@/layouts/auth-layout/project-wrapper";
// plane web imports
import { useWorkItemProperties } from "@/hooks/use-issue-properties";
import { WorkItemDetailRoot } from "@/components/browse/workItem-detail";

function IssueDetailsPage(): React.JSX.Element | null {
  // router
  const router = useRouter();
  const { workspaceSlug, workItem } = Route.useParams();
  // hooks
  const { resolvedTheme } = useTheme();
  // store hooks
  const { t } = useTranslation();
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const issueDetailSidebarCollapsed = useIssueDetailSidebarCollapsed();
  const { toggleIssueDetailSidebar } = useSidebarActions();

  const [projectIdentifier, sequence_id] = workItem.split("-");

  const projectDetails = projectRows?.find((project) => project.identifier === projectIdentifier);
  const detail = useWorkItemByIdentifier(workspaceSlug, projectIdentifier, sequence_id, projectDetails?.id);
  const { issue, isPending: isLoading, isError: error } = detail;

  // derived values
  const issueId = issue?.id;
  const projectId = issue?.project_id ?? projectDetails?.id ?? "";
  const project =
    (issue?.project_id && projectRows?.find((projectRow) => projectRow.id === issue?.project_id)) || undefined;
  const issueLoader = !issue || isLoading;
  const pageTitle = project && issue ? `${project.identifier}-${issue.sequence_id} ${issue.name}` : undefined;

  useWorkItemProperties(
    projectId,
    workspaceSlug,
    issueId,
    issue?.is_epic ? EIssueServiceType.EPICS : EIssueServiceType.ISSUES
  );

  useEffect(() => {
    const handleToggleIssueDetailSidebar = () => {
      if (window && window.innerWidth < 768) {
        toggleIssueDetailSidebar(true);
      }
      if (window && issueDetailSidebarCollapsed && window.innerWidth >= 768) {
        toggleIssueDetailSidebar(false);
      }
    };
    window.addEventListener("resize", handleToggleIssueDetailSidebar);
    handleToggleIssueDetailSidebar();
    return () => window.removeEventListener("resize", handleToggleIssueDetailSidebar);
  }, [issueDetailSidebarCollapsed, toggleIssueDetailSidebar]);

  useEffect(() => {
    if (issue?.is_intake) {
      void router.navigate({
        href: `/${workspaceSlug}/projects/${issue.project_id}/intake/?currentTab=open&inboxIssueId=${issue.id}`,
      });
    }
  }, [workspaceSlug, issue, router]);

  if (error && !isLoading) {
    return (
      <EmptyState
        image={resolvedTheme === "dark" ? emptyIssueDark : emptyIssueLight}
        title={t("issue.empty_state.issue_detail.title")}
        description={t("issue.empty_state.issue_detail.description")}
        primaryButton={{
          text: t("issue.empty_state.issue_detail.primary_button.text"),
          onClick: () => void router.navigate({ href: `/${workspaceSlug}/workspace-views/all-issues/` }),
        }}
      />
    );
  }

  if (issueLoader) {
    return (
      <Loader className="flex h-full gap-5 p-5">
        <div className="basis-2/3 space-y-2">
          <Loader.Item height="30px" width="40%" />
          <Loader.Item height="15px" width="60%" />
          <Loader.Item height="15px" width="60%" />
          <Loader.Item height="15px" width="40%" />
        </div>
        <div className="basis-1/3 space-y-3">
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
        </div>
      </Loader>
    );
  }

  return (
    <>
      <PageHead title={pageTitle} />
      {workspaceSlug && projectId && issueId && (
        <ProjectAuthWrapper workspaceSlug={workspaceSlug} projectId={projectId}>
          <WorkItemDetailRoot workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId} issue={issue} />
        </ProjectAuthWrapper>
      )}
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/browse/$workItem/")({
  component: IssueDetailsPage,
});

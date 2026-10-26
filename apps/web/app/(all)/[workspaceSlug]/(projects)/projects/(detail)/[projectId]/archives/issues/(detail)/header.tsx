/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

import { useQuery, skipToken } from "@tanstack/react-query";
// ui
import { ArchiveIcon, WorkItemsIcon } from "@plane/propel/icons";
import { Breadcrumbs, Header } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { IssueDetailQuickActions } from "@/components/issues/issue-detail/issue-detail-quick-actions";
// constants
// hooks
import { useRouteProject } from "@/hooks/use-projects";
// plane web
import { ProjectBreadcrumb } from "@/components/breadcrumbs/project";
// services
import { IssueService } from "@/services/issue";

const issueService = new IssueService();

export const ProjectArchivedIssueDetailsHeader = function ProjectArchivedIssueDetailsHeader() {
  // router
  const { workspaceSlug, projectId, archivedIssueId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/archives/issues/$archivedIssueId",
  });
  // store hooks
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;
  const loader = routeProjectQuery.isPending;

  const { data: issueDetails } = useQuery({
    queryKey: ["ISSUE_DETAILS", workspaceSlug, projectId, archivedIssueId],
    queryFn:
      workspaceSlug && projectId && archivedIssueId
        ? () => issueService.retrieve(workspaceSlug.toString(), projectId.toString(), archivedIssueId.toString())
        : skipToken,
  });

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs isLoading={loader}>
          <ProjectBreadcrumb workspaceSlug={workspaceSlug?.toString()} projectId={projectId?.toString()} />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                href={`/${workspaceSlug}/projects/${projectId}/archives/issues`}
                label="Archives"
                icon={<ArchiveIcon className="h-4 w-4 text-tertiary" />}
              />
            }
          />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                href={`/${workspaceSlug}/projects/${projectId}/archives/issues`}
                label="Work items"
                icon={<WorkItemsIcon className="h-4 w-4 text-tertiary" />}
              />
            }
          />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label={
                  currentProjectDetails && issueDetails
                    ? `${currentProjectDetails.identifier}-${issueDetails.sequence_id}`
                    : ""
                }
              />
            }
          />
        </Breadcrumbs>
      </Header.LeftItem>
      <Header.RightItem>
        <IssueDetailQuickActions
          workspaceSlug={workspaceSlug.toString()}
          projectId={projectId.toString()}
          issueId={archivedIssueId.toString()}
        />
      </Header.RightItem>
    </Header>
  );
};

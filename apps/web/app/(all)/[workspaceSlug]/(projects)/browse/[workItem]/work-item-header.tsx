/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams, useRouter } from "@tanstack/react-router";
import React from "react";

// plane ui
import { WorkItemsIcon } from "@plane/propel/icons";
import { Breadcrumbs, Header } from "@plane/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { IssueDetailQuickActions } from "@/components/issues/issue-detail/issue-detail-quick-actions";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkItemByIdentifier } from "@/hooks/use-work-item-detail";

// plane web imports
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";

export const WorkItemDetailsHeader = function WorkItemDetailsHeader() {
  // router
  const router = useRouter();
  const { workspaceSlug, workItem } = useParams({ from: "/_authenticated/$workspaceSlug/_projects/browse/$workItem" });
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const loader = projectCatalogue.isPending;
  const [projectIdentifier, sequenceId] = workItem.split("-");
  const projectIdFromCatalogue = projectRows?.find((project) => project.identifier === projectIdentifier)?.id;
  const detail = useWorkItemByIdentifier(workspaceSlug, projectIdentifier, sequenceId, projectIdFromCatalogue);
  const issueDetails = detail.issue;
  const issueId = issueDetails?.id;
  const projectId = issueDetails?.project_id;
  const projectDetails = projectId ? projectRows?.find((project) => project.id === projectId) : undefined;

  if (!workspaceSlug || !projectId || !issueId) return null;
  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs onBack={router.history.back} isLoading={loader}>
          <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label="Work Items"
                href={`/${workspaceSlug}/projects/${projectId}/issues/`}
                icon={<WorkItemsIcon className="h-4 w-4 text-tertiary" />}
              />
            }
          />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label={projectDetails && issueDetails ? `${projectDetails.identifier}-${issueDetails.sequence_id}` : ""}
              />
            }
          />
        </Breadcrumbs>
      </Header.LeftItem>
      <Header.RightItem>
        <IssueDetailQuickActions workspaceSlug={workspaceSlug} projectId={projectId} issueId={issueId} />
      </Header.RightItem>
    </Header>
  );
};

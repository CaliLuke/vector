/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { ProjectTabHeader } from "@/components/navigation/project-tab-header";
// components
import { AppHeader } from "@/components/core/app-header";
// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useWorkItemByIdentifier } from "@/hooks/use-work-item-detail";
// local components
import { WorkItemDetailsHeader } from "./work-item-header";

export const ProjectWorkItemDetailsHeader = function ProjectWorkItemDetailsHeader() {
  // router
  const { workspaceSlug, workItem } = useParams({ from: "/_authenticated/$workspaceSlug/_projects/browse/$workItem" });
  // store hooks
  const projectCatalogue = useWorkspaceProjectDetails();
  const [projectIdentifier, sequenceId] = (workItem ?? "").split("-");
  const projectId = projectCatalogue.projects?.find((project) => project.identifier === projectIdentifier)?.id;
  const detail = useWorkItemByIdentifier(workspaceSlug, projectIdentifier, sequenceId, projectId);
  const issueDetails = detail.issue;

  return (
    <>
      <ProjectTabHeader workspaceSlug={workspaceSlug} projectId={issueDetails?.project_id?.toString() ?? ""} />
      <AppHeader header={<WorkItemDetailsHeader />} />
    </>
  );
};

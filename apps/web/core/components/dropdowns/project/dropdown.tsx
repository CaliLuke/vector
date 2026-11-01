/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";
// local imports
import type { ProjectDropdownProps } from "./base";
import { ProjectDropdownBase } from "./base";

export const ProjectDropdown = function ProjectDropdown(props: ProjectDropdownProps) {
  // store hooks
  const { workspaceSlug } = useParams({ strict: false });
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const joinedProjectIds = useJoinedProjectIds(workspaceSlug?.toString());

  return (
    <ProjectDropdownBase
      {...props}
      getProjectById={(projectId) => projectRows?.find((projectRow) => projectRow.id === projectId)}
      projectIds={joinedProjectIds}
    />
  );
};

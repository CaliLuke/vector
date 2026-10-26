/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import { useWorkspaceProjects } from "@/hooks/use-projects";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { effectiveProjectRole } from "@/helpers/permission-policy";
import sortBy from "lodash-es/sortBy.js";

export function useJoinedProjectIds(workspaceSlug: string | undefined) {
  const { data: workspace } = useWorkspaceBySlug(workspaceSlug);
  const { roles, workspaceRole } = useWorkspaceProjectRoles(workspaceSlug);
  const projectCatalogue = useWorkspaceProjects(workspaceSlug);
  const joinedProjectIds = new Set(
    roles
      ? Object.keys(roles).filter((projectId) => effectiveProjectRole(workspaceRole, roles, projectId) !== undefined)
      : []
  );
  return sortBy(projectCatalogue.projects ?? [], "sort_order")
    .filter(
      (project) => project.workspace === workspace?.id && !project.archived_at && joinedProjectIds.has(project.id)
    )
    .map((project) => project.id);
}

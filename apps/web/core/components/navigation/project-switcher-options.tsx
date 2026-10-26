/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ProjectIcon } from "@plane/propel/icons";
import type { ICustomSearchSelectOption, TProject } from "@plane/types";
import { SwitcherLabel } from "@/components/common/switcher-label";
import { useWorkspaceProjects } from "@/hooks/use-projects";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";

export function useProjectSwitcherData(workspaceSlug: string, projectId: string) {
  const projectCatalogue = useWorkspaceProjects(workspaceSlug);
  const projectRows = projectCatalogue.projects;
  const joinedProjectIds = useJoinedProjectIds(workspaceSlug);
  const currentProject = projectRows?.find((project) => project.id === projectId);

  return { projectRows, joinedProjectIds, currentProject };
}

export function getProjectSwitcherOptions(
  joinedProjectIds: readonly string[],
  projectRows: readonly TProject[] | undefined
): ICustomSearchSelectOption[] {
  const projectsById = new Map(projectRows?.map((project) => [project.id, project]));
  return joinedProjectIds.flatMap((id) => {
    const project = projectsById.get(id);
    if (!project) return [];
    return [
      {
        value: id,
        query: project.name,
        content: (
          <SwitcherLabel name={project.name} logo_props={project.logo_props} LabelIcon={ProjectIcon} type="material" />
        ),
      },
    ];
  });
}

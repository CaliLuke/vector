/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCurrentWorkspace } from "@/hooks/use-workspaces";

// plane types
import type { IPartialProject } from "@plane/types";
import { Spinner } from "@plane/ui";
// components
import { PowerKProjectsMenu } from "@/components/power-k/menus/projects";
// hooks
import { useWorkspaceProjects } from "@/hooks/use-projects";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";

type Props = {
  handleSelect: (project: IPartialProject) => void;
};

export const PowerKOpenProjectMenu = function PowerKOpenProjectMenu(props: Props) {
  const { handleSelect } = props;
  // store hooks
  const { data: workspace } = useCurrentWorkspace();
  const projectCatalogue = useWorkspaceProjects(workspace?.slug);
  const projectRows = projectCatalogue.projects;
  const loader = projectCatalogue.isPending;
  const joinedProjectIds = useJoinedProjectIds(workspace?.slug);
  // derived values
  const projectsList = joinedProjectIds
    ? joinedProjectIds
        .map((id) => projectRows?.find((project) => project.id === id))
        .filter((project) => project !== undefined)
    : [];

  if (loader) return <Spinner />;

  return <PowerKProjectsMenu projects={projectsList} onSelect={handleSelect} />;
};

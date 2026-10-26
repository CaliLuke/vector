/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import type { ReactNode } from "react";

// hooks
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useJoinedProjectIds } from "@/hooks/use-joined-project-ids";
// local imports
import type { TDropdownProps } from "../types";
import { ProjectDropdownBase } from "./base";

type Props = TDropdownProps & {
  button?: ReactNode;
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  onClose?: () => void;
  renderCondition?: (projectId: string) => boolean;
  renderByDefault?: boolean;
  currentProjectId?: string;
} & (
    | {
        multiple: false;
        onChange: (val: string) => void;
        value: string | null;
      }
    | {
        multiple: true;
        onChange: (val: string[]) => void;
        value: string[];
      }
  );

export const ProjectDropdown = function ProjectDropdown(props: Props) {
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

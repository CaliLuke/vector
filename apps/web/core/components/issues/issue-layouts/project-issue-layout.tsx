/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ComponentType, FC } from "react";
import { useParams } from "@tanstack/react-router";
import { EUserPermissions } from "@plane/constants";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { ProjectIssueQuickActions } from "./quick-action-dropdowns";
import type { IQuickActionProps } from "./list/list-view-types";

interface ProjectIssueRootProps {
  QuickActions: FC<IQuickActionProps>;
  canEditPropertiesBasedOnProject?: (projectId: string) => boolean;
  viewId?: string | undefined;
}

interface ProjectIssueLayoutProps {
  Layout: ComponentType<ProjectIssueRootProps>;
  viewId?: string | undefined;
}

export function ProjectIssueLayout({ Layout, viewId }: ProjectIssueLayoutProps) {
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const { canProject } = useWorkspaceProjectRoles(workspaceSlug);
  const canEditPropertiesBasedOnProject = (projectId: string) =>
    canProject(projectId, [EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  return (
    <Layout
      QuickActions={ProjectIssueQuickActions}
      canEditPropertiesBasedOnProject={canEditPropertiesBasedOnProject}
      viewId={viewId}
    />
  );
}

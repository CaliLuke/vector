/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { EUserPermissions } from "@plane/types";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { effectiveProjectRole } from "@/helpers/permission-policy";

export function useCreationPermissions() {
  const { workspaceSlug } = useParams({ strict: false });
  const { roles, workspaceRole } = useWorkspaceProjectRoles(workspaceSlug);
  const projectsWithCreatePermissions = Object.fromEntries(
    Object.keys(roles ?? {}).flatMap((projectId) => {
      const role = effectiveProjectRole(workspaceRole, roles, projectId);
      return role !== undefined && role >= EUserPermissions.MEMBER ? [[projectId, role]] : [];
    })
  );
  return {
    projectsWithCreatePermissions,
    canPerformAnyCreateAction: Object.keys(projectsWithCreatePermissions).length > 0,
  };
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EUserPermissions } from "@plane/constants";
import { EUserWorkspaceRoles } from "@plane/types";
import type { EUserProjectRoles, IUserProjectsRole, TUserPermissions } from "@plane/types";

export type PermissionRole = EUserPermissions | EUserWorkspaceRoles | EUserProjectRoles | TUserPermissions;

const VALID_ROLES = new Set<number>([EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST]);

export function isWorkspaceRole(value: unknown): value is EUserWorkspaceRoles {
  return (
    value === EUserWorkspaceRoles.ADMIN || value === EUserWorkspaceRoles.MEMBER || value === EUserWorkspaceRoles.GUEST
  );
}

export function evaluateWorkspacePermission(
  isActive: boolean | undefined,
  workspaceRole: EUserWorkspaceRoles | null | undefined,
  allowedRoles: readonly PermissionRole[]
) {
  return isActive === true && isWorkspaceRole(workspaceRole) && allowedRoles.includes(workspaceRole);
}

/** Project membership must exist before workspace-admin elevation is applied. */
export function effectiveProjectRole(
  workspaceRole: EUserWorkspaceRoles | null | undefined,
  projectRoles: IUserProjectsRole | undefined,
  projectId: string | undefined
): TUserPermissions | undefined {
  if (!isWorkspaceRole(workspaceRole) || !projectId || !projectRoles) return undefined;
  if (!Object.hasOwn(projectRoles, projectId)) return undefined;
  const role = projectRoles[projectId];
  if (!VALID_ROLES.has(role)) return undefined;
  if (workspaceRole === EUserWorkspaceRoles.ADMIN) return EUserPermissions.ADMIN;
  return role;
}

export function evaluateProjectPermission(
  isWorkspaceActive: boolean | undefined,
  workspaceRole: EUserWorkspaceRoles | null | undefined,
  projectRoles: IUserProjectsRole | undefined,
  projectId: string | undefined,
  allowedRoles: readonly PermissionRole[]
) {
  if (isWorkspaceActive !== true || workspaceRole === null || workspaceRole === undefined) return false;
  const role = effectiveProjectRole(workspaceRole, projectRoles, projectId);
  return role !== undefined && allowedRoles.includes(role);
}

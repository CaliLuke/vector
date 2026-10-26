/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IWorkspaceMemberMe } from "@plane/types";
import { useCallback } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useIdentityGeneration } from "./use-identity-generation";
import { joinProjectOptions, leaveProjectOptions, leaveWorkspaceOptions } from "@/queries/permission-mutations";
import { useCurrentUser } from "@/hooks/use-current-user";
import {
  evaluateProjectPermission,
  evaluateWorkspacePermission,
  effectiveProjectRole,
  isWorkspaceRole,
} from "@/helpers/permission-policy";
import type { PermissionRole } from "@/helpers/permission-policy";
import { workspaceMembershipOptions, workspaceProjectRolesOptions } from "@/queries/permissions";

function useWorkspaceMembership(slug: string | undefined) {
  const { data: user } = useCurrentUser();
  return useQuery(workspaceMembershipOptions(user?.id, slug));
}

export function useWorkspacePermissions(slug: string | undefined) {
  const membershipQuery = useWorkspaceMembership(slug);
  const membership = membershipQuery.isSuccess ? membershipQuery.data : undefined;
  const workspaceRole =
    membership?.is_active === true && isWorkspaceRole(membership.role) ? membership.role : undefined;
  const can = (allowedRoles: readonly PermissionRole[]) =>
    membership !== undefined && evaluateWorkspacePermission(membership.is_active, membership.role, allowedRoles);
  return {
    membershipQuery,
    workspaceRole,
    isPending: membershipQuery.isPending,
    isError: membershipQuery.isError,
    error: membershipQuery.error,
    can,
  };
}

function activeWorkspaceMembership(membership: IWorkspaceMemberMe) {
  if (membership.is_active && isWorkspaceRole(membership.role) && membership.updated_at !== undefined)
    return { role: membership.role, updatedAt: membership.updated_at };
  return undefined;
}

export function useWorkspaceProjectRoles(slug: string | undefined) {
  const { data: user } = useCurrentUser();
  const membershipQuery = useWorkspaceMembership(slug);
  const membership = membershipQuery.isSuccess ? activeWorkspaceMembership(membershipQuery.data) : undefined;
  const rolesQuery = useQuery(workspaceProjectRolesOptions(user?.id, slug, membership?.role, membership?.updatedAt));
  const roles = membership && rolesQuery.isSuccess ? rolesQuery.data : undefined;
  const workspaceRole = membership?.role;
  const isWorkspaceActive = membership !== undefined;
  const canProject = useCallback(
    (projectId: string | undefined, allowedRoles: readonly PermissionRole[]) =>
      evaluateProjectPermission(isWorkspaceActive, workspaceRole, roles, projectId, allowedRoles),
    [isWorkspaceActive, roles, workspaceRole]
  );
  return {
    membershipQuery,
    rolesQuery,
    workspaceRole,
    roles,
    canProject,
    isPending: membershipQuery.isPending || Boolean(membership && rolesQuery.isPending),
    isError: membershipQuery.isError || Boolean(membership && rolesQuery.isError),
    error: membershipQuery.error ?? rolesQuery.error,
  };
}

export function useProjectPermissions(slug: string | undefined, projectId: string | undefined) {
  const permissions = useWorkspaceProjectRoles(slug);
  const projectRole =
    permissions.roles && projectId
      ? effectiveProjectRole(permissions.workspaceRole ?? null, permissions.roles, projectId)
      : undefined;
  const can = (allowedRoles: readonly PermissionRole[]) =>
    projectRole !== undefined && allowedRoles.includes(projectRole);
  return { ...permissions, projectRole, can };
}

export function useJoinProject(slug: string, projectId: string) {
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  useIdentityGeneration(client);
  return useMutation(joinProjectOptions(client, user?.id, slug, projectId));
}

export function useLeaveProject(slug: string, projectId: string) {
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  useIdentityGeneration(client);
  return useMutation(leaveProjectOptions(client, user?.id, slug, projectId));
}

export function useLeaveWorkspace(slug: string) {
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  useIdentityGeneration(client);
  return useMutation(leaveWorkspaceOptions(client, user?.id, slug));
}

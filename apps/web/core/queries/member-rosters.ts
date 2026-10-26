/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { EUserWorkspaceRoles } from "@plane/types";
import type { IWorkspaceMemberMe, TUserPermissions } from "@plane/types";
import { effectiveProjectRole, isWorkspaceRole } from "@/helpers/permission-policy";
import { captureIdentityScope, isIdentityScopeVerified } from "@/queries/identity";
import { permissionKeys } from "@/queries/permissions";
import { ProjectMemberService } from "@/services/project/project-member.service";
import { WorkspaceService } from "@/services/workspace.service";

type WorkspaceRosterProjection = "guest" | "full";

export type TWorkspaceRosterScope = {
  accountId: string | undefined;
  workspaceSlug: string | undefined;
  membershipRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
};

export type TProjectRosterScope = TWorkspaceRosterScope & {
  projectId: string | undefined;
  projectRole: TUserPermissions | undefined;
};

const workspaceService = new WorkspaceService();
const projectMemberService = new ProjectMemberService();

function rosterProjection(role: EUserWorkspaceRoles | undefined): WorkspaceRosterProjection | undefined {
  if (role === EUserWorkspaceRoles.GUEST) return "guest";
  if (role === EUserWorkspaceRoles.MEMBER || role === EUserWorkspaceRoles.ADMIN) return "full";
  return undefined;
}

export const memberRosterKeys = {
  workspace: (scope: TWorkspaceRosterScope) =>
    [
      "member-rosters",
      scope.accountId,
      scope.workspaceSlug,
      scope.membershipRevision,
      rosterProjection(scope.membershipRole),
      "workspace",
    ] as const,
  project: (scope: TProjectRosterScope) =>
    [
      "member-rosters",
      scope.accountId,
      scope.workspaceSlug,
      scope.membershipRevision,
      scope.membershipRole,
      scope.projectRole,
      scope.projectId,
      "project",
    ] as const,
};

function captureWorkspaceRosterScope(client: QueryClient, scope: TWorkspaceRosterScope, rosterKey: readonly unknown[]) {
  const identity = captureIdentityScope(client, scope.accountId);
  const rosterQuery = client.getQueryCache().find({ queryKey: rosterKey, exact: true });
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
  const isCurrent = () => {
    const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
    return Boolean(
      scope.accountId &&
      scope.workspaceSlug &&
      scope.membershipRevision &&
      scope.membershipRole !== undefined &&
      rosterQuery !== undefined &&
      membershipQuery !== undefined &&
      isIdentityScopeVerified(client, identity) &&
      client.getQueryCache().find({ queryKey: rosterKey, exact: true }) === rosterQuery &&
      client.getQueryCache().find({ queryKey: membershipKey, exact: true }) === membershipQuery &&
      membershipQuery?.state.status === "success" &&
      membership?.is_active === true &&
      membership.role === scope.membershipRole &&
      membership.updated_at === scope.membershipRevision
    );
  };
  return { isCurrent };
}

export function workspaceMemberRosterOptions(scope: TWorkspaceRosterScope) {
  const queryKey = memberRosterKeys.workspace(scope);
  const enabled = Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.membershipRevision &&
    isWorkspaceRole(scope.membershipRole) &&
    rosterProjection(scope.membershipRole)
  );
  return queryOptions({
    queryKey,
    queryFn: enabled
      ? async ({ client, signal }) => {
          const { isCurrent } = captureWorkspaceRosterScope(client, scope, queryKey);
          if (!isCurrent()) throw new Error("The workspace member scope changed.");
          const members = await workspaceService.fetchWorkspaceMembers(scope.workspaceSlug!, signal);
          if (!isCurrent()) throw new Error("The workspace member scope changed.");
          return members;
        }
      : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    retry: false,
  });
}

export function projectMemberRosterOptions(scope: TProjectRosterScope, canLoad = true) {
  const queryKey = memberRosterKeys.project(scope);
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.membershipRole,
    scope.membershipRevision
  );
  const enabled = Boolean(
    canLoad &&
    scope.accountId &&
    scope.workspaceSlug &&
    scope.projectId &&
    scope.membershipRevision &&
    isWorkspaceRole(scope.membershipRole) &&
    scope.projectRole !== undefined
  );
  return queryOptions({
    queryKey,
    queryFn: enabled
      ? async ({ client, signal }) => {
          const workspaceScope = captureWorkspaceRosterScope(client, scope, queryKey);
          const rolesQuery = client.getQueryCache().find({ queryKey: rolesKey, exact: true });
          const isCurrent = () => {
            const projectRoles = client.getQueryData<Record<string, TUserPermissions>>(rolesKey);
            return Boolean(
              workspaceScope.isCurrent() &&
              client.getQueryCache().find({ queryKey: rolesKey, exact: true }) === rolesQuery &&
              rolesQuery?.state.status === "success" &&
              scope.projectId &&
              scope.projectRole !== undefined &&
              effectiveProjectRole(scope.membershipRole ?? null, projectRoles, scope.projectId) === scope.projectRole
            );
          };
          if (!isCurrent()) throw new Error("The project member scope changed.");
          const memberships = await projectMemberService.fetchProjectMembers(
            scope.workspaceSlug!,
            scope.projectId!,
            signal
          );
          if (!isCurrent()) throw new Error("The project member scope changed.");
          return memberships;
        }
      : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    retry: false,
  });
}

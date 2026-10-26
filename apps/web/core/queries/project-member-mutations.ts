/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions } from "@tanstack/react-query";
import sortBy from "lodash-es/sortBy.js";
import type { Query, QueryClient } from "@tanstack/react-query";
import { EUserPermissions, EUserWorkspaceRoles } from "@plane/types";
import type {
  EUserProjectRoles,
  IProjectBulkAddFormData,
  IWorkspaceMember,
  IWorkspaceMemberMe,
  TProjectMembership,
  TUserPermissions,
} from "@plane/types";
import { effectiveProjectRole, isWorkspaceRole } from "@/helpers/permission-policy";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import { capturePermissionChanges } from "@/queries/permission-mutations";
import { permissionKeys } from "@/queries/permissions";
import { ProjectMemberService } from "@/services/project/project-member.service";
import { memberRosterKeys } from "@/queries/member-rosters";
import { projectKeys } from "@/queries/projects";
import { captureProjectRolesQuery, projectRolesRevision } from "@/queries/project-access-scope";
import { invalidateCaptured, updateCaptured } from "./member-roster-shared";

const projectMemberService = new ProjectMemberService();

function projectMemberMutationBase(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  projectId: string,
  action: string,
  memberId?: string
) {
  const identity = captureIdentityScope(client, accountId);
  const generation = getIdentitySessionGeneration(client);
  return {
    identity,
    mutationKey: ["member-rosters", accountId, generation, slug, projectId, action, ...(memberId ? [memberId] : [])],
    scope: { id: `member-rosters/${accountId}/${generation}/${slug}/${projectId}/roster` },
    retry: false,
  };
}

type ProjectMutationScope = {
  identity: ReturnType<typeof captureIdentityScope>;
  permissionChanges: ReturnType<typeof capturePermissionChanges>;
  membershipQuery: Query | undefined;
  rolesQuery: Query | undefined;
  projectRosterQueries: Query[];
  projectMetadataQueries: Query[];
  membershipRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
  projectRole: TUserPermissions | undefined;
  rolesRevision: string | undefined;
};

function captureProjectMutationScope(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  projectId: string,
  role: TUserPermissions | undefined
): ProjectMutationScope {
  const identity = captureIdentityScope(client, accountId);
  const membershipKey = permissionKeys.workspaceMembership(accountId, slug);
  const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
  const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
  const membershipRole = membership?.role ?? undefined;
  const rolesSnapshot = captureProjectRolesQuery(client, accountId, slug, membershipRole, membership?.updated_at);
  const { query: rolesQuery, roles } = rolesSnapshot;
  const catalogueScope = {
    accountId,
    sessionGeneration: getIdentitySessionGeneration(client),
    workspaceSlug: slug,
    workspaceRole: membershipRole,
    membershipRevision: membership?.updated_at,
    projectRoles: roles ? sortBy(Object.entries(roles), ([id]) => id) : undefined,
    projectId,
    projectMembershipRole: roles?.[projectId],
    projectRole: effectiveProjectRole(membershipRole, roles, projectId),
  };
  const projectRosterKey = memberRosterKeys.project({
    accountId,
    workspaceSlug: slug,
    projectId,
    membershipRole,
    membershipRevision: membership?.updated_at,
    projectRole: role,
  });
  const projectRosterQueries = client.getQueryCache().findAll({ queryKey: projectRosterKey, exact: true });
  const projectMetadataKeys = catalogueScope.projectRoles
    ? [
        projectKeys.workspace(catalogueScope, "partial"),
        projectKeys.workspace(catalogueScope, "full"),
        projectKeys.detail(catalogueScope),
      ]
    : [];
  const projectMetadataQueries = projectMetadataKeys.flatMap((queryKey) => {
    const query = client.getQueryCache().find({ queryKey, exact: true });
    return query ? [query] : [];
  });
  return {
    identity,
    permissionChanges: capturePermissionChanges(client, accountId, slug),
    membershipQuery,
    rolesQuery,
    projectRosterQueries,
    projectMetadataQueries,
    membershipRole,
    membershipRevision: membership?.updated_at,
    projectRole: role,
    rolesRevision: rolesSnapshot.revision,
  };
}

function projectMutationScopeCurrent(
  client: QueryClient,
  slug: string,
  projectId: string,
  scope: ProjectMutationScope
) {
  const membershipKey = permissionKeys.workspaceMembership(scope.identity?.userId, slug);
  const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.identity?.userId,
    slug,
    scope.membershipRole,
    scope.membershipRevision
  );
  const roles = client.getQueryData<Record<string, TUserPermissions>>(rolesKey);
  return Boolean(
    isIdentityScopeCurrent(client, scope.identity) &&
    scope.membershipQuery &&
    client.getQueryCache().find({ queryKey: membershipKey, exact: true }) === scope.membershipQuery &&
    scope.rolesQuery &&
    client.getQueryCache().find({ queryKey: rolesKey, exact: true }) === scope.rolesQuery &&
    scope.membershipQuery.state.status === "success" &&
    scope.rolesQuery.state.status === "success" &&
    membership?.is_active === true &&
    membership.role === scope.membershipRole &&
    membership.updated_at === scope.membershipRevision &&
    projectRolesRevision(roles) === scope.rolesRevision &&
    effectiveProjectRole(scope.membershipRole, roles, projectId) === scope.projectRole
  );
}

async function updateProjectMemberships(
  client: QueryClient,
  slug: string,
  projectId: string,
  scope: ProjectMutationScope,
  update: (current: TProjectMembership[] | undefined) => TProjectMembership[] | undefined
) {
  await updateCaptured<TProjectMembership[]>(
    client,
    scope.projectRosterQueries,
    () => projectMutationScopeCurrent(client, slug, projectId, scope),
    update
  );
}

async function refreshProjectMetadata(
  client: QueryClient,
  slug: string,
  projectId: string,
  scope: ProjectMutationScope
) {
  if (!projectMutationScopeCurrent(client, slug, projectId, scope)) return;
  await invalidateCaptured(client, scope.projectMetadataQueries, () =>
    projectMutationScopeCurrent(client, slug, projectId, scope)
  );
}

function canChangeProjectMemberRole(
  actorId: string | undefined,
  targetId: string,
  actorWorkspaceRole: EUserWorkspaceRoles | undefined,
  projectRole: TUserPermissions | undefined,
  targetWorkspaceRole: TUserPermissions | EUserWorkspaceRoles | undefined
) {
  return (
    (actorWorkspaceRole === EUserWorkspaceRoles.ADMIN && actorId === targetId) ||
    (projectRole === EUserPermissions.ADMIN &&
      actorId !== targetId &&
      targetWorkspaceRole !== undefined &&
      targetWorkspaceRole !== EUserPermissions.ADMIN)
  );
}

function canAssignProjectMemberRole(
  actorId: string | undefined,
  memberId: string,
  actorWorkspaceRole: EUserWorkspaceRoles | undefined,
  projectRole: TUserPermissions,
  targetProjectRole: TUserPermissions,
  requestedRole: TUserPermissions,
  targetWorkspaceRole: TUserPermissions | EUserWorkspaceRoles
) {
  const canEditTarget = canChangeProjectMemberRole(
    actorId,
    memberId,
    actorWorkspaceRole,
    projectRole,
    targetWorkspaceRole
  );
  const actorRankAllowsChange =
    actorWorkspaceRole === EUserWorkspaceRoles.ADMIN ||
    (targetProjectRole < projectRole && requestedRole < projectRole);
  const workspaceRoleAllowsAssignment =
    targetWorkspaceRole !== EUserWorkspaceRoles.GUEST || requestedRole === EUserPermissions.GUEST;
  return canEditTarget && actorRankAllowsChange && workspaceRoleAllowsAssignment;
}

function activeWorkspaceMembers(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  scope: ProjectMutationScope
) {
  const workspaceKey = memberRosterKeys.workspace({
    accountId,
    workspaceSlug: slug,
    membershipRole: scope.membershipRole,
    membershipRevision: scope.membershipRevision,
  });
  const workspaceQuery = client.getQueryCache().find({ queryKey: workspaceKey, exact: true });
  const members =
    workspaceQuery?.state.status === "success" ? client.getQueryData<IWorkspaceMember[]>(workspaceKey) : undefined;
  if (!workspaceQuery || workspaceQuery.state.status !== "success" || !members) {
    throw new Error("Workspace member list is unavailable.");
  }
  return new Map(members.filter((member) => member.is_active).map((member) => [member.member.id, member]));
}

const PROJECT_ROLE_BY_VALUE: ReadonlyMap<number, TUserPermissions> = new Map([
  [EUserPermissions.ADMIN, EUserPermissions.ADMIN],
  [EUserPermissions.MEMBER, EUserPermissions.MEMBER],
  [EUserPermissions.GUEST, EUserPermissions.GUEST],
]);

function normalizeProjectRole(
  role: TUserPermissions | EUserProjectRoles | EUserWorkspaceRoles
): TUserPermissions | undefined {
  return PROJECT_ROLE_BY_VALUE.get(role);
}

function projectRoleAllowedForWorkspace(
  role: TUserPermissions | EUserProjectRoles,
  workspaceRole: EUserWorkspaceRoles | TUserPermissions,
  projectRole: TUserPermissions
) {
  const normalizedRole = normalizeProjectRole(role);
  const normalizedWorkspaceRole = normalizeProjectRole(workspaceRole);
  return (
    normalizedRole !== undefined &&
    normalizedWorkspaceRole !== undefined &&
    normalizedRole <= projectRole &&
    ((normalizedWorkspaceRole !== EUserPermissions.GUEST && normalizedWorkspaceRole !== EUserPermissions.ADMIN) ||
      normalizedRole === normalizedWorkspaceRole)
  );
}

export function addProjectMembersOptions(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  projectId: string,
  projectRole: TUserPermissions | undefined
) {
  const identity = captureIdentityScope(client, accountId);
  const generation = getIdentitySessionGeneration(client);
  return mutationOptions({
    mutationKey: ["member-rosters", accountId, generation, slug, projectId, "add-members"],
    scope: { id: `member-rosters/${accountId}/${generation}/${slug}/${projectId}/roster` },
    retry: false,
    onMutate: () => captureProjectMutationScope(client, accountId, slug, projectId, projectRole),
    mutationFn: async (data: IProjectBulkAddFormData) => {
      const scope = captureProjectMutationScope(client, accountId, slug, projectId, projectRole);
      if (
        !isIdentityScopeVerified(client, identity) ||
        projectRole !== EUserPermissions.ADMIN ||
        !projectMutationScopeCurrent(client, slug, projectId, scope) ||
        data.members.some((member) => normalizeProjectRole(member.role) === undefined)
      )
        throw new Error("The account or project access changed.");
      const workspaceMembers = activeWorkspaceMembers(client, accountId, slug, scope);
      if (
        data.members.some((member) => {
          const workspaceMember = workspaceMembers.get(member.member_id);
          return !workspaceMember || !projectRoleAllowedForWorkspace(member.role, workspaceMember.role, projectRole);
        })
      ) {
        throw new Error("Project member roles must match the active workspace member role.");
      }
      return projectMemberService.bulkAddMembersToProject(slug, projectId, data);
    },
    onSuccess: async (members, _data, scope) => {
      await updateProjectMemberships(client, slug, projectId, scope, (current) => {
        if (!current) return current;
        const byMember = new Map(current.map((membership) => [membership.member, membership]));
        for (const membership of members) byMember.set(membership.member, membership);
        return [...byMember.values()];
      });
      await refreshProjectMetadata(client, slug, projectId, scope);
    },
  });
}

export function updateProjectMemberRoleOptions(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  projectId: string,
  memberId: string,
  projectRole: TUserPermissions | undefined
) {
  const { identity, ...mutationBase } = projectMemberMutationBase(
    client,
    accountId,
    slug,
    projectId,
    "member-role",
    memberId
  );
  return mutationOptions({
    ...mutationBase,
    onMutate: () => captureProjectMutationScope(client, accountId, slug, projectId, projectRole),
    mutationFn: async (role: TUserPermissions) => {
      const captured = captureProjectMutationScope(client, accountId, slug, projectId, projectRole);
      if (
        !isIdentityScopeVerified(client, identity) ||
        !isWorkspaceRole(role) ||
        projectRole === undefined ||
        !projectMutationScopeCurrent(client, slug, projectId, captured)
      )
        throw new Error("The account or project access changed.");
      const roster = captured.projectRosterQueries[0];
      const memberships =
        roster?.state.status === "success" ? client.getQueryData<TProjectMembership[]>(roster.queryKey) : undefined;
      const member = memberships?.find((membership) => membership.member === memberId);
      if (!member?.id) throw new Error("Project member not found.");
      const targetProjectRole = normalizeProjectRole(member.role);
      if (targetProjectRole === undefined) throw new Error("Project member role is invalid.");
      const workspaceMembers = activeWorkspaceMembers(client, accountId, slug, captured);
      const targetWorkspaceMember = workspaceMembers.get(memberId);
      if (!targetWorkspaceMember) throw new Error("Workspace member not found.");
      const targetWorkspaceRole = targetWorkspaceMember.role;
      if (
        !canAssignProjectMemberRole(
          accountId,
          memberId,
          captured.membershipRole,
          projectRole,
          targetProjectRole,
          role,
          targetWorkspaceRole
        )
      )
        throw new Error("The current role cannot change this project member.");
      return projectMemberService.updateProjectMember(slug, projectId, member.id, { role });
    },
    onSuccess: async (updated, _role, scope) => {
      await updateProjectMemberships(client, slug, projectId, scope, (current) =>
        current?.map((membership) => (membership.member === updated.member ? updated : membership))
      );
      if (accountId === memberId) await scope.permissionChanges.resetProjectRoles();
    },
  });
}

export function removeProjectMemberOptions(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  projectId: string,
  memberId: string,
  projectRole: TUserPermissions | undefined
) {
  const { identity, ...mutationBase } = projectMemberMutationBase(
    client,
    accountId,
    slug,
    projectId,
    "remove-member",
    memberId
  );
  return mutationOptions({
    ...mutationBase,
    onMutate: () => captureProjectMutationScope(client, accountId, slug, projectId, projectRole),
    mutationFn: async () => {
      const scope = captureProjectMutationScope(client, accountId, slug, projectId, projectRole);
      if (
        !isIdentityScopeVerified(client, identity) ||
        projectRole !== EUserPermissions.ADMIN ||
        !projectMutationScopeCurrent(client, slug, projectId, scope)
      )
        throw new Error("The account or project access changed.");
      if (accountId === memberId) throw new Error("Use the project leave action for your own membership.");
      const roster = scope.projectRosterQueries[0];
      if (roster?.state.status !== "success") throw new Error("Project member list is unavailable.");
      const memberships = client.getQueryData<TProjectMembership[]>(roster.queryKey);
      const member = memberships?.find((membership) => membership.member === memberId);
      if (!member?.id) throw new Error("Project member not found.");
      await projectMemberService.deleteProjectMember(slug, projectId, member.id);
    },
    onSuccess: async (_data, _variables, scope) => {
      await updateProjectMemberships(client, slug, projectId, scope, (current) =>
        current?.filter((membership) => membership.member !== memberId)
      );
      await refreshProjectMetadata(client, slug, projectId, scope);
    },
  });
}

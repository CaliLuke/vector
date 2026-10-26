/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { EUserPermissions, EUserWorkspaceRoles } from "@plane/types";
import type { IWorkspaceBulkInviteFormData, IWorkspaceMember, IWorkspaceMemberMe } from "@plane/types";
import { isWorkspaceRole } from "@/helpers/permission-policy";
import { captureIdentityScope, getIdentitySessionGeneration, isIdentityScopeVerified } from "@/queries/identity";
import { permissionKeys } from "@/queries/permissions";
import { WorkspaceService } from "@/services/workspace.service";
import {
  captureWorkspaceInvitationQueries,
  captureWorkspaceMemberProjectCascade,
  captureWorkspaceRosterQueries,
  captureWorkspaceScope,
  cancelAndInvalidateCaptured,
  invalidateCaptured,
  updateCaptured,
} from "./member-roster-shared";
import { memberRosterKeys } from "@/queries/member-rosters";

const workspaceService = new WorkspaceService();

function captureWorkspaceMemberMutationScope(client: QueryClient, accountId: string | undefined, slug: string) {
  const scope = captureWorkspaceScope(client, accountId, slug);
  return {
    scope,
    rosters: captureWorkspaceRosterQueries(client, accountId, slug, scope.membership),
    projectCascade: captureWorkspaceMemberProjectCascade(client, accountId, slug, scope.membership),
  };
}

export function inviteWorkspaceOptions(client: QueryClient, accountId: string | undefined, slug: string) {
  const identity = captureIdentityScope(client, accountId);
  const generation = getIdentitySessionGeneration(client);
  return mutationOptions({
    mutationKey: ["member-rosters", accountId, generation, slug, "invite-workspace"],
    scope: { id: `member-rosters/${accountId}/${generation}/${slug}/invitations` },
    retry: false,
    onMutate: () => {
      const scope = captureWorkspaceScope(client, accountId, slug);
      return { scope, invitations: captureWorkspaceInvitationQueries(client, accountId, slug, scope.membership) };
    },
    mutationFn: async (data: IWorkspaceBulkInviteFormData) => {
      if (!slug || !isIdentityScopeVerified(client, identity)) throw new Error("The account or workspace changed.");
      const membershipKey = permissionKeys.workspaceMembership(accountId, slug);
      const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
      const actorRole = membership?.role;
      const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
      if (
        membershipQuery?.state.status !== "success" ||
        membership?.is_active !== true ||
        (actorRole !== EUserWorkspaceRoles.MEMBER && actorRole !== EUserWorkspaceRoles.ADMIN) ||
        data.emails.some((invitation) => !isWorkspaceRole(invitation.role) || invitation.role > actorRole)
      )
        throw new Error("The current workspace role cannot send these invitations.");
      return workspaceService.inviteWorkspace(slug, data);
    },
    onSuccess: async (_result, _data, captured) => {
      await invalidateCaptured(client, captured.invitations, captured.scope.isCurrent);
    },
  });
}

export function updateWorkspaceMemberRoleOptions(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  memberId: string
) {
  const identity = captureIdentityScope(client, accountId);
  const generation = getIdentitySessionGeneration(client);
  return mutationOptions({
    mutationKey: ["member-rosters", accountId, generation, slug, "workspace-role", memberId],
    scope: { id: `member-rosters/${accountId}/${generation}/${slug}/workspace-member/${memberId}` },
    retry: false,
    onMutate: () => captureWorkspaceMemberMutationScope(client, accountId, slug),
    mutationFn: async (role: EUserPermissions) => {
      if (!slug || !memberId || !isIdentityScopeVerified(client, identity))
        throw new Error("The account or workspace changed.");
      if (accountId === memberId) throw new Error("You cannot change your own workspace role.");
      const membershipKey = permissionKeys.workspaceMembership(accountId, slug);
      const actorMembership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
      const actorQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
      if (
        actorQuery?.state.status !== "success" ||
        actorMembership?.is_active !== true ||
        actorMembership.role !== EUserWorkspaceRoles.ADMIN ||
        !isWorkspaceRole(role)
      )
        throw new Error("Only a workspace admin can change member roles.");
      const rosterKey = memberRosterKeys.workspace({
        accountId,
        workspaceSlug: slug,
        membershipRole: actorMembership.role,
        membershipRevision: actorMembership.updated_at,
      });
      const roster = client.getQueryCache().find({ queryKey: rosterKey, exact: true });
      const memberships =
        roster?.state.status === "success" ? client.getQueryData<IWorkspaceMember[]>(roster.queryKey) : undefined;
      const member = memberships?.find((entry) => entry.member.id === memberId);
      if (!member) throw new Error("Workspace member not found.");
      return workspaceService.updateWorkspaceMember(slug, member.id, { role });
    },
    onSuccess: async (updated, role, captured) => {
      await updateCaptured<IWorkspaceMember[]>(client, captured.rosters, captured.scope.isCurrent, (members) =>
        members?.map((member) => (member.id === updated.id ? updated : member))
      );
      if (role === EUserPermissions.GUEST) {
        await cancelAndInvalidateCaptured(client, captured.projectCascade.queries, captured.projectCascade.isCurrent);
      }
    },
  });
}

export function removeWorkspaceMemberOptions(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  membershipId: string,
  memberUserId: string
) {
  const identity = captureIdentityScope(client, accountId);
  const generation = getIdentitySessionGeneration(client);
  return mutationOptions({
    mutationKey: ["member-rosters", accountId, generation, slug, "remove-workspace-member", memberUserId],
    scope: { id: `member-rosters/${accountId}/${generation}/${slug}/workspace-member/${memberUserId}` },
    retry: false,
    onMutate: () => captureWorkspaceMemberMutationScope(client, accountId, slug),
    mutationFn: async () => {
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      if (accountId === memberUserId) throw new Error("Use the workspace leave action for your own membership.");
      const membershipKey = permissionKeys.workspaceMembership(accountId, slug);
      const actorMembership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
      const actorQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
      if (
        actorQuery?.state.status !== "success" ||
        actorMembership?.is_active !== true ||
        actorMembership.role !== EUserWorkspaceRoles.ADMIN
      )
        throw new Error("Only a workspace admin can remove another member.");
      await workspaceService.deleteWorkspaceMember(slug, membershipId);
    },
    onSuccess: async (_data, _variables, captured) => {
      await updateCaptured<IWorkspaceMember[]>(client, captured.rosters, captured.scope.isCurrent, (members) =>
        members?.map((member) => (member.id === membershipId ? { ...member, is_active: false } : member))
      );
      await cancelAndInvalidateCaptured(client, captured.projectCascade.queries, captured.projectCascade.isCurrent);
    },
  });
}

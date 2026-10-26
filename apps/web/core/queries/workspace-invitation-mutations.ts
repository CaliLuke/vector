/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { EUserWorkspaceRoles } from "@plane/types";
import type { IWorkspaceMemberInvitation, IWorkspaceMemberMe } from "@plane/types";
import { isWorkspaceRole } from "@/helpers/permission-policy";
import { captureIdentityScope, getIdentitySessionGeneration, isIdentityScopeVerified } from "@/queries/identity";
import { permissionKeys } from "@/queries/permissions";
import { WorkspaceService } from "@/services/workspace.service";
import {
  captureWorkspaceInvitationQueries,
  captureWorkspaceScope,
  exactQueryCurrent,
  updateCaptured,
} from "./member-roster-shared";
import { workspaceInvitationKeys, type WorkspaceInvitationScope } from "./member-roster-shared";

const workspaceService = new WorkspaceService();

export function workspaceInvitationListOptions(scope: WorkspaceInvitationScope) {
  const queryKey = workspaceInvitationKeys.list(scope);
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  const enabled = Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.membershipRevision &&
    (scope.membershipRole === EUserWorkspaceRoles.MEMBER || scope.membershipRole === EUserWorkspaceRoles.ADMIN)
  );
  return queryOptions<IWorkspaceMemberInvitation[]>({
    queryKey,
    queryFn: enabled
      ? async ({ client, signal }) => {
          const captured = captureWorkspaceScope(client, scope.accountId, scope.workspaceSlug);
          const query = client.getQueryCache().find({ queryKey, exact: true });
          const isCurrent = () => {
            const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
            const membershipRole = membership?.role;
            return Boolean(
              captured.isCurrent() &&
              query &&
              exactQueryCurrent(client, query) &&
              membershipRole === scope.membershipRole &&
              membership?.updated_at === scope.membershipRevision
            );
          };
          if (!isCurrent()) throw new Error("The workspace invitation scope changed.");
          const data = await workspaceService.workspaceInvitations(scope.workspaceSlug!, signal);
          if (!isCurrent()) throw new Error("The workspace invitation scope changed.");
          return data;
        }
      : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    retry: false,
  });
}
export function updateWorkspaceInvitationOptions(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  invitationId: string
) {
  const identity = captureIdentityScope(client, accountId);
  const generation = getIdentitySessionGeneration(client);
  return mutationOptions({
    mutationKey: ["member-rosters", accountId, generation, slug, "invitation-role", invitationId],
    scope: { id: `member-rosters/${accountId}/${generation}/${slug}/invitation/${invitationId}` },
    retry: false,
    onMutate: () => {
      const scope = captureWorkspaceScope(client, accountId, slug);
      return { scope, invitations: captureWorkspaceInvitationQueries(client, accountId, slug, scope.membership) };
    },
    mutationFn: async (change: Pick<IWorkspaceMemberInvitation, "role">) => {
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      const membershipKey = permissionKeys.workspaceMembership(accountId, slug);
      const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
      const actorRole = membership?.role;
      const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
      if (
        membershipQuery?.state.status !== "success" ||
        membership?.is_active !== true ||
        (actorRole !== EUserWorkspaceRoles.MEMBER && actorRole !== EUserWorkspaceRoles.ADMIN) ||
        !isWorkspaceRole(change.role) ||
        change.role > actorRole
      )
        throw new Error("The current workspace role cannot set this invitation role.");
      return workspaceService.updateWorkspaceInvitation(slug, invitationId, change);
    },
    onSuccess: async (updated, _change, captured) => {
      await updateCaptured<IWorkspaceMemberInvitation[]>(
        client,
        captured.invitations,
        captured.scope.isCurrent,
        (rows) => rows?.map((row) => (row.id === invitationId ? updated : row))
      );
    },
  });
}

export function deleteWorkspaceInvitationOptions(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  invitationId: string
) {
  const identity = captureIdentityScope(client, accountId);
  const generation = getIdentitySessionGeneration(client);
  return mutationOptions({
    mutationKey: ["member-rosters", accountId, generation, slug, "delete-invitation", invitationId],
    scope: { id: `member-rosters/${accountId}/${generation}/${slug}/invitation/${invitationId}` },
    retry: false,
    onMutate: () => {
      const scope = captureWorkspaceScope(client, accountId, slug);
      return { scope, invitations: captureWorkspaceInvitationQueries(client, accountId, slug, scope.membership) };
    },
    mutationFn: async () => {
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      const membershipKey = permissionKeys.workspaceMembership(accountId, slug);
      const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
      const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
      if (
        membershipQuery?.state.status !== "success" ||
        membership?.is_active !== true ||
        membership.role !== EUserWorkspaceRoles.ADMIN
      )
        throw new Error("Only a workspace admin can remove an invitation.");
      await workspaceService.deleteWorkspaceInvitations(slug, invitationId);
    },
    onSuccess: async (_data, _variables, captured) => {
      await updateCaptured<IWorkspaceMemberInvitation[]>(
        client,
        captured.invitations,
        captured.scope.isCurrent,
        (rows) => rows?.filter((row) => row.id !== invitationId)
      );
    },
  });
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { MutationObserver, mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import { v4 as uuidv4 } from "uuid";
import orderBy from "lodash-es/orderBy.js";
import type { QueryClient } from "@tanstack/react-query";
import { EUserWorkspaceRoles } from "@plane/types";
import type { IFavorite, IWorkspaceMemberMe, IUserProjectsRole } from "@plane/types";
import {
  captureIdentityScope,
  getCurrentUserId,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import { workspaceProjectRolesOptions, permissionKeys } from "@/queries/permissions";
import { FavoriteService } from "@/services/favorite";

const favoriteService = new FavoriteService();

export type TFavoriteScope = {
  accountId: string | undefined;
  workspaceSlug: string | undefined;
  workspaceRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
  projectRolesRevision: string | undefined;
};

export function favoriteProjectRolesRevision(roles: IUserProjectsRole | undefined) {
  if (!roles) return undefined;
  return JSON.stringify(orderBy(Object.entries(roles), ([projectId]) => projectId));
}

export const favoriteKeys = {
  workspace: (accountId: string | undefined, workspaceSlug: string | undefined) =>
    ["workspace-favorites", accountId, workspaceSlug] as const,
  all: (scope: TFavoriteScope) =>
    [
      "workspace-favorites",
      scope.accountId,
      scope.workspaceSlug,
      scope.workspaceRole,
      scope.membershipRevision,
      scope.projectRolesRevision,
      "all",
    ] as const,
  writes: (scope: TFavoriteScope, generation: number) =>
    [
      "workspace-favorites",
      scope.accountId,
      scope.workspaceSlug,
      scope.workspaceRole,
      scope.membershipRevision,
      scope.projectRolesRevision,
      generation,
      "write",
    ] as const,
};

function canUseFavorites(
  role: EUserWorkspaceRoles | null | undefined
): role is EUserWorkspaceRoles.ADMIN | EUserWorkspaceRoles.MEMBER {
  return role === EUserWorkspaceRoles.ADMIN || role === EUserWorkspaceRoles.MEMBER;
}

function exactQueryCurrent(client: QueryClient, key: readonly unknown[], query: object | undefined) {
  return query !== undefined && client.getQueryCache().find({ queryKey: key, exact: true }) === query;
}

type TCapturedFavoriteScope = {
  queryKey: ReturnType<typeof favoriteKeys.all>;
  isIdentityAndQueryCurrent: () => boolean;
  isAuthorizedCurrent: () => boolean;
  isAuthorizedVerified: () => boolean;
};

function captureFavoriteScope(client: QueryClient, scope: TFavoriteScope): TCapturedFavoriteScope {
  const queryKey = favoriteKeys.all(scope);
  const identity = captureIdentityScope(client, scope.accountId);
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  const projectRolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.workspaceRole,
    scope.membershipRevision
  );
  const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
  const projectRolesQuery = client.getQueryCache().find({ queryKey: projectRolesKey, exact: true });
  const favoriteQuery = client.getQueryCache().find({ queryKey, exact: true });

  const isIdentityAndQueryCurrent = () =>
    identity !== undefined &&
    isIdentityScopeCurrent(client, identity) &&
    exactQueryCurrent(client, queryKey, favoriteQuery);

  const isWorkspaceMembershipCurrent = () => {
    const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
    return (
      exactQueryCurrent(client, membershipKey, membershipQuery) &&
      membershipQuery?.state.status === "success" &&
      membership?.is_active === true &&
      membership.role === scope.workspaceRole &&
      membership.updated_at === scope.membershipRevision
    );
  };
  const isProjectRolesCurrent = () =>
    exactQueryCurrent(client, projectRolesKey, projectRolesQuery) &&
    projectRolesQuery?.state.status === "success" &&
    favoriteProjectRolesRevision(client.getQueryData<IUserProjectsRole>(projectRolesKey)) ===
      scope.projectRolesRevision;
  const isAuthorizedCurrent = () => {
    return (
      scope.accountId !== undefined &&
      scope.workspaceSlug !== undefined &&
      scope.membershipRevision !== undefined &&
      scope.projectRolesRevision !== undefined &&
      canUseFavorites(scope.workspaceRole) &&
      isIdentityAndQueryCurrent() &&
      isWorkspaceMembershipCurrent() &&
      isProjectRolesCurrent()
    );
  };
  const isAuthorizedVerified = () => isAuthorizedCurrent() && isIdentityScopeVerified(client, identity);

  return { queryKey, isIdentityAndQueryCurrent, isAuthorizedCurrent, isAuthorizedVerified };
}

export function removeFavoriteSubtree(favorites: IFavorite[], favoriteId: string) {
  const removedIds = new Set([favoriteId]);
  let added = true;
  while (added) {
    added = false;
    for (const favorite of favorites) {
      if (favorite.parent && removedIds.has(favorite.parent) && !removedIds.has(favorite.id)) {
        removedIds.add(favorite.id);
        added = true;
      }
    }
  }
  return favorites.filter((favorite) => !removedIds.has(favorite.id));
}

export function captureWorkspaceFavoritesRefresh(client: QueryClient, workspaceSlug: string) {
  const accountId = getCurrentUserId(client);
  const identity = captureIdentityScope(client, accountId);
  if (!accountId || !identity) return async () => undefined;
  const queryKey = favoriteKeys.workspace(accountId, workspaceSlug);
  return async () => {
    if (!isIdentityScopeCurrent(client, identity)) return;
    try {
      await client.cancelQueries({ queryKey });
      if (!isIdentityScopeCurrent(client, identity)) return;
      await client.invalidateQueries({ queryKey, refetchType: "active" });
    } catch {
      // A completed entity deletion stays successful when the follow-up read fails.
    }
  };
}

export async function addProjectFavoriteAfterAccessRefresh(
  client: QueryClient,
  workspaceSlug: string,
  projectId: string,
  projectName: string
) {
  const accountId = getCurrentUserId(client);
  const identity = captureIdentityScope(client, accountId);
  if (!accountId || !identity) throw new Error("The account session is unavailable.");
  const membershipKey = permissionKeys.workspaceMembership(accountId, workspaceSlug);
  const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
  const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
  if (
    !membershipQuery ||
    membershipQuery.state.status !== "success" ||
    membership?.is_active !== true ||
    !canUseFavorites(membership.role) ||
    !membership.updated_at
  )
    throw new Error("Workspace access is unavailable.");
  const isMembershipCurrent = () => {
    const currentMembership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
    return (
      isIdentityScopeVerified(client, identity) &&
      client.getQueryCache().find({ queryKey: membershipKey, exact: true }) === membershipQuery &&
      membershipQuery.state.status === "success" &&
      currentMembership?.is_active === true &&
      currentMembership.role === membership.role &&
      currentMembership.updated_at === membership.updated_at
    );
  };
  if (!isMembershipCurrent()) throw new Error("Workspace access is unavailable.");

  const rolesKey = permissionKeys.workspaceProjectRoles(
    accountId,
    workspaceSlug,
    membership.role,
    membership.updated_at
  );
  const roles = await client.fetchQuery({
    ...workspaceProjectRolesOptions(accountId, workspaceSlug, membership.role, membership.updated_at),
    staleTime: 0,
  });
  const rolesQuery = client.getQueryCache().find({ queryKey: rolesKey, exact: true });
  if (!isMembershipCurrent() || rolesQuery?.state.status !== "success" || !Object.hasOwn(roles, projectId))
    throw new Error("Project access is unavailable.");

  const scope: TFavoriteScope = {
    accountId,
    workspaceSlug,
    workspaceRole: membership.role,
    membershipRevision: membership.updated_at,
    projectRolesRevision: favoriteProjectRolesRevision(roles),
  };
  await client.fetchQuery(workspaceFavoritesOptions(scope));
  if (!isMembershipCurrent()) throw new Error("Workspace access changed.");
  const observer = new MutationObserver(client, favoriteMutationOptions(client, scope));
  return observer.mutate({
    type: "create",
    clientId: uuidv4(),
    data: {
      entity_type: "project",
      entity_identifier: projectId,
      project_id: projectId,
      entity_data: { name: projectName },
    },
  });
}

export function workspaceFavoritesOptions(scope: TFavoriteScope) {
  const queryKey = favoriteKeys.all(scope);
  const enabled = Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.membershipRevision &&
    scope.projectRolesRevision !== undefined &&
    canUseFavorites(scope.workspaceRole)
  );
  return queryOptions({
    queryKey,
    queryFn: enabled
      ? async ({ client, signal }) => {
          const captured = captureFavoriteScope(client, scope);
          const workspaceSlug = scope.workspaceSlug;
          if (!captured.isAuthorizedVerified() || !workspaceSlug)
            throw new Error("The workspace favorite scope changed.");
          const favorites = await favoriteService.getFavorites(workspaceSlug, signal);
          if (!captured.isAuthorizedVerified()) throw new Error("The workspace favorite scope changed.");
          return favorites;
        }
      : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    retry: false,
  });
}

export type TFavoriteWrite =
  | { type: "create"; clientId: string; data: Partial<IFavorite> }
  | { type: "update"; favoriteId: string; data: Partial<IFavorite> }
  | { type: "delete"; favoriteId: string };

type TFavoriteMutationScope = {
  queryKey: ReturnType<typeof favoriteKeys.all>;
  writeKey: ReturnType<typeof favoriteKeys.writes>;
  refresh: () => Promise<void>;
  isIdentityAndQueryCurrent: () => boolean;
  isAuthorizedCurrent: () => boolean;
  isAuthorizedVerified: () => boolean;
};

function captureFavoriteMutationScope(
  client: QueryClient,
  scope: TFavoriteScope,
  generation: number
): TFavoriteMutationScope {
  const captured = captureFavoriteScope(client, scope);
  const queryKey = captured.queryKey;
  const writeKey = favoriteKeys.writes(scope, generation);
  return {
    queryKey,
    writeKey,
    refresh: scope.workspaceSlug
      ? captureWorkspaceFavoritesRefresh(client, scope.workspaceSlug)
      : async () => undefined,
    isIdentityAndQueryCurrent: captured.isIdentityAndQueryCurrent,
    isAuthorizedCurrent: captured.isAuthorizedCurrent,
    isAuthorizedVerified: captured.isAuthorizedVerified,
  };
}

export function favoriteMutationOptions(client: QueryClient, scope: TFavoriteScope) {
  const generation = getIdentitySessionGeneration(client);
  const captured = captureFavoriteMutationScope(client, scope, generation);
  return mutationOptions({
    mutationKey: captured.writeKey,
    scope: { id: `workspace-favorites/${scope.accountId}/${generation}/${scope.workspaceSlug}` },
    retry: false,
    mutationFn: async (write: TFavoriteWrite) => {
      if (!captured.isAuthorizedVerified()) throw new Error("The workspace favorite scope changed.");
      const favorites = client.getQueryData<IFavorite[]>(captured.queryKey);
      if (client.getQueryState(captured.queryKey)?.status !== "success" || favorites === undefined)
        throw new Error("Workspace favorites are not loaded.");

      if (write.type === "create") {
        if (write.data.project_id) {
          const rolesKey = permissionKeys.workspaceProjectRoles(
            scope.accountId,
            scope.workspaceSlug,
            scope.workspaceRole,
            scope.membershipRevision
          );
          const roles = client.getQueryData<IUserProjectsRole>(rolesKey);
          if (!roles || !Object.hasOwn(roles, write.data.project_id)) throw new Error("Project access is unavailable.");
        }
        const entityIdentifier = write.data.entity_identifier;
        const existing = entityIdentifier
          ? favorites.find(
              (favorite) =>
                favorite.entity_type === write.data.entity_type && favorite.entity_identifier === entityIdentifier
            )
          : undefined;
        return existing ?? favoriteService.addFavorite(scope.workspaceSlug!, write.data);
      }
      if (!favorites.some((favorite) => favorite.id === write.favoriteId))
        throw new Error("The favorite is no longer available.");
      if (write.type === "update")
        return favoriteService.updateFavorite(scope.workspaceSlug!, write.favoriteId, write.data);
      await favoriteService.deleteFavorite(scope.workspaceSlug!, write.favoriteId);
      return undefined;
    },
    onSuccess: async (result, write) => {
      const canCommit = captured.isAuthorizedCurrent();
      if (canCommit && write.type === "create" && result) {
        client.setQueryData<IFavorite[]>(captured.queryKey, (favorites) => {
          if (!favorites) return favorites;
          return favorites.some((favorite) => favorite.id === result.id)
            ? favorites.map((favorite) => (favorite.id === result.id ? result : favorite))
            : [...favorites, result];
        });
      } else if (canCommit && write.type === "update" && result) {
        client.setQueryData<IFavorite[]>(captured.queryKey, (favorites) =>
          favorites?.map((favorite) => (favorite.id === result.id ? result : favorite))
        );
      } else if (canCommit && write.type === "delete") {
        client.setQueryData<IFavorite[]>(captured.queryKey, (favorites) =>
          favorites ? removeFavoriteSubtree(favorites, write.favoriteId) : favorites
        );
      }
      if (captured.isIdentityAndQueryCurrent()) await captured.refresh();
    },
  });
}

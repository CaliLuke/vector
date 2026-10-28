/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useCallback, useMemo } from "react";
import { useMutation, useMutationState, useQueryClient } from "@tanstack/react-query";
import { EUserWorkspaceRoles } from "@plane/types";
import type { IFavorite } from "@plane/types";
import orderBy from "lodash-es/orderBy.js";
import { v4 as uuidv4 } from "uuid";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { getIdentitySessionGeneration } from "@/queries/identity";
import {
  favoriteProjectRolesRevision,
  favoriteKeys,
  addProjectFavoriteAfterAccessRefresh,
  favoriteMutationOptions,
  removeFavoriteSubtree,
  workspaceFavoritesOptions,
} from "@/queries/favorites";
import type { TFavoriteScope, TFavoriteWrite } from "@/queries/favorites";

type TPendingFavoriteWrite = { write: TFavoriteWrite; submittedAt: number; mutationId: number };

function isFavoriteWrite(value: unknown): value is TFavoriteWrite {
  if (typeof value !== "object" || value === null || !("type" in value)) return false;
  return value.type === "create" || value.type === "update" || value.type === "delete";
}

function applyPendingWrites(favorites: IFavorite[], pending: TPendingFavoriteWrite[]): IFavorite[] {
  let result = [...favorites];
  for (const { write } of orderBy(pending, [(entry) => entry.submittedAt, (entry) => entry.mutationId])) {
    if (write.type === "create") {
      const identifier = write.data.entity_identifier;
      if (
        identifier &&
        result.some(
          (favorite) => favorite.entity_type === write.data.entity_type && favorite.entity_identifier === identifier
        )
      )
        continue;
      const name = write.data.name ?? write.data.entity_data?.name ?? "";
      result.push({
        ...write.data,
        id: write.clientId,
        entity_type: write.data.entity_type ?? "",
        entity_identifier: identifier,
        entity_data: write.data.entity_data ?? { name },
        name,
        is_folder: write.data.entity_type === "folder",
        sort_order: write.data.sort_order ?? 0,
        parent: write.data.parent ?? null,
        children: [],
        project_id: write.data.project_id ?? null,
        sequence: write.data.sequence ?? 0,
        workspace_id: "",
      });
    } else if (write.type === "update") {
      result = result.map((favorite) =>
        favorite.id === write.favoriteId ? { ...favorite, ...write.data, id: favorite.id } : favorite
      );
    } else {
      result = removeFavoriteSubtree(result, write.favoriteId);
    }
  }
  return result;
}

function groupFavorites(favorites: IFavorite[]) {
  const groups = Object.fromEntries(
    favorites.map((favorite) => [favorite.id, { ...favorite, children: [] }])
  ) as Record<string, IFavorite>;
  for (const favorite of favorites) {
    const parent = favorite.parent ? groups[favorite.parent] : undefined;
    if (parent) parent.children.push(groups[favorite.id]);
  }
  return groups;
}

export function useFavorites(workspaceSlug: string | undefined) {
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  const permissions = useWorkspaceProjectRoles(workspaceSlug);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const rolesReady = permissions.rolesQuery.isSuccess;
  const scope: TFavoriteScope = {
    accountId: user?.id,
    workspaceSlug,
    workspaceRole: permissions.workspaceRole,
    membershipRevision: membership?.updated_at,
    projectRolesRevision: rolesReady ? favoriteProjectRolesRevision(permissions.roles) : undefined,
  };
  const query = useRecordCollection(workspaceFavoritesOptions(scope));
  const canAccess =
    membership?.is_active === true &&
    (permissions.workspaceRole === EUserWorkspaceRoles.ADMIN ||
      permissions.workspaceRole === EUserWorkspaceRoles.MEMBER);
  const canRead = canAccess && rolesReady;
  const writeOptions = favoriteMutationOptions(client, scope);
  const mutation = useMutation(writeOptions);
  const writeKey = favoriteKeys.writes(scope, getIdentitySessionGeneration(client));
  const pending = useMutationState({
    filters: { mutationKey: writeKey, exact: true, status: "pending" },
    select: (entry): TPendingFavoriteWrite | undefined => {
      const write = entry.state.variables;
      return isFavoriteWrite(write)
        ? { write, submittedAt: entry.state.submittedAt, mutationId: entry.mutationId }
        : undefined;
    },
  });
  const favorites = useMemo(() => {
    if (!canRead || !query.isSuccess) return undefined;
    const pendingWrites = pending.filter((entry): entry is TPendingFavoriteWrite => entry !== undefined);
    return applyPendingWrites(query.data, pendingWrites).filter(
      (favorite) => favorite.project_id === null || permissions.roles?.[favorite.project_id] !== undefined
    );
  }, [canRead, pending, permissions.roles, query.data, query.isSuccess]);
  const groupedFavorites = useMemo(() => (favorites ? groupFavorites(favorites) : undefined), [favorites]);
  const favoritesByEntity = useMemo(() => {
    if (!favorites) return undefined;
    return new Map(
      favorites
        .filter((favorite) => favorite.entity_identifier)
        .map((favorite) => [`${favorite.entity_type}:${favorite.entity_identifier}`, favorite])
    );
  }, [favorites]);
  const existingFolders = useMemo(() => favorites?.map((favorite) => favorite.name) ?? [], [favorites]);
  const retry = useCallback(async () => {
    if (permissions.membershipQuery.isError) {
      await permissions.membershipQuery.refetch();
      return;
    }
    if (permissions.rolesQuery.isError) {
      await permissions.rolesQuery.refetch();
      return;
    }
    if (query.isError) await query.refetch();
  }, [permissions.membershipQuery, permissions.rolesQuery, query]);
  const addProjectFavorite = useCallback(
    (projectId: string, projectName: string) => {
      if (!workspaceSlug) return Promise.reject(new Error("Workspace scope is unavailable."));
      return addProjectFavoriteAfterAccessRefresh(client, workspaceSlug, projectId, projectName);
    },
    [client, workspaceSlug]
  );
  const addFavorite = useCallback(
    (data: Partial<IFavorite>) =>
      mutation.mutateAsync({
        type: "create",
        clientId: uuidv4(),
        data: { ...data, parent: null, is_folder: data.entity_type === "folder" },
      }),
    [mutation]
  );
  const updateFavorite = useCallback(
    (favoriteId: string, data: Partial<IFavorite>) => mutation.mutateAsync({ type: "update", favoriteId, data }),
    [mutation]
  );
  const deleteFavorite = useCallback(
    (favoriteId: string) => mutation.mutateAsync({ type: "delete", favoriteId }),
    [mutation]
  );
  const removeFavoriteEntity = useCallback(
    (entityType: string, entityId: string) => {
      const favorite = favoritesByEntity?.get(`${entityType}:${entityId}`);
      return favorite ? deleteFavorite(favorite.id) : Promise.resolve(undefined);
    },
    [deleteFavorite, favoritesByEntity]
  );
  const moveFavoriteToFolder = useCallback(
    (favoriteId: string, parentId: string) => updateFavorite(favoriteId, { parent: parentId }),
    [updateFavorite]
  );
  const removeFromFavoriteFolder = useCallback(
    (favoriteId: string) => updateFavorite(favoriteId, { parent: null }),
    [updateFavorite]
  );
  const reOrderFavorite = useCallback(
    (favoriteId: string, destinationId: string, edge: string | undefined) => {
      const currentFavorites = favorites ?? [];
      const source = currentFavorites.find((favorite) => favorite.id === favoriteId);
      const destination = currentFavorites.find((favorite) => favorite.id === destinationId);
      if (!source || !destination || favoriteId === destinationId) return Promise.resolve(undefined);

      const siblings = orderBy(
        currentFavorites.filter((favorite) => favorite.id !== favoriteId && favorite.parent === destination.parent),
        "sequence",
        "desc"
      );
      const destinationIndex = siblings.findIndex((favorite) => favorite.id === destinationId);
      const destinationSequence = destination.sequence;
      let sequence = 10_000;

      if (destinationIndex >= 0 && typeof destinationSequence === "number") {
        if (edge === "reorder-above") {
          const previousSequence = siblings[destinationIndex - 1]?.sequence;
          sequence =
            typeof previousSequence === "number"
              ? (destinationSequence + previousSequence) / 2
              : destinationSequence + 10_000;
        } else if (edge === "reorder-below") {
          const nextSequence = siblings[destinationIndex + 1]?.sequence;
          sequence =
            typeof nextSequence === "number" ? (destinationSequence + nextSequence) / 2 : destinationSequence - 10_000;
        }
      }
      return updateFavorite(favoriteId, { sequence });
    },
    [favorites, updateFavorite]
  );
  const toggleFavorite = useCallback(
    (entityType: string, entityId: string, projectId?: string | null) => {
      const favorite = favoritesByEntity?.get(`${entityType}:${entityId}`);
      return favorite
        ? deleteFavorite(favorite.id)
        : addFavorite({ entity_type: entityType, entity_identifier: entityId, project_id: projectId ?? null });
    },
    [addFavorite, deleteFavorite, favoritesByEntity]
  );

  return {
    query,
    favorites,
    groupedFavorites,
    favoritesByEntity,
    existingFolders,
    canAccess,
    isLoading: permissions.isPending || (canRead && query.isLoading),
    isError: permissions.isError || (canRead && query.isError),
    error: permissions.error ?? query.error,
    retry,
    addFavorite,
    addProjectFavorite,
    updateFavorite,
    deleteFavorite,
    removeFavoriteEntity,
    moveFavoriteToFolder,
    removeFromFavoriteFolder,
    reOrderFavorite,
    toggleFavorite,
    mutation,
  };
}

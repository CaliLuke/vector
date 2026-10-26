/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { IWorkspaceSidebarNavigationItem, IWorkspaceUserPropertiesResponse } from "@plane/types";
import { WorkspaceService } from "@/services/workspace.service";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "./identity";

export type SidebarPreferencePatch = { key: string } & Partial<
  Pick<IWorkspaceSidebarNavigationItem, "is_pinned" | "sort_order">
>;
export type SidebarNavigationData = Record<string, Partial<IWorkspaceSidebarNavigationItem>>;

type MutationScope = ReturnType<typeof captureIdentityScope>;

const service = new WorkspaceService();

export const navigationPreferenceKeys = {
  sidebar: (userId: string | undefined, slug: string | undefined) =>
    ["navigation-preferences", userId, slug, "sidebar"] as const,
  workspaceProperties: (userId: string | undefined, slug: string | undefined) =>
    ["navigation-preferences", userId, slug, "workspace-properties"] as const,
  sidebarMutation: (userId: string | undefined, slug: string | undefined, generation: number) =>
    ["navigation-preferences", userId, slug, "sidebar", "patch", generation] as const,
  workspacePropertiesMutation: (userId: string | undefined, slug: string | undefined, generation: number) =>
    ["navigation-preferences", userId, slug, "workspace-properties", "patch", generation] as const,
};

function isCurrentIdentity(client: QueryClient, userId: string | undefined, scope: MutationScope) {
  return Boolean(userId && isIdentityScopeCurrent(client, scope));
}

function currentQuery(client: QueryClient, queryKey: readonly unknown[]) {
  return client.getQueryCache().find({ queryKey, exact: true });
}

function scopedNavigationReadOptions<T>(
  userId: string | undefined,
  slug: string | undefined,
  queryKey: readonly unknown[],
  resource: string,
  fetch: (workspaceSlug: string, signal: AbortSignal) => Promise<T>
) {
  return queryOptions({
    queryKey,
    staleTime: 5 * 60_000,
    refetchOnMount: true,
    retry: false,
    queryFn:
      userId && slug
        ? async ({ client, signal }): Promise<T> => {
            const scope = captureIdentityScope(client, userId);
            const query = currentQuery(client, queryKey);
            const isCurrent = () =>
              isIdentityScopeVerified(client, scope) && query && query === currentQuery(client, queryKey);
            if (!isCurrent()) throw new Error(`The account session changed before ${resource} loaded.`);
            const preferences = await fetch(slug, signal);
            if (!isCurrent()) throw new Error(`The account session changed while ${resource} loaded.`);
            return preferences;
          }
        : skipToken,
  });
}

export function sidebarNavigationOptions(userId: string | undefined, slug: string | undefined) {
  return scopedNavigationReadOptions<SidebarNavigationData>(
    userId,
    slug,
    navigationPreferenceKeys.sidebar(userId, slug),
    "navigation preferences",
    (workspaceSlug, signal) => service.fetchSidebarNavigationPreferences(workspaceSlug, signal)
  );
}

export function workspacePropertiesOptions(userId: string | undefined, slug: string | undefined) {
  return scopedNavigationReadOptions<IWorkspaceUserPropertiesResponse>(
    userId,
    slug,
    navigationPreferenceKeys.workspaceProperties(userId, slug),
    "workspace preferences",
    (workspaceSlug, signal) => service.fetchWorkspaceFilters(workspaceSlug, signal)
  );
}

function createMutationScope(client: QueryClient, userId: string | undefined, slug: string | undefined) {
  const identity = captureIdentityScope(client, userId);
  const generation = identity?.generation ?? getIdentitySessionGeneration(client);
  const assertIdentity = () => {
    if (!slug || !isIdentityScopeVerified(client, identity))
      throw new Error("The account session has changed or is unverified.");
  };
  return { identity, generation, assertIdentity };
}

function isCapturedQueryCurrent(
  client: QueryClient,
  userId: string | undefined,
  identity: MutationScope,
  queryKey: readonly unknown[],
  capturedQuery: ReturnType<typeof currentQuery>
) {
  return Boolean(
    capturedQuery && isCurrentIdentity(client, userId, identity) && capturedQuery === currentQuery(client, queryKey)
  );
}

export function applySidebarPreferencePatches(
  previous: SidebarNavigationData | undefined,
  patches: SidebarPreferencePatch[]
) {
  const result = { ...previous };
  for (const patch of patches) {
    const { key, ...fields } = patch;
    result[key] = { ...result[key], ...fields };
  }
  return result;
}

export function patchSidebarNavigationOptions(
  client: QueryClient,
  userId: string | undefined,
  slug: string | undefined
) {
  const { identity, generation, assertIdentity } = createMutationScope(client, userId, slug);
  const queryKey = navigationPreferenceKeys.sidebar(userId, slug);
  return mutationOptions({
    mutationKey: navigationPreferenceKeys.sidebarMutation(userId, slug, generation),
    scope: { id: `navigation/${userId}/${slug}/sidebar/${generation}` },
    retry: false,
    onMutate: () => currentQuery(client, queryKey),
    mutationFn: async (patches: SidebarPreferencePatch[]) => {
      assertIdentity();
      if (!slug) throw new Error("A workspace is required.");
      await service.updateBulkSidebarPreferences(slug, patches);
    },
    onSuccess: async (_saved, patches, capturedQuery) => {
      if (!isCapturedQueryCurrent(client, userId, identity, queryKey, capturedQuery)) return;
      await client.cancelQueries({ queryKey, exact: true });
      if (!isCapturedQueryCurrent(client, userId, identity, queryKey, capturedQuery)) return;
      const confirmed = client.getQueryData<SidebarNavigationData>(queryKey);
      client.setQueryData(queryKey, applySidebarPreferencePatches(confirmed, patches));
      void client.invalidateQueries({ queryKey, exact: true }).catch(() => undefined);
    },
  });
}

export function patchWorkspaceNavigationOptions(
  client: QueryClient,
  userId: string | undefined,
  slug: string | undefined
) {
  const { identity, generation, assertIdentity } = createMutationScope(client, userId, slug);
  const queryKey = navigationPreferenceKeys.workspaceProperties(userId, slug);
  return mutationOptions({
    mutationKey: navigationPreferenceKeys.workspacePropertiesMutation(userId, slug, generation),
    scope: { id: `navigation/${userId}/${slug}/workspace-properties/${generation}` },
    retry: false,
    onMutate: () => currentQuery(client, queryKey),
    mutationFn: async (patch: Partial<IWorkspaceUserPropertiesResponse>) => {
      assertIdentity();
      if (!slug) throw new Error("A workspace is required.");
      return service.patchWorkspaceFilters(slug, patch);
    },
    onSuccess: async (saved: IWorkspaceUserPropertiesResponse, _patch, capturedQuery) => {
      if (!isCapturedQueryCurrent(client, userId, identity, queryKey, capturedQuery)) return;
      await client.cancelQueries({ queryKey, exact: true });
      if (isCapturedQueryCurrent(client, userId, identity, queryKey, capturedQuery))
        client.setQueryData(queryKey, saved);
    },
  });
}

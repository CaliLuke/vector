/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { IProjectUserPropertiesResponse } from "@plane/types";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import { ProjectService } from "@/services/project";

const service = new ProjectService();

export type ProjectUserPropertiesScope = {
  accountId: string | undefined;
  workspaceSlug: string | undefined;
  projectId: string | undefined;
};

export type ProjectUserPropertiesQueryAccess = {
  canAccess: boolean;
  fetchEnabled: boolean;
};

type ProjectPreferencesPatch = {
  pages?: Partial<IProjectUserPropertiesResponse["preferences"]["pages"]>;
  navigation?: Partial<IProjectUserPropertiesResponse["preferences"]["navigation"]>;
};

export type ProjectUserPropertiesPatch = {
  preferences?: ProjectPreferencesPatch;
  navigationAction?: { type: "hide-tab" | "show-tab"; tabKey: string };
  rich_filters?: IProjectUserPropertiesResponse["rich_filters"];
  display_filters?: Partial<IProjectUserPropertiesResponse["display_filters"]>;
  display_properties?: Partial<IProjectUserPropertiesResponse["display_properties"]>;
  sort_order?: IProjectUserPropertiesResponse["sort_order"];
};

export const projectUserPropertiesKeys = {
  detail: ({ accountId, workspaceSlug, projectId }: ProjectUserPropertiesScope) =>
    ["project-user-properties", accountId, workspaceSlug, projectId] as const,
  patch: ({ accountId, workspaceSlug, projectId }: ProjectUserPropertiesScope, generation: number) =>
    ["project-user-properties", accountId, workspaceSlug, projectId, "patch", generation] as const,
};

function findExactQuery(client: QueryClient, queryKey: readonly unknown[]) {
  return client.getQueryCache().find({ queryKey, exact: true });
}

function isCurrentIdentity(
  client: QueryClient,
  accountId: string | undefined,
  identity: ReturnType<typeof captureIdentityScope>
) {
  return Boolean(accountId && isIdentityScopeCurrent(client, identity));
}

function isVerifiedIdentity(
  client: QueryClient,
  accountId: string | undefined,
  identity: ReturnType<typeof captureIdentityScope>
) {
  return Boolean(accountId && isIdentityScopeVerified(client, identity));
}

export function projectUserPropertiesOptions(
  client: QueryClient,
  scope: ProjectUserPropertiesScope,
  access: ProjectUserPropertiesQueryAccess
) {
  const { accountId, workspaceSlug, projectId } = scope;
  const queryKey = projectUserPropertiesKeys.detail(scope);
  const canRunQuery = Boolean(access.canAccess && accountId && workspaceSlug && projectId);
  return queryOptions({
    queryKey,
    staleTime: 30_000,
    refetchOnMount: true,
    retry: false,
    enabled: canRunQuery && access.fetchEnabled,
    queryFn:
      access.canAccess && accountId && workspaceSlug && projectId
        ? async ({ signal }) => {
            const identity = captureIdentityScope(client, accountId);
            const query = findExactQuery(client, queryKey);
            const isCurrent = () =>
              isVerifiedIdentity(client, accountId, identity) && findExactQuery(client, queryKey) === query;
            if (!isCurrent()) throw new Error("The project preferences scope changed before loading.");
            const properties = await service.getProjectUserProperties(workspaceSlug, projectId, signal);
            if (!isCurrent()) throw new Error("The project preferences scope changed while loading.");
            return properties;
          }
        : skipToken,
  });
}

export function applyProjectUserPropertiesPatches(
  initial: IProjectUserPropertiesResponse,
  patches: readonly ProjectUserPropertiesPatch[]
) {
  const result = { ...initial };
  for (const patch of patches) {
    const preferences = patch.preferences;
    const navigationAction = patch.navigationAction;
    Object.assign(result, {
      ...(patch.rich_filters !== undefined ? { rich_filters: patch.rich_filters } : {}),
      ...(patch.display_filters !== undefined
        ? {
            display_filters: {
              ...result.display_filters,
              ...patch.display_filters,
              ...(patch.display_filters.calendar !== undefined
                ? { calendar: { ...result.display_filters.calendar, ...patch.display_filters.calendar } }
                : {}),
            },
          }
        : {}),
      ...(patch.display_properties !== undefined
        ? { display_properties: { ...result.display_properties, ...patch.display_properties } }
        : {}),
      ...(patch.sort_order !== undefined ? { sort_order: patch.sort_order } : {}),
    });
    if (preferences) {
      result.preferences = {
        ...result.preferences,
        ...(preferences.pages ? { pages: { ...result.preferences.pages, ...preferences.pages } } : {}),
        ...(preferences.navigation
          ? { navigation: { ...result.preferences.navigation, ...preferences.navigation } }
          : {}),
      };
    }
    if (navigationAction) {
      const hiddenTabs = result.preferences.navigation.hide_in_more_menu;
      result.preferences = {
        ...result.preferences,
        navigation: {
          ...result.preferences.navigation,
          hide_in_more_menu:
            navigationAction.type === "hide-tab"
              ? hiddenTabs.includes(navigationAction.tabKey)
                ? hiddenTabs
                : [...hiddenTabs, navigationAction.tabKey]
              : hiddenTabs.filter((key) => key !== navigationAction.tabKey),
        },
      };
    }
  }
  return result;
}

function payloadForPatch(
  base: IProjectUserPropertiesResponse,
  currentPatch: ProjectUserPropertiesPatch
): Partial<IProjectUserPropertiesResponse> {
  const composed = applyProjectUserPropertiesPatches(base, [currentPatch]);
  return {
    ...(currentPatch.preferences !== undefined || currentPatch.navigationAction !== undefined
      ? { preferences: composed.preferences }
      : {}),
    ...(currentPatch.rich_filters !== undefined ? { rich_filters: composed.rich_filters } : {}),
    ...(currentPatch.display_filters !== undefined ? { display_filters: composed.display_filters } : {}),
    ...(currentPatch.display_properties !== undefined ? { display_properties: composed.display_properties } : {}),
    ...(currentPatch.sort_order !== undefined ? { sort_order: composed.sort_order } : {}),
  };
}

export function patchProjectUserPropertiesOptions(
  client: QueryClient,
  scope: ProjectUserPropertiesScope,
  canAccess: boolean
) {
  const { accountId, workspaceSlug, projectId } = scope;
  const identity = captureIdentityScope(client, accountId);
  const generation = getIdentitySessionGeneration(client);
  const queryKey = projectUserPropertiesKeys.detail(scope);
  const mutationKey = projectUserPropertiesKeys.patch(scope, generation);
  let capturedQuery = findExactQuery(client, queryKey);
  return mutationOptions({
    mutationKey,
    scope: { id: `project-user-properties/${accountId}/${workspaceSlug}/${projectId}/${generation}` },
    retry: false,
    onMutate: async () => {
      if (!isVerifiedIdentity(client, accountId, identity))
        throw new Error("The account or project preferences scope changed or is unverified.");
      await client.cancelQueries({ queryKey, exact: true });
      capturedQuery ??= findExactQuery(client, queryKey);
      return { query: capturedQuery };
    },
    mutationFn: async (patch: ProjectUserPropertiesPatch) => {
      capturedQuery ??= findExactQuery(client, queryKey);
      if (
        !canAccess ||
        !accountId ||
        !workspaceSlug ||
        !projectId ||
        !capturedQuery ||
        !isVerifiedIdentity(client, accountId, identity) ||
        findExactQuery(client, queryKey) !== capturedQuery
      )
        throw new Error("The account or project preferences scope changed.");
      const base = client.getQueryData<IProjectUserPropertiesResponse>(queryKey);
      if (!base) throw new Error("Project preferences must load before they can be updated.");
      return service.updateProjectUserProperties(workspaceSlug, projectId, payloadForPatch(base, patch));
    },
    onSuccess: async (saved, _patch, captured) => {
      const isCurrent = () =>
        canAccess &&
        isCurrentIdentity(client, accountId, identity) &&
        captured?.query === capturedQuery &&
        captured?.query &&
        findExactQuery(client, queryKey) === captured.query;
      if (!isCurrent()) return;
      await client.cancelQueries({ queryKey, exact: true });
      if (!isCurrent()) return;
      client.setQueryData(queryKey, saved);
      // The PATCH response is canonical. A failed background synchronization must
      // leave the mutation accepted and the response available for display/retry.
      void client.invalidateQueries({ queryKey, exact: true }).catch(() => undefined);
    },
  });
}

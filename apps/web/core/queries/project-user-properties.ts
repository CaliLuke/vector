/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient, QueryStatus } from "@tanstack/react-query";
import type { IProjectUserPropertiesResponse } from "@plane/types";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import { ProjectService } from "@/services/project";
import type { ProjectUserPropertiesFields } from "@/services/project";
import {
  ProjectAccessReadError,
  isDefinitiveProjectAccessDenial,
  isTransientProjectAccessError,
  projectAccessReadRetry,
} from "./project-access-diagnostics";

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

export type ProjectUserPropertiesPatch = ProjectUserPropertiesFields & {
  navigationAction?: { type: "hide-tab" | "show-tab"; tabKey: string };
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

export function projectPreferencesDataAvailable(
  state: { data: unknown; status: QueryStatus; error: unknown } | undefined
) {
  return (
    state?.data !== undefined &&
    (state.status === "success" || (state.status === "error" && isTransientProjectAccessError(state.error)))
  );
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
    ...projectAccessReadRetry("project_preferences"),
    enabled: canRunQuery && access.fetchEnabled,
    queryFn:
      access.canAccess && accountId && workspaceSlug && projectId
        ? async ({ signal }) => {
            const identity = captureIdentityScope(client, accountId);
            const query = findExactQuery(client, queryKey);
            const isCurrent = () =>
              isVerifiedIdentity(client, accountId, identity) && findExactQuery(client, queryKey) === query;
            if (!isCurrent())
              throw new ProjectAccessReadError(
                "scope_changed",
                "The project preferences scope changed before loading."
              );
            let properties: IProjectUserPropertiesResponse;
            try {
              properties = await service.getProjectUserProperties(workspaceSlug, projectId, signal);
            } catch (error) {
              // A subsequent transient retry must never resurrect data after a definitive denial.
              if (query && isCurrent() && !signal.aborted && isDefinitiveProjectAccessDenial(error)) {
                query.setState({ data: undefined });
              }
              throw error;
            }
            if (!isCurrent())
              throw new ProjectAccessReadError("scope_changed", "The project preferences scope changed while loading.");
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
      if (!projectPreferencesDataAvailable(capturedQuery.state))
        throw new Error("Project preferences must load before they can be updated.");
      const { navigationAction, ...fields } = patch;
      return service.updateProjectUserProperties(workspaceSlug, projectId, {
        ...fields,
        merge: true,
        ...(navigationAction
          ? { navigation_action: { type: navigationAction.type, tab_key: navigationAction.tabKey } }
          : {}),
      });
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

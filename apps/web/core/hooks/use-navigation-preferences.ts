/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import sortBy from "lodash-es/sortBy.js";
import { useMutation, useMutationState, useQuery, useQueryClient } from "@tanstack/react-query";
import type { DefaultError, Mutation, QueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { DEFAULT_APP_RAIL_PREFERENCES, DEFAULT_PERSONAL_PREFERENCES, DEFAULT_PROJECT_PREFERENCES } from "@plane/types";
import type {
  IWorkspaceUserPropertiesResponse,
  TAppRailDisplayMode,
  TAppRailPreferences,
  TPersonalNavigationItemKey,
  TPersonalNavigationPreferences,
  TProjectNavigationMode,
  TProjectNavigationPreferences,
  TWorkspaceNavigationItemState,
  TWorkspaceNavigationPreferences,
} from "@plane/types";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { captureIdentityScope, getIdentitySessionGeneration, isIdentityScopeCurrent } from "@/queries/identity";
import {
  applySidebarPreferencePatches,
  navigationPreferenceKeys,
  patchSidebarNavigationOptions,
  patchWorkspaceNavigationOptions,
  sidebarNavigationOptions,
  workspacePropertiesOptions,
} from "@/queries/navigation-preferences";
import type { SidebarPreferencePatch } from "@/queries/navigation-preferences";
import { useIdentityGeneration } from "./use-identity-generation";
import { useCurrentUser } from "./use-current-user";
import { useLocalStorage } from "@plane/hooks";
import { useParams } from "@tanstack/react-router";

const APP_RAIL_PREFERENCES_KEY = "app_rail_preferences";

type PreferenceRouter = ReturnType<typeof useRouter>;
type Translator = ReturnType<typeof useTranslation>["t"];

function reportPreferenceFailure(
  client: QueryClient,
  userId: string | undefined,
  generation: number,
  router: PreferenceRouter,
  t: Translator
) {
  const identity = captureIdentityScope(client, userId);
  const routeHref = router.state.location.href;
  return () => {
    if (
      generation !== getIdentitySessionGeneration(client) ||
      !isIdentityScopeCurrent(client, identity) ||
      routeHref !== router.state.location.href
    )
      return;
    setToast({ type: TOAST_TYPE.ERROR, title: t("toast.error"), message: t("common.error.message") });
  };
}

type PendingSidebarPatch = {
  variables: SidebarPreferencePatch[];
  submittedAt: number;
  mutationId: number;
};

type SidebarPatchMutation = Mutation<void, DefaultError, SidebarPreferencePatch[]>;

function usePendingSidebarPatches(userId: string | undefined, slug: string | undefined, generation: number) {
  const mutations = useMutationState<PendingSidebarPatch, SidebarPatchMutation>({
    filters: {
      mutationKey: navigationPreferenceKeys.sidebarMutation(userId, slug, generation),
      exact: true,
      status: "pending",
    },
    select: (mutation) => ({
      variables: mutation.state.variables ?? [],
      submittedAt: mutation.state.submittedAt,
      mutationId: mutation.mutationId,
    }),
  });
  return useMemo(
    () =>
      sortBy(mutations, [(mutation) => mutation.submittedAt, (mutation) => mutation.mutationId]).flatMap(
        ({ variables }) => variables
      ),
    [mutations]
  );
}

type PendingWorkspacePatch = {
  variables: Partial<IWorkspaceUserPropertiesResponse>;
  submittedAt: number;
  mutationId: number;
};

type WorkspacePatchMutation = Mutation<
  IWorkspaceUserPropertiesResponse,
  DefaultError,
  Partial<IWorkspaceUserPropertiesResponse>
>;

function usePendingWorkspacePatches(userId: string | undefined, slug: string | undefined, generation: number) {
  const mutations = useMutationState<PendingWorkspacePatch, WorkspacePatchMutation>({
    filters: {
      mutationKey: navigationPreferenceKeys.workspacePropertiesMutation(userId, slug, generation),
      exact: true,
      status: "pending",
    },
    select: (mutation) => ({
      variables: mutation.state.variables ?? {},
      submittedAt: mutation.state.submittedAt,
      mutationId: mutation.mutationId,
    }),
  });
  return useMemo(
    () =>
      sortBy(mutations, [(mutation) => mutation.submittedAt, (mutation) => mutation.mutationId]).map(
        ({ variables }) => variables
      ),
    [mutations]
  );
}

type WorkspaceNavigationFields = Pick<
  IWorkspaceUserPropertiesResponse,
  "navigation_control_preference" | "navigation_project_limit"
>;

function mergeWorkspacePatches(
  data: IWorkspaceUserPropertiesResponse | undefined,
  patches: Partial<IWorkspaceUserPropertiesResponse>[]
): WorkspaceNavigationFields {
  return patches.reduce<WorkspaceNavigationFields>(
    (result, patch) => ({
      navigation_control_preference: patch.navigation_control_preference ?? result.navigation_control_preference,
      navigation_project_limit: patch.navigation_project_limit ?? result.navigation_project_limit,
    }),
    {
      navigation_control_preference: data?.navigation_control_preference,
      navigation_project_limit: data?.navigation_project_limit,
    }
  );
}

function useSidebarNavigation() {
  const { workspaceSlug } = useParams({ strict: false });
  const client = useQueryClient();
  const router = useRouter();
  const { t } = useTranslation();
  const { data: user } = useCurrentUser();
  const userId = user?.id;
  const slug = workspaceSlug;
  const generation = useIdentityGeneration(client);
  const { data: confirmed } = useQuery(sidebarNavigationOptions(userId, slug));
  const pending = usePendingSidebarPatches(userId, slug, generation);
  const preferences = useMemo(() => applySidebarPreferencePatches(confirmed, pending), [confirmed, pending]);
  const mutation = useMutation(patchSidebarNavigationOptions(client, userId, slug));
  const updateSidebar = useCallback(
    (patches: SidebarPreferencePatch[]) => {
      if (!slug) return;
      void mutation.mutateAsync(patches).catch(reportPreferenceFailure(client, userId, generation, router, t));
    },
    [client, generation, mutation, router, slug, t, userId]
  );
  return { preferences, updateSidebar };
}

export const usePersonalNavigationPreferences = () => {
  const { preferences: preferencesData, updateSidebar } = useSidebarNavigation();

  const preferences: TPersonalNavigationPreferences = useMemo(() => {
    const personalItems: TPersonalNavigationPreferences["items"] = {
      stickies: {
        enabled: preferencesData.stickies?.is_pinned ?? DEFAULT_PERSONAL_PREFERENCES.items.stickies.enabled,
        sort_order: preferencesData.stickies?.sort_order ?? DEFAULT_PERSONAL_PREFERENCES.items.stickies.sort_order,
      },
      your_work: {
        enabled: preferencesData.your_work?.is_pinned ?? DEFAULT_PERSONAL_PREFERENCES.items.your_work.enabled,
        sort_order: preferencesData.your_work?.sort_order ?? DEFAULT_PERSONAL_PREFERENCES.items.your_work.sort_order,
      },
      drafts: {
        enabled: preferencesData.drafts?.is_pinned ?? DEFAULT_PERSONAL_PREFERENCES.items.drafts.enabled,
        sort_order: preferencesData.drafts?.sort_order ?? DEFAULT_PERSONAL_PREFERENCES.items.drafts.sort_order,
      },
    };
    return { items: personalItems };
  }, [preferencesData]);

  const togglePersonalItem = useCallback(
    (key: TPersonalNavigationItemKey, enabled: boolean) => {
      updateSidebar([{ key, is_pinned: enabled }]);
    },
    [updateSidebar]
  );
  const updatePersonalItemOrder = useCallback(
    (items: Array<{ key: TPersonalNavigationItemKey; sortOrder: number }>) => {
      updateSidebar(items.map(({ key, sortOrder }) => ({ key, sort_order: sortOrder })));
    },
    [updateSidebar]
  );
  const isPersonalItemEnabled = useCallback(
    (key: TPersonalNavigationItemKey): boolean => preferences.items[key]?.enabled ?? true,
    [preferences]
  );

  return { preferences, togglePersonalItem, updatePersonalItemOrder, isPersonalItemEnabled };
};

export const useProjectNavigationPreferences = () => {
  const { workspaceSlug } = useParams({ strict: false });
  const client = useQueryClient();
  const router = useRouter();
  const { t } = useTranslation();
  const { data: user } = useCurrentUser();
  const userId = user?.id;
  const slug = workspaceSlug;
  const generation = useIdentityGeneration(client);
  const { data: confirmed } = useQuery(workspacePropertiesOptions(userId, slug));
  const pending = usePendingWorkspacePatches(userId, slug, generation);
  const workspaceProperties = useMemo(() => mergeWorkspacePatches(confirmed, pending), [confirmed, pending]);
  const mutation = useMutation(patchWorkspaceNavigationOptions(client, userId, slug));

  const preferences: TProjectNavigationPreferences = useMemo(() => {
    if (
      workspaceProperties.navigation_control_preference ||
      workspaceProperties.navigation_project_limit !== undefined
    ) {
      const limit = workspaceProperties.navigation_project_limit ?? DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount;
      return {
        navigationMode: workspaceProperties.navigation_control_preference || DEFAULT_PROJECT_PREFERENCES.navigationMode,
        limitedProjectsCount: limit > 0 ? limit : DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount,
        showLimitedProjects: limit > 0,
      };
    }
    return DEFAULT_PROJECT_PREFERENCES;
  }, [workspaceProperties]);

  const updateNavigationMode = useCallback(
    (mode: TProjectNavigationMode) => {
      if (!slug) return;
      void mutation
        .mutateAsync({ navigation_control_preference: mode })
        .catch(reportPreferenceFailure(client, userId, generation, router, t));
    },
    [client, generation, mutation, router, slug, t, userId]
  );
  const updateShowLimitedProjects = useCallback(
    (show: boolean) => {
      if (!slug) return;
      const limit = show ? preferences.limitedProjectsCount || DEFAULT_PROJECT_PREFERENCES.limitedProjectsCount : 0;
      void mutation
        .mutateAsync({ navigation_project_limit: limit })
        .catch(reportPreferenceFailure(client, userId, generation, router, t));
    },
    [client, generation, mutation, preferences.limitedProjectsCount, router, slug, t, userId]
  );
  const updateLimitedProjectsCount = useCallback(
    (count: number) => {
      if (!slug) return Promise.reject(new Error("A workspace is required."));
      return mutation.mutateAsync({ navigation_project_limit: count });
    },
    [mutation, slug]
  );

  return { preferences, updateNavigationMode, updateShowLimitedProjects, updateLimitedProjectsCount };
};

export const useWorkspaceNavigationPreferences = () => {
  const { preferences: sidebar, updateSidebar } = useSidebarNavigation();

  const preferences: TWorkspaceNavigationPreferences = useMemo(() => {
    const items = Object.fromEntries(
      Object.entries(sidebar).map(([key, item]) => [
        key,
        {
          is_pinned: item.is_pinned ?? false,
          sort_order: item.sort_order ?? 0,
        },
      ])
    );
    return { items };
  }, [sidebar]);

  const toggleWorkspaceItem = useCallback(
    (key: string, isPinned: boolean) => {
      updateSidebar([{ key, is_pinned: isPinned }]);
    },
    [updateSidebar]
  );
  const updateWorkspaceItemOrder = useCallback(
    (items: Array<{ key: string; sortOrder: number }>) => {
      updateSidebar(items.map(({ key, sortOrder }) => ({ key, sort_order: sortOrder })));
    },
    [updateSidebar]
  );
  const getWorkspaceItemState = useCallback(
    (key: string): TWorkspaceNavigationItemState => preferences.items[key] || { is_pinned: false, sort_order: 0 },
    [preferences]
  );
  const isWorkspaceItemPinned = useCallback(
    (key: string): boolean => getWorkspaceItemState(key).is_pinned,
    [getWorkspaceItemState]
  );
  const updateWorkspaceItemSortOrder = useCallback(
    (key: string, sortOrder: number) => {
      updateSidebar([{ key, sort_order: sortOrder }]);
    },
    [updateSidebar]
  );

  return {
    preferences,
    toggleWorkspaceItem,
    updateWorkspaceItemOrder,
    updateWorkspaceItemSortOrder,
    getWorkspaceItemState,
    isWorkspaceItemPinned,
  };
};

export const useAppRailPreferences = () => {
  const { storedValue, setValue } = useLocalStorage<TAppRailPreferences>(
    APP_RAIL_PREFERENCES_KEY,
    DEFAULT_APP_RAIL_PREFERENCES
  );

  const updateDisplayMode = useCallback(
    (mode: TAppRailDisplayMode) => {
      setValue({ displayMode: mode });
    },
    [setValue]
  );

  const toggleDisplayMode = useCallback(() => {
    const currentPreferences = storedValue || DEFAULT_APP_RAIL_PREFERENCES;
    const newMode = currentPreferences.displayMode === "icon_only" ? "icon_with_label" : "icon_only";
    updateDisplayMode(newMode);
  }, [storedValue, updateDisplayMode]);

  return {
    preferences: storedValue || DEFAULT_APP_RAIL_PREFERENCES,
    updateDisplayMode,
    toggleDisplayMode,
  };
};

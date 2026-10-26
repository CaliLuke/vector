/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import { EUserWorkspaceRoles } from "@plane/types";
import type { IProjectView, TUserPermissions } from "@plane/types";
import type { TProjectRosterScope } from "@/queries/member-rosters";
import { getValidatedViewFilters } from "@plane/utils/project-views";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import { captureWorkspaceFavoritesRefresh } from "@/queries/favorites";
import { ViewService } from "@/services/view.service";

export type TProjectViewsScope = TProjectRosterScope & {
  projectMembershipRole: TUserPermissions | undefined;
  sessionGeneration: number;
};

const service = new ViewService();

export const projectViewKeys = {
  list: (scope: TProjectViewsScope) =>
    [
      "project-views",
      scope.accountId,
      scope.workspaceSlug,
      scope.membershipRole,
      scope.membershipRevision,
      scope.projectId,
      scope.projectMembershipRole,
      scope.projectRole,
      scope.sessionGeneration,
      "list",
    ] as const,
  detail: (scope: TProjectViewsScope, viewId: string | undefined) =>
    [
      "project-views",
      scope.accountId,
      scope.workspaceSlug,
      scope.membershipRole,
      scope.membershipRevision,
      scope.projectId,
      scope.projectMembershipRole,
      scope.projectRole,
      scope.sessionGeneration,
      "detail",
      viewId,
    ] as const,
  mutations: (scope: TProjectViewsScope, generation: number) => [
    "project-views",
    scope.accountId,
    scope.workspaceSlug,
    scope.membershipRole,
    scope.membershipRevision,
    scope.projectId,
    scope.projectMembershipRole,
    scope.projectRole,
    scope.sessionGeneration,
    generation,
  ],
};

function canRead(scope: TProjectViewsScope) {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.projectId &&
    scope.membershipRevision &&
    scope.membershipRole !== undefined &&
    scope.projectMembershipRole !== undefined &&
    scope.projectRole !== undefined
  );
}

export function projectViewsOptions(client: QueryClient, scope: TProjectViewsScope) {
  const queryKey = projectViewKeys.list(scope);
  return queryOptions({
    queryKey,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    retry: false,
    queryFn: canRead(scope)
      ? async ({ signal }) => {
          const access = captureProjectAccessScope(client, scope);
          if (!access.isVerified()) throw new Error("Project view access is unavailable.");
          const views = await service.getViews(scope.workspaceSlug!, scope.projectId!, signal);
          if (!access.isVerified()) throw new Error("Project view access changed while loading.");
          return views;
        }
      : skipToken,
  });
}

export function projectViewOptions(client: QueryClient, scope: TProjectViewsScope, viewId: string | undefined) {
  const queryKey = projectViewKeys.detail(scope, viewId);
  return queryOptions({
    queryKey,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    retry: false,
    queryFn:
      canRead(scope) && viewId
        ? async ({ signal }) => {
            const access = captureProjectAccessScope(client, scope);
            if (!access.isVerified()) throw new Error("Project view access is unavailable.");
            const view = await service.getViewDetails(scope.workspaceSlug!, scope.projectId!, viewId, signal);
            if (!access.isVerified()) throw new Error("Project view access changed while loading.");
            return view;
          }
        : skipToken,
  });
}

type CapturedMutation = {
  access: ReturnType<typeof captureProjectAccessScope>;
  listKey: ReturnType<typeof projectViewKeys.list>;
  listQuery: Query | undefined;
  detailKey: ReturnType<typeof projectViewKeys.detail>;
  detailQuery: Query | undefined;
};

function captureMutation(client: QueryClient, scope: TProjectViewsScope, viewId?: string): CapturedMutation {
  const access = captureProjectAccessScope(client, scope);
  const listKey = projectViewKeys.list(scope);
  const detailKey = projectViewKeys.detail(scope, viewId);
  return {
    access,
    listKey,
    listQuery: client.getQueryCache().find({ queryKey: listKey, exact: true }),
    detailKey,
    detailQuery: client.getQueryCache().find({ queryKey: detailKey, exact: true }),
  };
}

function queryIsCurrent(client: QueryClient, key: readonly unknown[], query: Query | undefined) {
  return query !== undefined && client.getQueryCache().find({ queryKey: key, exact: true }) === query;
}

async function cancelCapturedQueries(client: QueryClient, captured: CapturedMutation) {
  const cancellations = [];
  if (captured.listQuery && queryIsCurrent(client, captured.listKey, captured.listQuery)) {
    cancellations.push(client.cancelQueries({ queryKey: captured.listKey, exact: true }));
  }
  if (captured.detailQuery && queryIsCurrent(client, captured.detailKey, captured.detailQuery)) {
    cancellations.push(client.cancelQueries({ queryKey: captured.detailKey, exact: true }));
  }
  await Promise.all(cancellations);
}

function reconcileCapturedCaches(
  client: QueryClient,
  scope: TProjectViewsScope,
  captured: CapturedMutation,
  change: (views: IProjectView[]) => IProjectView[],
  detail?: IProjectView
) {
  if (captured.listQuery && queryIsCurrent(client, captured.listKey, captured.listQuery)) {
    const current = client.getQueryData<IProjectView[]>(captured.listKey);
    if (current !== undefined) client.setQueryData<IProjectView[]>(captured.listKey, change(current));
  }
  if (detail && captured.detailQuery && queryIsCurrent(client, captured.detailKey, captured.detailQuery)) {
    client.setQueryData(captured.detailKey, detail);
  } else if (detail && !captured.detailQuery) {
    const detailKey = projectViewKeys.detail(scope, detail.id);
    if (!client.getQueryCache().find({ queryKey: detailKey, exact: true })) client.setQueryData(detailKey, detail);
  }
}

async function reconcile(
  client: QueryClient,
  scope: TProjectViewsScope,
  captured: CapturedMutation | undefined,
  change: (views: IProjectView[]) => IProjectView[],
  detail?: IProjectView
) {
  if (!captured || !captured.access.isCurrent()) return;

  await cancelCapturedQueries(client, captured);
  if (!captured.access.isCurrent()) return;
  reconcileCapturedCaches(client, scope, captured, change, detail);
  const favoritesRefresh = captureWorkspaceFavoritesRefresh(client, scope.workspaceSlug!);
  await Promise.all([
    client.invalidateQueries({ queryKey: projectViewKeys.list(scope), exact: true }),
    detail
      ? client.invalidateQueries({ queryKey: projectViewKeys.detail(scope, detail.id), exact: true })
      : Promise.resolve(),
    favoritesRefresh(),
  ]);
}

function createOptions(client: QueryClient, scope: TProjectViewsScope, generation: number) {
  return mutationOptions({
    mutationKey: [...projectViewKeys.mutations(scope, generation), "create"],
    scope: { id: `project-views/${scope.accountId}/${scope.projectId}/${generation}` },
    retry: false,
    onMutate: () => {
      const captured = captureMutation(client, scope);
      if (!captured.access.isVerified()) throw new Error("Project view access is unavailable or has changed.");
      return captured;
    },
    mutationFn: async (data: Partial<IProjectView>) => {
      if (!captureProjectAccessScope(client, scope).isVerified())
        throw new Error("Project view access is unavailable or has changed.");
      const saved = await service.createView(scope.workspaceSlug!, scope.projectId!, validatedViewPayload(data));
      if (!saved.id || saved.project !== scope.projectId) {
        void client.invalidateQueries({ queryKey: projectViewKeys.list(scope), exact: true });
        throw new Error("The server returned a different project view.");
      }
      return saved;
    },
    onSuccess: (saved, _data, state) => {
      return reconcile(
        client,
        scope,
        state,
        (views) => [...views.filter((view) => view.id !== saved.id), saved],
        saved
      );
    },
  });
}

function updateOptions(client: QueryClient, scope: TProjectViewsScope, generation: number) {
  return mutationOptions({
    mutationKey: [...projectViewKeys.mutations(scope, generation), "update"],
    scope: { id: `project-views/${scope.accountId}/${scope.projectId}/${generation}` },
    retry: false,
    onMutate: async ({ viewId }: { viewId: string }) => {
      const access = captureProjectAccessScope(client, scope);
      if (!access.isVerified()) throw new Error("Project view access is unavailable or has changed.");
      const view = await client.fetchQuery({ ...projectViewOptions(client, scope, viewId), staleTime: 0 });
      if (!access.isVerified() || view.owned_by !== scope.accountId || view.is_locked)
        throw new Error("Only the owner can update an unlocked project view.");
      return captureMutation(client, scope, viewId);
    },
    mutationFn: async ({ viewId, data }: { viewId: string; data: Partial<IProjectView> }) => {
      if (!captureProjectAccessScope(client, scope).isVerified())
        throw new Error("Project view access is unavailable or has changed.");
      const view = client.getQueryData<IProjectView>(projectViewKeys.detail(scope, viewId));
      if (
        client.getQueryState(projectViewKeys.detail(scope, viewId))?.status !== "success" ||
        !view ||
        view.project !== scope.projectId ||
        view.owned_by !== scope.accountId ||
        view.is_locked
      )
        throw new Error("Only the owner can update an unlocked project view.");
      const updated = await service.patchView(
        scope.workspaceSlug!,
        scope.projectId,
        viewId,
        validatedViewPayload(data)
      );
      if (updated.id !== viewId || updated.project !== scope.projectId) {
        void client.invalidateQueries({ queryKey: projectViewKeys.list(scope), exact: true });
        void client.invalidateQueries({ queryKey: projectViewKeys.detail(scope, viewId), exact: true });
        throw new Error("The server returned a different project view.");
      }
      return updated;
    },
    onSuccess: (saved, { viewId }, state) => {
      return reconcile(
        client,
        scope,
        state,
        (views) => views.map((view) => (view.id === viewId ? saved : view)),
        saved
      );
    },
  });
}

function deleteOptions(client: QueryClient, scope: TProjectViewsScope, generation: number) {
  return mutationOptions({
    mutationKey: [...projectViewKeys.mutations(scope, generation), "delete"],
    scope: { id: `project-views/${scope.accountId}/${scope.projectId}/${generation}` },
    retry: false,
    onMutate: async (viewId: string) => {
      const access = captureProjectAccessScope(client, scope);
      if (!access.isVerified()) throw new Error("Project view access is unavailable or has changed.");
      const view = await client.fetchQuery({ ...projectViewOptions(client, scope, viewId), staleTime: 0 });
      const isWorkspaceAdmin = scope.membershipRole === EUserWorkspaceRoles.ADMIN;
      if (!access.isVerified() || (view.owned_by !== scope.accountId && !isWorkspaceAdmin))
        throw new Error("You cannot delete this project view.");
      return captureMutation(client, scope, viewId);
    },
    mutationFn: async (viewId: string) => {
      if (!captureProjectAccessScope(client, scope).isVerified())
        throw new Error("Project view access is unavailable or has changed.");
      const view = client.getQueryData<IProjectView>(projectViewKeys.detail(scope, viewId));
      const isWorkspaceAdmin = scope.membershipRole === EUserWorkspaceRoles.ADMIN;
      if (
        client.getQueryState(projectViewKeys.detail(scope, viewId))?.status !== "success" ||
        !view ||
        view.project !== scope.projectId ||
        (view.owned_by !== scope.accountId && !isWorkspaceAdmin)
      )
        throw new Error("You cannot delete this project view.");
      await service.deleteView(scope.workspaceSlug!, scope.projectId, viewId);
      return viewId;
    },
    onSuccess: (viewId, _input, state) => {
      if (state?.access.isCurrent() && state.detailQuery && queryIsCurrent(client, state.detailKey, state.detailQuery))
        client.removeQueries({ queryKey: state.detailKey, exact: true });
      return reconcile(client, scope, state, (views) => views.filter((view) => view.id !== viewId));
    },
  });
}

export const projectViewMutationOptions = {
  create: createOptions,
  update: updateOptions,
  delete: deleteOptions,
};

function validatedViewPayload(data: Partial<IProjectView>) {
  return getValidatedViewFilters({
    ...data,
    display_filters: data.display_filters ? { ...data.display_filters } : undefined,
  });
}

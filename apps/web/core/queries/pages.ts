/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import { EPageAccess } from "@plane/constants";
import type {
  EUserWorkspaceRoles,
  TDocumentPayload,
  TPage,
  TPageFilters,
  TPageVersion,
  TUserPermissions,
} from "@plane/types";
import { filterPagesByPageType, getPageName, orderPages, shouldFilterPage } from "@plane/utils/page";
import { captureWorkspaceFavoritesRefresh } from "@/queries/favorites";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import { ProjectPageVersionService } from "@/services/page";
import { ProjectPageService } from "@/services/page";
import type { TProjectRosterScope } from "@/queries/member-rosters";

const service = new ProjectPageService();
const versionService = new ProjectPageVersionService();

export type TProjectPageScope = TProjectRosterScope & {
  projectMembershipRole: TUserPermissions | undefined;
  rolesRevision: string | undefined;
  sessionGeneration: number;
};

export type TProjectPageQueryScope = TProjectPageScope & {
  accountId: string;
  workspaceSlug: string;
  projectId: string;
  membershipRole: EUserWorkspaceRoles;
  membershipRevision: string;
  projectMembershipRole: TUserPermissions;
  projectRole: TUserPermissions;
  rolesRevision: string;
};

export const pageKeys = {
  project: (scope: TProjectPageScope) =>
    [
      "project-pages",
      scope.accountId,
      scope.workspaceSlug,
      scope.membershipRole,
      scope.membershipRevision,
      scope.projectId,
      scope.projectMembershipRole,
      scope.projectRole,
      scope.rolesRevision,
      scope.sessionGeneration,
      "list",
    ] as const,
  detail: (scope: TProjectPageScope, pageId: string | undefined) =>
    [
      "project-pages",
      scope.accountId,
      scope.workspaceSlug,
      scope.membershipRole,
      scope.membershipRevision,
      scope.projectId,
      scope.projectMembershipRole,
      scope.projectRole,
      scope.rolesRevision,
      scope.sessionGeneration,
      "detail",
      pageId,
    ] as const,
  mutations: (scope: TProjectPageScope, pageId: string | undefined) =>
    [
      "project-pages",
      scope.accountId,
      scope.workspaceSlug,
      scope.membershipRevision,
      scope.projectId,
      scope.projectMembershipRole,
      scope.rolesRevision,
      scope.sessionGeneration,
      pageId,
      "write",
    ] as const,
};

export function canReadProjectPages(scope: TProjectPageScope): scope is TProjectPageQueryScope {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.projectId &&
    scope.membershipRole !== undefined &&
    scope.membershipRevision &&
    scope.projectMembershipRole !== undefined &&
    scope.projectRole !== undefined &&
    scope.rolesRevision !== undefined
  );
}

export function projectPagesOptions(client: QueryClient, scope: TProjectPageScope) {
  const queryKey = pageKeys.project(scope);
  return queryOptions({
    queryKey,
    staleTime: 0,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    retry: false,
    queryFn: canReadProjectPages(scope)
      ? async ({ signal }) => {
          const access = captureProjectAccessScope(client, scope);
          if (!access.isVerified()) throw new Error("Project page access is unavailable.");
          const pages = await service.fetchAll(scope.workspaceSlug, scope.projectId, signal);
          if (!access.isVerified()) throw new Error("Project page access changed while loading.");
          return pages;
        }
      : skipToken,
  });
}

export function projectPageOptions(
  client: QueryClient,
  scope: TProjectPageScope,
  pageId: string | undefined,
  trackVisit = true
) {
  const queryKey = pageKeys.detail(scope, pageId);
  return queryOptions({
    queryKey,
    staleTime: 0,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    refetchOnReconnect: true,
    retry: false,
    queryFn:
      canReadProjectPages(scope) && pageId
        ? async ({ signal }) => {
            const access = captureProjectAccessScope(client, scope);
            if (!access.isVerified()) throw new Error("Project page access is unavailable.");
            const page = await service.fetchById(scope.workspaceSlug, scope.projectId, pageId, trackVisit, signal);
            if (!access.isVerified()) throw new Error("Project page access changed while loading.");
            if (page.id !== pageId || !page.project_ids?.includes(scope.projectId))
              throw new Error("The server returned a page outside the requested project.");
            return page;
          }
        : skipToken,
  });
}

type TPageMutationCapture = {
  access: ReturnType<typeof captureProjectAccessScope>;
  listKey: ReturnType<typeof pageKeys.project>;
  listQuery: Query | undefined;
  detailKey: ReturnType<typeof pageKeys.detail>;
  detailQuery: Query | undefined;
};

function capturePageMutation(client: QueryClient, scope: TProjectPageQueryScope, pageId?: string) {
  const listKey = pageKeys.project(scope);
  const detailKey = pageKeys.detail(scope, pageId);
  return {
    access: captureProjectAccessScope(client, scope),
    listKey,
    listQuery: client.getQueryCache().find({ queryKey: listKey, exact: true }),
    detailKey,
    detailQuery: client.getQueryCache().find({ queryKey: detailKey, exact: true }),
  } satisfies TPageMutationCapture;
}

function queryIsCurrent(client: QueryClient, key: readonly unknown[], query: Query | undefined) {
  return query !== undefined && client.getQueryCache().find({ queryKey: key, exact: true }) === query;
}

function captureIsCurrent(client: QueryClient, captured: TPageMutationCapture) {
  return (
    captured.access.isCurrent() &&
    (!captured.listQuery || queryIsCurrent(client, captured.listKey, captured.listQuery)) &&
    (!captured.detailQuery || queryIsCurrent(client, captured.detailKey, captured.detailQuery))
  );
}

async function cancelCapturedPages(client: QueryClient, captured: TPageMutationCapture) {
  const requests: Promise<void>[] = [];
  if (queryIsCurrent(client, captured.listKey, captured.listQuery))
    requests.push(client.cancelQueries({ queryKey: captured.listKey, exact: true }));
  if (queryIsCurrent(client, captured.detailKey, captured.detailQuery))
    requests.push(client.cancelQueries({ queryKey: captured.detailKey, exact: true }));
  await Promise.all(requests);
}

async function preparePageMutation(client: QueryClient, scope: TProjectPageScope, pageId?: string) {
  if (!canReadProjectPages(scope)) throw new Error("Project page access is unavailable.");
  const captured = capturePageMutation(client, scope, pageId);
  if (!captured.access.isVerified()) throw new Error("Project page access is unavailable.");
  await cancelCapturedPages(client, captured);
  return captured;
}

async function refreshCapturedPageList(client: QueryClient, captured: TPageMutationCapture) {
  if (!captureIsCurrent(client, captured)) return;
  if (queryIsCurrent(client, captured.listKey, captured.listQuery)) {
    await client.invalidateQueries({ queryKey: captured.listKey, exact: true, refetchType: "active" });
  }
}

function optimisticPageChange(
  client: QueryClient,
  captured: TPageMutationCapture,
  pageId: string,
  change: (page: TPage) => TPage
) {
  if (!captureIsCurrent(client, captured)) return undefined;
  const previousList = captured.listQuery ? client.getQueryData<TPage[]>(captured.listKey) : undefined;
  const previousDetail = captured.detailQuery ? client.getQueryData<TPage>(captured.detailKey) : undefined;
  const previousPage = previousDetail ?? previousList?.find((page) => page.id === pageId);
  if (!previousPage) return undefined;
  const optimistic = change(previousPage);
  if (captured.listQuery) {
    client.setQueryData<TPage[]>(captured.listKey, (pages) =>
      pages?.map((page) => (page.id === pageId ? optimistic : page))
    );
  }
  if (captured.detailQuery) client.setQueryData<TPage>(captured.detailKey, optimistic);
  return {
    previousList,
    previousDetail,
    optimisticList: client.getQueryData<TPage[]>(captured.listKey)?.find((page) => page.id === pageId),
    optimisticDetail: client.getQueryData<TPage>(captured.detailKey),
  };
}

function rollbackPageChange(
  client: QueryClient,
  captured: TPageMutationCapture,
  pageId: string,
  context: ReturnType<typeof optimisticPageChange>
) {
  if (!context || !captureIsCurrent(client, captured)) return;
  if (captured.listQuery) {
    client.setQueryData<TPage[]>(captured.listKey, (pages) =>
      pages?.map((page) =>
        page.id === pageId && page === context.optimisticList
          ? (context.previousList?.find((item) => item.id === pageId) ?? page)
          : page
      )
    );
  }
  if (captured.detailQuery && client.getQueryData(captured.detailKey) === context.optimisticDetail)
    client.setQueryData<TPage>(captured.detailKey, context.previousDetail);
}

function commitPageChange(
  client: QueryClient,
  captured: TPageMutationCapture,
  pageId: string,
  saved: TPage,
  optimistic?: ReturnType<typeof optimisticPageChange>
) {
  if (!captureIsCurrent(client, captured)) return;
  const currentDetail = captured.detailQuery ? client.getQueryData<TPage>(captured.detailKey) : undefined;
  const currentList = captured.listQuery ? client.getQueryData<TPage[]>(captured.listKey) : undefined;
  const current = currentDetail ?? currentList?.find((page) => page.id === pageId);
  if (captured.detailQuery && current === undefined) {
    void client.invalidateQueries({ queryKey: captured.detailKey, exact: true, refetchType: "active" });
    return;
  }
  const reconcile = (currentPage: TPage, expected: TPage | undefined): TPage => {
    if (!expected) return saved;
    const expectedFields = new Map(Object.entries(expected));
    const newerFields = Object.fromEntries(
      Object.entries(currentPage).filter(([key, value]) => value !== expectedFields.get(key))
    );
    return { ...saved, ...newerFields };
  };
  if (captured.listQuery && currentList?.some((page) => page.id === pageId)) {
    client.setQueryData<TPage[]>(captured.listKey, (pages) =>
      pages?.map((page) => (page.id === pageId ? reconcile(page, optimistic?.optimisticList) : page))
    );
  }
  if (captured.detailQuery && currentDetail)
    client.setQueryData<TPage>(captured.detailKey, reconcile(currentDetail, optimistic?.optimisticDetail));
}

export function createProjectPageOptions(client: QueryClient, scope: TProjectPageScope) {
  return mutationOptions({
    mutationKey: pageKeys.mutations(scope, undefined),
    scope: {
      id: `pages/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}/${scope.projectId}/create`,
    },
    retry: false,
    onMutate: async () => ({ captured: await preparePageMutation(client, scope) }),
    mutationFn: async (data: Partial<TPage>) => {
      if (!canReadProjectPages(scope) || !captureProjectAccessScope(client, scope).isVerified())
        throw new Error("Project page access is unavailable.");
      const saved = await service.create(scope.workspaceSlug, scope.projectId, data);
      if (!saved.id || !saved.project_ids?.includes(scope.projectId))
        throw new Error("The server returned an invalid project page.");
      return { ...saved, id: saved.id };
    },
    onSuccess: async (_saved, _variables, context) => {
      if (context) await refreshCapturedPageList(client, context.captured);
    },
  });
}

export function updateProjectPageOptions(client: QueryClient, scope: TProjectPageScope, pageId: string) {
  return mutationOptions({
    mutationKey: pageKeys.mutations(scope, pageId),
    scope: {
      id: `pages/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}/${scope.projectId}/${pageId}`,
    },
    retry: false,
    onMutate: async (change: Partial<TPage>) => {
      if (!canReadProjectPages(scope)) throw new Error("Project page access is unavailable.");
      const captured = capturePageMutation(client, scope, pageId);
      if (!captured.access.isVerified()) throw new Error("Project page access is unavailable.");
      await cancelCapturedPages(client, captured);
      return {
        captured,
        optimistic: optimisticPageChange(client, captured, pageId, (page) => ({ ...page, ...change })),
      };
    },
    mutationFn: async (change: Partial<TPage>) => {
      if (!canReadProjectPages(scope) || !captureProjectAccessScope(client, scope).isVerified())
        throw new Error("Project page access changed.");
      const saved = await service.update(scope.workspaceSlug, scope.projectId, pageId, change);
      if (saved.id !== pageId || !saved.project_ids?.includes(scope.projectId))
        throw new Error("The server returned a page outside the requested project.");
      return saved;
    },
    onSuccess: (saved, _change, context) => {
      if (context) commitPageChange(client, context.captured, pageId, saved, context.optimistic);
    },
    onError: (_error, _change, context) => {
      if (context) rollbackPageChange(client, context.captured, pageId, context.optimistic);
    },
  });
}

export function updateProjectPageDescriptionOptions(client: QueryClient, scope: TProjectPageScope, pageId: string) {
  return mutationOptions({
    mutationKey: pageKeys.mutations(scope, pageId),
    scope: {
      id: `pages/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}/${scope.projectId}/${pageId}`,
    },
    retry: false,
    onMutate: async (document: TDocumentPayload) => {
      if (!canReadProjectPages(scope)) throw new Error("Project page access is unavailable.");
      const captured = capturePageMutation(client, scope, pageId);
      if (!captured.access.isVerified()) throw new Error("Project page access is unavailable.");
      await cancelCapturedPages(client, captured);
      const optimistic = optimisticPageChange(client, captured, pageId, (page) => ({
        ...page,
        description_json: document.description_json,
        description_html: document.description_html,
      }));
      return { captured, optimistic };
    },
    mutationFn: async (document: TDocumentPayload) => {
      if (!canReadProjectPages(scope) || !captureProjectAccessScope(client, scope).isVerified())
        throw new Error("Project page access is unavailable.");
      await service.updateDescription(scope.workspaceSlug, scope.projectId, pageId, document);
    },
    onError: (_error, _document, context) => {
      if (context) rollbackPageChange(client, context.captured, pageId, context.optimistic);
    },
  });
}

export type TProjectPageAction = "lock" | "unlock" | "makePublic" | "makePrivate" | "archive" | "restore";

function applyPageAction(action: TProjectPageAction, page: TPage, saved?: { archived_at: string }) {
  if (action === "lock") return { ...page, is_locked: true };
  if (action === "unlock") return { ...page, is_locked: false };
  if (action === "makePublic") return { ...page, access: EPageAccess.PUBLIC };
  if (action === "makePrivate") return { ...page, access: EPageAccess.PRIVATE };
  if (action === "archive") return { ...page, archived_at: saved?.archived_at ?? new Date().toISOString() };
  return { ...page, archived_at: null };
}

export function projectPageActionOptions(client: QueryClient, scope: TProjectPageScope, pageId: string) {
  const mutationKey = pageKeys.mutations(scope, pageId);

  return mutationOptions({
    mutationKey,
    scope: {
      id: `pages/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}/${scope.projectId}/${pageId}`,
    },
    retry: false,
    onMutate: async (action: TProjectPageAction) => {
      if (!canReadProjectPages(scope)) throw new Error("Project page access is unavailable.");
      const captured = capturePageMutation(client, scope, pageId);
      if (!captured.access.isVerified()) throw new Error("Project page access is unavailable.");
      await cancelCapturedPages(client, captured);
      return {
        captured,
        optimistic: optimisticPageChange(client, captured, pageId, (page) => applyPageAction(action, page)),
      };
    },
    mutationFn: async (action: TProjectPageAction) => {
      if (!canReadProjectPages(scope) || !captureProjectAccessScope(client, scope).isVerified())
        throw new Error("Project page access changed.");
      if (action === "lock") await service.lock(scope.workspaceSlug, scope.projectId, pageId);
      else if (action === "unlock") await service.unlock(scope.workspaceSlug, scope.projectId, pageId);
      else if (action === "makePublic")
        await service.updateAccess(scope.workspaceSlug, scope.projectId, pageId, { access: EPageAccess.PUBLIC });
      else if (action === "makePrivate")
        await service.updateAccess(scope.workspaceSlug, scope.projectId, pageId, { access: EPageAccess.PRIVATE });
      else if (action === "archive") return service.archive(scope.workspaceSlug, scope.projectId, pageId);
      else await service.restore(scope.workspaceSlug, scope.projectId, pageId);
      return undefined;
    },
    onSuccess: (saved, action, context) => {
      if (!context || !captureIsCurrent(client, context.captured)) return;
      const page =
        client.getQueryData<TPage>(context.captured.detailKey) ??
        client.getQueryData<TPage[]>(context.captured.listKey)?.find((item) => item.id === pageId);
      if (
        page &&
        (page === context.optimistic?.optimisticDetail ||
          page === context.optimistic?.optimisticList ||
          !context.optimistic)
      ) {
        commitPageChange(client, context.captured, pageId, applyPageAction(action, page, saved));
      } else if (context.captured.detailQuery) {
        void client.invalidateQueries({ queryKey: context.captured.detailKey, exact: true, refetchType: "active" });
      }
    },
    onError: (_error, _variables, context) => {
      if (context) rollbackPageChange(client, context.captured, pageId, context.optimistic);
    },
  });
}

export function deleteProjectPageOptions(client: QueryClient, scope: TProjectPageScope, pageId: string) {
  return mutationOptions({
    mutationKey: pageKeys.mutations(scope, pageId),
    scope: {
      id: `pages/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}/${scope.projectId}/${pageId}`,
    },
    retry: false,
    onMutate: async () => {
      if (!canReadProjectPages(scope)) throw new Error("Project page access is unavailable.");
      return {
        captured: await preparePageMutation(client, scope, pageId),
        refreshFavorites: captureWorkspaceFavoritesRefresh(client, scope.workspaceSlug),
      };
    },
    mutationFn: async () => {
      if (!canReadProjectPages(scope) || !captureProjectAccessScope(client, scope).isVerified())
        throw new Error("Project page access is unavailable.");
      await service.remove(scope.workspaceSlug, scope.projectId, pageId);
    },
    onSuccess: async (_data, _variables, context) => {
      if (!context || !captureIsCurrent(client, context.captured)) return;
      if (queryIsCurrent(client, context.captured.detailKey, context.captured.detailQuery))
        client.removeQueries({ queryKey: context.captured.detailKey, exact: true });
      if (queryIsCurrent(client, context.captured.listKey, context.captured.listQuery))
        await client.invalidateQueries({ queryKey: context.captured.listKey, exact: true, refetchType: "active" });
      if (context.captured.access.isCurrent()) await Promise.allSettled([context.refreshFavorites()]);
    },
  });
}

export function duplicateProjectPageOptions(client: QueryClient, scope: TProjectPageScope, pageId: string) {
  return mutationOptions({
    mutationKey: pageKeys.mutations(scope, pageId),
    scope: {
      id: `pages/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}/${scope.projectId}/${pageId}`,
    },
    retry: false,
    onMutate: async () => ({ captured: await preparePageMutation(client, scope, pageId) }),
    mutationFn: async () => {
      if (!canReadProjectPages(scope) || !captureProjectAccessScope(client, scope).isVerified())
        throw new Error("Project page access is unavailable.");
      const saved = await service.duplicate(scope.workspaceSlug, scope.projectId, pageId);
      if (!saved.id || !saved.project_ids?.includes(scope.projectId))
        throw new Error("The server returned an invalid duplicate page.");
      return saved;
    },
    onSuccess: async (_saved, _variables, context) => {
      if (context) await refreshCapturedPageList(client, context.captured);
    },
  });
}

export function pageFilterList(
  pages: TPage[],
  projectId: string,
  pageType: "public" | "private" | "archived",
  filters: TPageFilters
) {
  return orderPages(
    filterPagesByPageType(pageType, pages).filter(
      (page) =>
        page.project_ids?.includes(projectId) &&
        getPageName(page.name).toLowerCase().includes(filters.searchQuery.toLowerCase()) &&
        shouldFilterPage(page, filters.filters)
    ),
    filters.sortKey,
    filters.sortBy
  );
}

export function updateProjectPageCache(
  client: QueryClient,
  scope: TProjectPageScope,
  pageId: string,
  update: (page: TPage) => TPage
) {
  if (!canReadProjectPages(scope) || !captureProjectAccessScope(client, scope).isCurrent()) return;
  const listKey = pageKeys.project(scope);
  const detailKey = pageKeys.detail(scope, pageId);
  client.setQueryData<TPage[]>(listKey, (pages) => pages?.map((page) => (page.id === pageId ? update(page) : page)));
  client.setQueryData<TPage>(detailKey, (page) => (page?.id === pageId ? update(page) : page));
}

export function removeProjectPageCache(client: QueryClient, scope: TProjectPageScope, pageId: string) {
  if (!canReadProjectPages(scope) || !captureProjectAccessScope(client, scope).isCurrent()) return;
  const listKey = pageKeys.project(scope);
  const detailKey = pageKeys.detail(scope, pageId);
  client.setQueryData<TPage[]>(listKey, (pages) => pages?.filter((page) => page.id !== pageId));
  client.removeQueries({ queryKey: detailKey, exact: true });
}

export function projectPageVersionsOptions(client: QueryClient, scope: TProjectPageScope, pageId: string | undefined) {
  return queryOptions({
    queryKey: [...pageKeys.detail(scope, pageId), "versions"],
    retry: false,
    queryFn:
      canReadProjectPages(scope) && pageId
        ? async ({ signal }) => {
            const access = captureProjectAccessScope(client, scope);
            if (!access.isVerified()) throw new Error("Project page access is unavailable.");
            const versions = await versionService.fetchAllVersions(
              scope.workspaceSlug,
              scope.projectId,
              pageId,
              signal
            );
            if (!access.isVerified()) throw new Error("Project page access changed while loading.");
            return versions;
          }
        : skipToken,
  });
}

export function projectPageVersionOptions(
  client: QueryClient,
  scope: TProjectPageScope,
  pageId: string,
  versionId: string | null
) {
  return queryOptions({
    queryKey: [...pageKeys.detail(scope, pageId), "version", versionId],
    retry: false,
    queryFn:
      canReadProjectPages(scope) && pageId && versionId
        ? async ({ signal }) => {
            const access = captureProjectAccessScope(client, scope);
            if (!access.isVerified()) throw new Error("Project page access is unavailable.");
            const version: TPageVersion = await versionService.fetchVersionById(
              scope.workspaceSlug,
              scope.projectId,
              pageId,
              versionId,
              signal
            );
            if (!access.isVerified()) throw new Error("Project page access changed while loading.");
            if (version.id !== versionId) throw new Error("The server returned a different page version.");
            return version;
          }
        : skipToken,
  });
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { infiniteQueryOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { hashKey } from "@tanstack/react-query";
import type { IssuePaginationOptions, TIssueParams, TIssuesResponse } from "@plane/types";
import { EIssueServiceType as IssueServiceType } from "@plane/types";
import type { ParentCatalogueScope } from "@/queries/cycle-module-scope";
import { parentScopeKey, parentScopeReady, captureParentAccess } from "@/queries/cycle-module-scope";
import type {
  TProjectIssueRequestInput,
  TWorkspaceIssueRequestInput,
  TProfileIssueRequestInput,
  TArchivedIssueRequestInput,
} from "@/helpers/issue-list/request";
import {
  getProjectIssueRequestParams,
  getProfileIssueRequestParams,
  getArchivedIssueRequestParams,
} from "@/helpers/issue-list/request";
import { getWorkspaceIssueRequestParams } from "@/helpers/issue-list/workspace-request";
import type { TParentIssueRequestInput } from "@/helpers/issue-list/parent-request";
import { parentIssueRequestParams } from "@/helpers/issue-list/parent-request";
import type { TProjectViewIssueRequestInput } from "@/helpers/issue-list/project-view-request";
import { getProjectViewIssueRequestParams } from "@/helpers/issue-list/project-view-request";
import { IssueService } from "@/services/issue";
import { IssueArchiveService } from "@/services/issue/issue_archive.service";
import { UserService } from "@/services/user.service";
import { WorkspaceService } from "@/services/workspace.service";
import { processIssueResponse } from "@/helpers/issue-list/response";

const issueListCacheGcTime = 5 * 60_000;

type ScopedRequest<TInput> = {
  input: TInput;
  /** Captured by the route/provider; true only while identity, access, and input revisions still match. */
  isReadCurrent: () => boolean;
};

export type IssueListRequest =
  | (ScopedRequest<TProjectIssueRequestInput> & { family: "project"; serviceType?: typeof IssueServiceType.ISSUES })
  | (ScopedRequest<TProjectIssueRequestInput> & { family: "epic"; serviceType: typeof IssueServiceType.EPICS })
  | (ScopedRequest<TWorkspaceIssueRequestInput> & { family: "workspace" })
  | (ScopedRequest<TParentIssueRequestInput & { kind: "cycle" }> & { family: "cycle" })
  | (ScopedRequest<TParentIssueRequestInput & { kind: "module" }> & { family: "module" })
  | (ScopedRequest<TProjectViewIssueRequestInput> & { family: "project-view" })
  | (ScopedRequest<TProfileIssueRequestInput> & { family: "profile" })
  | (ScopedRequest<TArchivedIssueRequestInput> & { family: "archived" });

export type IssueListRead = {
  scope: ParentCatalogueScope;
  request: IssueListRequest;
  pagination: IssuePaginationOptions;
  catalogueRevision?: string;
};
export type IssueListReadData = Readonly<{ generation: string; response: TIssuesResponse }>;
/** A response can contain flat groups and nested subgroups alongside one another. */
export type IssueListGroupedIds = Record<string, string[] | Record<string, string[]>>;

export const issueListKeys = {
  root: ["issue-list"] as const,
  list: ({ scope, request, pagination, catalogueRevision }: IssueListRead) =>
    [
      ...issueListKeys.root,
      parentScopeKey(scope),
      request.family,
      requestOwner(request),
      requestSignature(request),
      pagination,
      catalogueRevision,
    ] as const,
};

function requestOwner(request: IssueListRequest) {
  switch (request.family) {
    case "project":
    case "epic":
      return [request.input.workspaceSlug, request.input.projectId, request.input.projectIdentifier];
    case "workspace":
      return [request.input.workspaceSlug, request.input.viewId];
    case "cycle":
    case "module":
      return [request.input.workspaceSlug, request.input.projectId, request.input.viewId];
    case "project-view":
      return [request.input.workspaceSlug, request.input.projectId, request.input.viewId];
    case "profile":
      return [request.input.workspaceSlug, request.input.userId, request.input.view];
    case "archived":
      return [request.input.workspaceSlug, request.input.projectId];
    default:
      throw new Error("Unsupported issue list request family.");
  }
}

function requestSignature(request: IssueListRequest) {
  const { input } = request;
  return hashKey([input.richFilters, input.displayFilters]);
}

function isRequestCurrent(request: IssueListRequest, scope: ParentCatalogueScope) {
  if (!request.isReadCurrent()) return false;
  switch (request.family) {
    case "workspace":
      return isWorkspaceRequestCurrent(request, scope);
    case "cycle":
    case "module":
      return isParentRequestCurrent(request, scope);
    case "project-view":
      return isProjectScopedRequestCurrent(request.input, scope);
    case "profile":
      return isWorkspaceScopedRequestCurrent(request.input, scope);
    case "archived":
      return isProjectScopedRequestCurrent(request.input, scope);
    case "project":
    case "epic":
      return isProjectRequestCurrent(request.input, scope);
    default:
      return false;
  }
}

function isWorkspaceRequestCurrent(
  request: Extract<IssueListRequest, { family: "workspace" }>,
  scope: ParentCatalogueScope
) {
  return isWorkspaceScopedRequestCurrent(request.input, scope) && request.input.isReadScopeCurrent();
}

function isWorkspaceScopedRequestCurrent(
  input: { workspaceSlug: string; accountId: string },
  scope: ParentCatalogueScope
) {
  return !("projectId" in scope) && input.workspaceSlug === scope.workspaceSlug && input.accountId === scope.accountId;
}

function isProjectScopedRequestCurrent(
  input: { workspaceSlug: string; projectId: string; accountId: string; isReadScopeCurrent: () => boolean },
  scope: ParentCatalogueScope
) {
  return isProjectRequestCurrent(input, scope) && input.accountId === scope.accountId && input.isReadScopeCurrent();
}

function isProjectRequestCurrent(input: { workspaceSlug: string; projectId: string }, scope: ParentCatalogueScope) {
  return "projectId" in scope && input.workspaceSlug === scope.workspaceSlug && input.projectId === scope.projectId;
}

function isParentRequestCurrent(
  request: Extract<IssueListRequest, { family: "cycle" | "module" }>,
  scope: ParentCatalogueScope
) {
  return isProjectScopedRequestCurrent(request.input, scope) && request.input.kind === request.family;
}

function requestParams(
  request: IssueListRequest,
  pagination: IssuePaginationOptions,
  cursor?: string,
  groupId?: string,
  subGroupId?: string
): Partial<Record<TIssueParams, string | boolean>> | undefined {
  switch (request.family) {
    case "project":
    case "epic":
      return getProjectIssueRequestParams(request.input, pagination, cursor, groupId, subGroupId);
    case "workspace":
      return getWorkspaceIssueRequestParams(request.input, pagination, cursor, groupId, subGroupId);
    case "cycle":
    case "module":
      return parentIssueRequestParams(request.input, pagination, cursor, groupId, subGroupId);
    case "project-view":
      return getProjectViewIssueRequestParams(request.input, pagination, cursor, groupId, subGroupId);
    case "profile":
      return getProfileIssueRequestParams(request.input, pagination, cursor, groupId, subGroupId);
    case "archived":
      return getArchivedIssueRequestParams(request.input, pagination, cursor, groupId, subGroupId);
    default:
      return undefined;
  }
}

async function fetchIssueList(
  request: IssueListRequest,
  params: Partial<Record<TIssueParams, string | boolean>>,
  signal: AbortSignal
) {
  switch (request.family) {
    case "project":
    case "epic": {
      const { input } = request;
      return new IssueService(request.serviceType ?? IssueServiceType.ISSUES).getIssues(
        input.workspaceSlug,
        input.projectId,
        params,
        { signal }
      );
    }
    case "workspace": {
      const { input } = request;
      return new WorkspaceService().getViewIssues(input.workspaceSlug, params, { signal });
    }
    case "cycle":
    case "module": {
      const { input } = request;
      return new IssueService().getIssues(input.workspaceSlug, input.projectId, params, { signal });
    }
    case "project-view": {
      const { input } = request;
      return new IssueService().getIssues(input.workspaceSlug, input.projectId, params, { signal });
    }
    case "profile": {
      const { input } = request;
      return new UserService().getUserProfileIssues(input.workspaceSlug, input.userId, params, { signal });
    }
    case "archived": {
      const { input } = request;
      return new IssueArchiveService().getArchivedIssues(input.workspaceSlug, input.projectId, params, { signal });
    }
    default:
      throw new Error("Unsupported issue list request family.");
  }
}

function verifyQueryOwner(
  client: QueryClient,
  queryKey: readonly unknown[],
  query: object | undefined,
  signal: AbortSignal,
  request: IssueListRequest,
  scope: ParentCatalogueScope,
  access: ReturnType<typeof captureParentAccess>
) {
  return Boolean(
    !signal.aborted &&
    query &&
    client.getQueryCache().find({ queryKey, exact: true }) === query &&
    parentScopeReady(scope) &&
    access.isVerified() &&
    isRequestCurrent(request, scope)
  );
}

function validateListResponse(response: TIssuesResponse, scope: ParentCatalogueScope) {
  const allowedProjects =
    "projectId" in scope ? new Set([scope.projectId]) : new Set((scope.projectRoles ?? []).map(([id]) => id));
  const rows = processIssueResponse(response).issueList;
  if (rows.some((issue) => !issue.project_id || !allowedProjects.has(issue.project_id)))
    throw new Error("The work item list response did not match the authorized project scope.");
}

function selectedGroupResponse(
  response: TIssuesResponse,
  groupId: string | undefined,
  subGroupId: string | undefined,
  pagination: IssuePaginationOptions
): TIssuesResponse {
  if (Array.isArray(response.results)) return response;
  if (!groupId) throw new Error("The grouped work item response has no group identifier.");
  const group = response.results[groupId];
  if (!group) throw new Error("The work item group is missing from the accepted response.");
  const selected = subGroupId
    ? Array.isArray(group.results)
      ? undefined
      : group.results[subGroupId]
    : Array.isArray(group.results)
      ? { results: group.results, total_results: group.total_results }
      : undefined;
  if (!selected) throw new Error("The work item subgroup is missing from the accepted response.");
  const rows = selected.results;
  const nextPageResults = selected.total_results > rows.length;
  const cursor = `${pagination.perPageCount}:${Math.floor(rows.length / pagination.perPageCount)}:0`;
  const results = subGroupId
    ? {
        [groupId]: {
          total_results: group.total_results,
          results: { [subGroupId]: selected },
        },
      }
    : { [groupId]: selected };
  return {
    ...response,
    results,
    next_page_results: nextPageResults,
    next_cursor: cursor,
  };
}

export type IssueListGroup = Readonly<{ groupId?: string; subGroupId?: string }>;

/** Enumerate only group paths actually present in the accepted full read. */
export function issueListGroups(response: TIssuesResponse): IssueListGroup[] {
  if (Array.isArray(response.results)) return [{ groupId: undefined, subGroupId: undefined }];
  const groups: IssueListGroup[] = [];
  for (const [groupId, group] of Object.entries(response.results)) {
    if (Array.isArray(group.results)) {
      groups.push({ groupId, subGroupId: undefined });
      continue;
    }
    for (const subGroupId of Object.keys(group.results)) groups.push({ groupId, subGroupId });
  }
  return groups;
}

/** A guarded first-page read. The response and its records remain owned by this exact Query. */
export function issueListOptions(read: IssueListRead) {
  const queryKey = issueListKeys.list(read);
  const { scope, request, pagination } = read;
  return queryOptions<IssueListReadData>({
    queryKey,
    queryFn: parentScopeReady(scope)
      ? async ({ client, signal }) => {
          const query = client.getQueryCache().find({ queryKey, exact: true });
          const access = captureParentAccess(client, scope);
          const isCurrent = () => verifyQueryOwner(client, queryKey, query, signal, request, scope, access);
          if (!isCurrent()) throw new Error("Work item access or list preferences changed.");
          const params = requestParams(request, pagination);
          if (!params) throw new Error("Work item list parameters are unavailable.");
          const response: TIssuesResponse = await fetchIssueList(request, params, signal);
          if (!isCurrent()) throw new Error("Work item access or list preferences changed during the read.");
          validateListResponse(response, scope);
          return { generation: crypto.randomUUID(), response } satisfies IssueListReadData;
        }
      : skipToken,
    gcTime: issueListCacheGcTime,
    retry: false,
    refetchOnWindowFocus: false,
  });
}

/**
 * Each visible group gets its own infinite Query. The initial page is projected from the
 * full read, and `fullReadUpdatedAt` makes every replacement read a new pagination owner.
 */
export function issueListGroupOptions(read: IssueListRead, fullRead: IssueListReadData, group: IssueListGroup) {
  const fullQueryKey = issueListKeys.list(read);
  const queryKey = [
    ...fullQueryKey,
    "group-pages",
    fullRead.generation,
    group.groupId ?? null,
    group.subGroupId ?? null,
  ] as const;
  const seed = selectedGroupResponse(fullRead.response, group.groupId, group.subGroupId, read.pagination);
  return infiniteQueryOptions({
    queryKey,
    initialPageParam: "initial",
    initialData: { pages: [seed], pageParams: ["initial"] },
    staleTime: Number.POSITIVE_INFINITY,
    gcTime: issueListCacheGcTime,
    retry: false,
    refetchOnMount: false,
    queryFn: async ({ client, signal, pageParam }) => {
      const fullQuery = client.getQueryCache().find<IssueListReadData>({ queryKey: fullQueryKey, exact: true });
      const query = client.getQueryCache().find({ queryKey, exact: true });
      const access = captureParentAccess(client, read.scope);
      const currentFullRead = fullQuery?.state.data;
      const isCurrent = () =>
        verifyQueryOwner(client, queryKey, query, signal, read.request, read.scope, access) &&
        Boolean(
          fullQuery &&
          client.getQueryCache().find({ queryKey: fullQueryKey, exact: true }) === fullQuery &&
          fullQuery.state.data?.generation === fullRead.generation
        );
      if (!currentFullRead || !isCurrent()) throw new Error("The work item list changed before pagination.");
      if (pageParam === "initial")
        return selectedGroupResponse(currentFullRead.response, group.groupId, group.subGroupId, read.pagination);
      const params = requestParams(read.request, read.pagination, pageParam, group.groupId, group.subGroupId);
      if (!params) throw new Error("Work item pagination parameters are unavailable.");
      const response = await fetchIssueList(read.request, params, signal);
      if (!isCurrent()) throw new Error("Work item access or list changed during pagination.");
      validateListResponse(response, read.scope);
      return response;
    },
    getNextPageParam: (lastPage, pages) => {
      if (!lastPage.next_page_results) return undefined;
      const loadedCount = pages.reduce((total, page) => total + processIssueResponse(page).issueList.length, 0);
      const nextPage = Math.floor(loadedCount / read.pagination.perPageCount);
      return `${read.pagination.perPageCount}:${nextPage}:0`;
    },
  });
}

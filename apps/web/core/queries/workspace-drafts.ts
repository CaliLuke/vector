/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { infiniteQueryOptions, mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { InfiniteData, Query, QueryClient } from "@tanstack/react-query";
import type { TIssue, TWorkspaceDraftIssue, TWorkspaceDraftPaginationInfo } from "@plane/types";
import type { EUserWorkspaceRoles } from "@plane/types";
import { captureProjectAccessScope } from "./project-access-scope";
import { capturePermissionChanges } from "./permission-mutations";
import { getIdentitySessionGeneration } from "./identity";
import { WorkspaceDraftService } from "@/services/issue/workspace_draft.service";

const service = new WorkspaceDraftService();
const PAGE_SIZE = 50;

export type WorkspaceDraftScope = {
  accountId: string | undefined;
  sessionGeneration: number;
  workspaceSlug: string | undefined;
  membershipRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
};

export type WorkspaceDraftWrite =
  | { kind: "create"; data: Partial<TWorkspaceDraftIssue | TIssue> }
  | { kind: "update"; issueId: string; data: Partial<TWorkspaceDraftIssue | TIssue> }
  | { kind: "delete"; issueId: string };

type DraftReceipt = { kind: "create" | "update"; saved?: TWorkspaceDraftIssue } | { kind: "delete"; issueId: string };

export const workspaceDraftKeys = {
  scope: (scope: WorkspaceDraftScope) =>
    [
      "workspace-drafts",
      scope.accountId,
      scope.sessionGeneration,
      scope.workspaceSlug,
      scope.membershipRole,
      scope.membershipRevision,
    ] as const,
  list: (scope: WorkspaceDraftScope) => [...workspaceDraftKeys.scope(scope), "list"] as const,
  detail: (scope: WorkspaceDraftScope, issueId: string | undefined) =>
    [...workspaceDraftKeys.scope(scope), "detail", issueId] as const,
  writes: (scope: WorkspaceDraftScope) => [...workspaceDraftKeys.scope(scope), "write"] as const,
};

function scopeReady(scope: WorkspaceDraftScope): scope is WorkspaceDraftScope & {
  accountId: string;
  workspaceSlug: string;
  membershipRole: EUserWorkspaceRoles;
  membershipRevision: string;
} {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.sessionGeneration >= 0 &&
    scope.membershipRole !== undefined &&
    scope.membershipRevision
  );
}

function captureAccess(client: QueryClient, scope: WorkspaceDraftScope) {
  const captured = captureProjectAccessScope(client, {
    accountId: scope.accountId,
    sessionGeneration: scope.sessionGeneration,
    workspaceSlug: scope.workspaceSlug,
    projectId: undefined,
    projectRole: undefined,
    membershipRole: scope.membershipRole,
    membershipRevision: scope.membershipRevision,
  });
  return {
    isCurrent: () =>
      scopeReady(scope) && scope.sessionGeneration === getIdentitySessionGeneration(client) && captured.isCurrent(),
    isVerified: () =>
      scopeReady(scope) && scope.sessionGeneration === getIdentitySessionGeneration(client) && captured.isVerified(),
  };
}

function exactQuery(client: QueryClient, key: readonly unknown[]) {
  return client.getQueryCache().find({ queryKey: key, exact: true });
}

function assertPrivateDraft(issue: TWorkspaceDraftIssue | null, scope: WorkspaceDraftScope, issueId?: string) {
  if (!scopeReady(scope) || issue?.created_by !== scope.accountId || (issueId !== undefined && issue.id !== issueId))
    throw new Error("The server returned a draft outside the current account.");
}

function capturedRead(client: QueryClient, scope: WorkspaceDraftScope, key: readonly unknown[], signal: AbortSignal) {
  const access = captureAccess(client, scope);
  const query = exactQuery(client, key);
  const isCurrent = () =>
    !signal.aborted && query !== undefined && access.isVerified() && exactQuery(client, key) === query;
  if (!isCurrent()) throw new Error("Workspace draft access is unavailable.");
  return isCurrent;
}

export function workspaceDraftListOptions(client: QueryClient, scope: WorkspaceDraftScope, enabled = true) {
  const key = workspaceDraftKeys.list(scope);
  const canRead = scopeReady(scope);
  return infiniteQueryOptions({
    queryKey: key,
    initialPageParam: `${PAGE_SIZE}:0:0`,
    staleTime: 30_000,
    retry: false,
    queryFn:
      enabled && canRead
        ? async ({ pageParam, signal }) => {
            const isCurrent = capturedRead(client, scope, key, signal);
            const page = await service.getIssues(
              scope.workspaceSlug,
              { per_page: PAGE_SIZE, cursor: pageParam },
              signal
            );
            if (!isCurrent()) throw new Error("Workspace draft access changed while loading.");
            if (!page) throw new Error("Workspace drafts could not be loaded.");
            for (const row of page.results ?? []) assertPrivateDraft(row, scope);
            return page;
          }
        : skipToken,
    getNextPageParam: (page: TWorkspaceDraftPaginationInfo<TWorkspaceDraftIssue>) =>
      page.next_page_results ? page.next_cursor || undefined : undefined,
  });
}

export function workspaceDraftDetailOptions(
  client: QueryClient,
  scope: WorkspaceDraftScope,
  issueId: string | undefined,
  enabled = true
) {
  const key = workspaceDraftKeys.detail(scope, issueId);
  return queryOptions({
    queryKey: key,
    staleTime: 30_000,
    retry: false,
    queryFn:
      enabled && scopeReady(scope) && issueId
        ? async ({ signal }): Promise<TWorkspaceDraftIssue | null> => {
            const isCurrent = capturedRead(client, scope, key, signal);
            const issue = await service.getIssueById(scope.workspaceSlug, issueId, signal);
            if (!isCurrent()) throw new Error("Workspace draft access changed while loading.");
            if (!issue) throw new Error("Draft not found.");
            assertPrivateDraft(issue, scope, issueId);
            return issue;
          }
        : skipToken,
  });
}

function queryStillOwned(client: QueryClient, query: Query) {
  return client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) === query;
}

function captureDraftQueries(client: QueryClient, scope: WorkspaceDraftScope, issueId?: string) {
  const access = captureAccess(client, scope);
  const scopedQueries = client.getQueryCache().findAll({ queryKey: workspaceDraftKeys.scope(scope) });
  const queries = issueId
    ? scopedQueries.filter(
        (query) =>
          query.queryKey.at(-1) === "list" || (query.queryKey.at(-2) === "detail" && query.queryKey.at(-1) === issueId)
      )
    : scopedQueries;
  const querySlots = [
    workspaceDraftKeys.list(scope),
    ...(issueId ? [workspaceDraftKeys.detail(scope, issueId)] : []),
  ].map((queryKey) => ({
    queryKey,
    query: exactQuery(client, queryKey),
  }));
  return {
    queries,
    isCurrent: () =>
      access.isCurrent() &&
      queries.every((query) => queryStillOwned(client, query)) &&
      querySlots.every(({ query, queryKey }) => exactQuery(client, queryKey) === query),
  };
}

type WorkspaceDraftReceiptOwner = {
  accountId: string;
  captured: ReturnType<typeof captureDraftQueries>;
  permissionChanges: ReturnType<typeof capturePermissionChanges>;
  isCurrent: () => boolean;
};

export function captureWorkspaceDraftReceiptOwner(
  client: QueryClient,
  scope: WorkspaceDraftScope,
  issueId: string
): WorkspaceDraftReceiptOwner | undefined {
  const access = captureAccess(client, scope);
  if (!scopeReady(scope) || !access.isVerified()) return undefined;
  const captured = captureDraftQueries(client, scope, issueId);
  const permissionChanges = capturePermissionChanges(client, scope.accountId, scope.workspaceSlug);
  return {
    accountId: scope.accountId,
    captured,
    permissionChanges,
    isCurrent: () => access.isCurrent() && captured.isCurrent(),
  };
}

async function requireOwnedDraft(client: QueryClient, scope: WorkspaceDraftScope, issueId: string) {
  const cachedDetail = client.getQueryData<TWorkspaceDraftIssue>(workspaceDraftKeys.detail(scope, issueId));
  if (cachedDetail) {
    assertPrivateDraft(cachedDetail, scope, issueId);
    return;
  }
  const listQueries = client.getQueryCache().findAll({ queryKey: workspaceDraftKeys.list(scope), exact: true });
  for (const query of listQueries) {
    const cached = client.getQueryData<InfiniteData<TWorkspaceDraftPaginationInfo<TWorkspaceDraftIssue>, string>>(
      query.queryKey
    );
    const row = cached?.pages.flatMap((page) => page.results ?? []).find((item) => item.id === issueId);
    if (row) {
      assertPrivateDraft(row, scope, issueId);
      return;
    }
  }
  const canonical = await client.fetchQuery(workspaceDraftDetailOptions(client, scope, issueId));
  assertPrivateDraft(canonical, scope, issueId);
}

function reconcileDraftReceipt(
  client: QueryClient,
  captured: ReturnType<typeof captureDraftQueries>,
  receipt: DraftReceipt
) {
  if (!captured.isCurrent()) return;
  for (const query of captured.queries) {
    if (!captured.isCurrent()) return;
    if (query.queryKey.at(-1) === "list") {
      client.setQueryData<InfiniteData<TWorkspaceDraftPaginationInfo<TWorkspaceDraftIssue>, string>>(
        query.queryKey,
        (current) => {
          if (!current?.pages?.length) return current;
          const pages = current.pages.map((page) => ({
            ...page,
            results:
              receipt.kind === "delete"
                ? (page.results ?? []).filter((row) => row.id !== receipt.issueId)
                : receipt.kind === "create" && receipt.saved
                  ? (page.results ?? []).filter((row) => row.id !== receipt.saved?.id)
                  : (page.results ?? []),
          }));
          if (receipt.kind === "create" && receipt.saved) {
            const first = pages[0];
            pages[0] = {
              ...first,
              results: [receipt.saved, ...(first.results ?? []).filter((row) => row.id !== receipt.saved?.id)],
            };
          } else if (receipt.kind === "update" && receipt.saved) {
            const saved = receipt.saved;
            for (const page of pages) {
              page.results = (page.results ?? []).map((row) => (row.id === saved.id ? saved : row));
            }
          }
          return { ...current, pages };
        }
      );
    } else if (query.queryKey.at(-2) === "detail") {
      const issueId = query.queryKey.at(-1);
      if (receipt.kind === "delete" && issueId === receipt.issueId)
        client.setQueryData<TWorkspaceDraftIssue | null>(query.queryKey, null);
      if ((receipt.kind === "create" || receipt.kind === "update") && receipt.saved?.id === issueId)
        client.setQueryData(query.queryKey, receipt.saved);
    }
  }
}

async function refreshDraftLists(client: QueryClient, owner: WorkspaceDraftReceiptOwner) {
  await Promise.allSettled(
    owner.captured.queries.map(async (query) => {
      if (!owner.isCurrent() || query.queryKey.at(-1) !== "list") return;
      // List counts are canonical reads; a failed refresh cannot reject an accepted write.
      await client.invalidateQueries({ queryKey: query.queryKey, exact: true });
    })
  );
}

async function refreshDraftPermissions(owner: WorkspaceDraftReceiptOwner) {
  if (!owner.isCurrent()) return;
  await Promise.allSettled([owner.permissionChanges.refreshWorkspace()]);
}

async function cancelReceiptQueries(client: QueryClient, owner: WorkspaceDraftReceiptOwner) {
  await Promise.allSettled(
    owner.captured.queries.map(async (query) => {
      if (!owner.isCurrent()) return;
      await client.cancelQueries({ queryKey: query.queryKey, exact: true });
    })
  );
}

export function workspaceDraftMutationOptions(client: QueryClient, scope: WorkspaceDraftScope) {
  return mutationOptions({
    mutationKey: workspaceDraftKeys.writes(scope),
    scope: { id: JSON.stringify(workspaceDraftKeys.writes(scope)) },
    retry: false,
    onMutate: async (input: WorkspaceDraftWrite) => {
      const access = captureAccess(client, scope);
      if (!scopeReady(scope) || !access.isVerified()) throw new Error("Workspace draft access is unavailable.");
      if (input.kind !== "create") await requireOwnedDraft(client, scope, input.issueId);
      if (!access.isVerified()) throw new Error("Workspace draft access changed before saving.");
      const permissionChanges = capturePermissionChanges(client, scope.accountId, scope.workspaceSlug);
      const captured = captureDraftQueries(client, scope);
      await Promise.all(
        captured.queries.map(async (query) => {
          if (!captured.isCurrent()) throw new Error("Workspace draft access changed before saving.");
          await client.cancelQueries({ queryKey: query.queryKey, exact: true });
        })
      );
      if (!captured.isCurrent()) throw new Error("Workspace draft access changed before saving.");
      return {
        accountId: scope.accountId,
        captured,
        permissionChanges,
        isCurrent: () => access.isCurrent() && captured.isCurrent(),
      } satisfies WorkspaceDraftReceiptOwner;
    },
    mutationFn: async (input: WorkspaceDraftWrite): Promise<DraftReceipt> => {
      const access = captureAccess(client, scope);
      if (!access.isVerified() || !scopeReady(scope)) throw new Error("Workspace draft access is unavailable.");
      if (input.kind === "create") {
        const saved = await service.createIssue(scope.workspaceSlug, input.data);
        if (saved) assertPrivateDraft(saved, scope);
        return { kind: input.kind, saved };
      }
      if (input.kind === "update") {
        const saved = await service.updateIssue(scope.workspaceSlug, input.issueId, input.data);
        if (saved) assertPrivateDraft(saved, scope, input.issueId);
        return { kind: input.kind, saved };
      }
      await service.deleteIssue(scope.workspaceSlug, input.issueId);
      return { kind: input.kind, issueId: input.issueId };
    },
    onSuccess: async (receipt, input, owner) => {
      if (!owner?.isCurrent()) return;
      await cancelReceiptQueries(client, owner);
      if (!owner.isCurrent()) return;
      reconcileDraftReceipt(client, owner.captured, receipt);
      if (input.kind === "create" || input.kind === "delete")
        await Promise.all([refreshDraftLists(client, owner), refreshDraftPermissions(owner)]);
    },
  });
}

export async function removePublishedWorkspaceDraft(
  client: QueryClient,
  owner: WorkspaceDraftReceiptOwner | undefined,
  issueId: string
) {
  if (!owner?.isCurrent()) return;
  await cancelReceiptQueries(client, owner);
  if (!owner.isCurrent()) return;
  reconcileDraftReceipt(client, owner.captured, { kind: "delete", issueId });
  await Promise.all([refreshDraftPermissions(owner), refreshDraftLists(client, owner)]);
}

export async function acceptWorkspaceDraftReceipt(
  client: QueryClient,
  owner: WorkspaceDraftReceiptOwner | undefined,
  issue: TWorkspaceDraftIssue
) {
  if (!owner?.isCurrent() || issue.created_by !== owner.accountId) return;
  await cancelReceiptQueries(client, owner);
  if (!owner.isCurrent()) return;
  reconcileDraftReceipt(client, owner.captured, { kind: "update", saved: issue });
  await refreshDraftLists(client, owner);
}

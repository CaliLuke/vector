/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { InfiniteData, Query, QueryClient } from "@tanstack/react-query";
import { mutationOptions } from "@tanstack/react-query";
import { EIssueServiceType } from "@plane/types";
import type { TBaseIssue, TIssue, TIssueServiceType, TIssuesResponse } from "@plane/types";
import { ModuleService } from "@/services/module.service";
import { IssueService } from "@/services/issue/issue.service";
import { IssueArchiveService } from "@/services/issue/issue_archive.service";
import {
  currentParentCatalogueScope,
  captureParentAccess,
  parentScopeKey,
  parentScopeReady,
} from "@/queries/cycle-module-scope";
import type { ProjectCatalogueScope } from "@/queries/catalogue-scope";
import type { ParentCatalogueScope } from "@/queries/cycle-module-scope";
import { issueListKeys } from "@/queries/issue-lists";
import { getCurrentUserId, getIdentitySessionGeneration } from "@/queries/identity";
import type { IssueListReadData } from "@/queries/issue-lists";

export type IssueListMutationOwner = Readonly<{
  scope: ProjectCatalogueScope;
  listScopes: readonly ParentCatalogueScope[];
  serviceType: TIssueServiceType;
  isScopeCurrent: () => boolean;
  parentStatistics?: {
    isCurrent: () => boolean;
    refresh?: () => Promise<void>;
    refreshAssignments?: (domain: "cycle" | "module", parentIds: readonly string[]) => Promise<void>;
  };
}>;

export type IssueListMutation =
  | { kind: "create"; data: Partial<TIssue> }
  | { kind: "patch"; issueId: string; data: Partial<TIssue> }
  | { kind: "delete"; issueId: string }
  | { kind: "delete-many"; issueIds: string[] }
  | { kind: "archive"; issueId: string }
  | { kind: "restore"; issueId: string }
  | { kind: "dates"; updates: { id: string; start_date?: string; target_date?: string }[] }
  | { kind: "cycle-add"; cycleId: string; issueIds: string[] }
  | { kind: "cycle-remove"; cycleId: string; issueId: string }
  | { kind: "module-remove"; moduleId: string; issueId: string }
  | { kind: "module-add"; moduleId: string; issueIds: string[] }
  | { kind: "module-change"; issueId: string; added: string[]; removed: string[] };

export function issueListMutationOptions(client: QueryClient) {
  return mutationOptions({
    mutationKey: ["issue-list", "write"],
    retry: false,
    mutationFn: ({ owner, mutation }: { owner: IssueListMutationOwner; mutation: IssueListMutation }) =>
      issueListMutation(client, owner, mutation),
  });
}

export type CapturedIssueListMutationOwner = {
  access: ReturnType<typeof captureParentAccess>;
  queries: Set<Query>;
  listScopes: readonly ParentCatalogueScope[];
  isAuthorizedCurrent: () => boolean;
  isCurrent: () => boolean;
};

function sameListScope(query: Query, scope: ParentCatalogueScope) {
  return (
    query.queryKey[0] === issueListKeys.root[0] &&
    query.queryKey[1] !== undefined &&
    JSON.stringify(query.queryKey[1]) === JSON.stringify(parentScopeKey(scope)) &&
    (query.queryKey.length === 7 || (query.queryKey.length === 11 && query.queryKey[7] === "group-pages"))
  );
}

export function captureIssueListMutationOwner(
  client: QueryClient,
  owner: IssueListMutationOwner
): CapturedIssueListMutationOwner {
  const { scope } = owner;
  const access = captureParentAccess(client, scope);
  const queries = new Set(
    owner.listScopes.flatMap((listScope) =>
      client
        .getQueryCache()
        .findAll({ queryKey: [issueListKeys.root[0], parentScopeKey(listScope)] })
        .filter((query) => sameListScope(query, listScope))
    )
  );
  const isAuthorizedCurrent = () =>
    parentScopeReady(scope) &&
    access.isCurrent() &&
    getCurrentUserId(client) === scope.accountId &&
    getIdentitySessionGeneration(client) === scope.sessionGeneration;
  const isCurrent = () => isAuthorizedCurrent() && owner.isScopeCurrent();
  return { access, queries, listScopes: owner.listScopes, isAuthorizedCurrent, isCurrent };
}

async function reconcileAcceptedWrite(
  client: QueryClient,
  owner: IssueListMutationOwner,
  captured: CapturedIssueListMutationOwner,
  mutation: IssueListMutation,
  receipt?: TIssue
) {
  await refreshCapturedIssueLists(client, captured, receipt);
  const parentStats = owner.parentStatistics;
  if (!parentStats?.isCurrent()) return;
  const assignments = changedAssignments(mutation);
  if (assignments && parentStats.refreshAssignments) {
    void parentStats
      .refreshAssignments(assignments.domain, assignments.ids)
      .catch((error) => console.error("Could not refresh work item parent assignments", error));
  }
  if (parentStats.refresh) {
    void parentStats.refresh().catch((error) => console.error("Could not refresh parent work item statistics", error));
  }
}

/** Recount list queries captured before another domain accepted a canonical issue write. */
function replaceIssueRows(rows: readonly TBaseIssue[], receipt: TIssue) {
  return rows.map((row) => (row.id === receipt.id ? { ...row, ...receipt } : row));
}

function replaceIssueInResponse(response: TIssuesResponse, receipt: TIssue): TIssuesResponse {
  if (Array.isArray(response.results)) return { ...response, results: replaceIssueRows(response.results, receipt) };
  const results = Object.fromEntries(
    Object.entries(response.results).map(([groupId, group]) => [
      groupId,
      Array.isArray(group.results)
        ? { ...group, results: replaceIssueRows(group.results, receipt) }
        : {
            ...group,
            results: Object.fromEntries(
              Object.entries(group.results).map(([subGroupId, subgroup]) => [
                subGroupId,
                { ...subgroup, results: replaceIssueRows(subgroup.results, receipt) },
              ])
            ),
          },
    ])
  ) as TIssuesResponse["results"];
  return { ...response, results };
}

function reconcileIssueReceipt(client: QueryClient, captured: CapturedIssueListMutationOwner, receipt: TIssue) {
  for (const query of captured.queries) {
    if (client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) !== query) continue;
    if (query.queryKey.length === 7) {
      const current = client.getQueryData<IssueListReadData>(query.queryKey);
      if (current)
        client.setQueryData<IssueListReadData>(query.queryKey, {
          ...current,
          response: replaceIssueInResponse(current.response, receipt),
        });
      continue;
    }
    const current = client.getQueryData<InfiniteData<TIssuesResponse, string>>(query.queryKey);
    if (current)
      client.setQueryData<InfiniteData<TIssuesResponse, string>>(query.queryKey, {
        ...current,
        pages: current.pages.map((page) => replaceIssueInResponse(page, receipt)),
      });
  }
}

export async function refreshCapturedIssueLists(
  client: QueryClient,
  captured: CapturedIssueListMutationOwner,
  receipt?: TIssue
) {
  try {
    if (!captured.isAuthorizedCurrent()) return;
    const queriesToRefresh = new Set(captured.queries);
    if (captured.queries.size === 0 || !captured.isCurrent()) {
      for (const scope of captured.listScopes) {
        for (const query of client
          .getQueryCache()
          .findAll({ queryKey: [issueListKeys.root[0], parentScopeKey(scope)] })) {
          if (sameListScope(query, scope)) queriesToRefresh.add(query);
        }
      }
    }
    await client.cancelQueries({ predicate: (query) => queriesToRefresh.has(query) });
    if (captured.isAuthorizedCurrent()) {
      if (receipt) reconcileIssueReceipt(client, captured, receipt);
      await client.invalidateQueries({
        predicate: (query) => queriesToRefresh.has(query),
        refetchType: captured.access.isVerified() ? "active" : "none",
      });
    }
  } catch (error) {
    console.error("Could not refresh the work item list after an accepted write", error);
  }
}

function changedAssignments(mutation: IssueListMutation) {
  if (mutation.kind === "cycle-add")
    return { domain: "cycle" as const, ids: mutation.issueIds.length ? [mutation.cycleId] : [] };
  if (mutation.kind === "cycle-remove") return { domain: "cycle" as const, ids: [mutation.cycleId] };
  if (mutation.kind === "module-remove") return { domain: "module" as const, ids: [mutation.moduleId] };
  if (mutation.kind === "module-add") return { domain: "module" as const, ids: [mutation.moduleId] };
  if (mutation.kind === "module-change")
    return { domain: "module" as const, ids: [...mutation.added, ...mutation.removed] };
  if (mutation.kind === "patch" && typeof mutation.data.cycle_id === "string")
    return { domain: "cycle" as const, ids: [mutation.data.cycle_id] };
  if (mutation.kind === "patch" && Array.isArray(mutation.data.module_ids))
    return { domain: "module" as const, ids: mutation.data.module_ids };
  return undefined;
}

function assertWriteReady(
  client: QueryClient,
  owner: IssueListMutationOwner,
  captured: CapturedIssueListMutationOwner
) {
  const { scope } = owner;
  if (
    !scope.workspaceSlug ||
    !scope.projectId ||
    !captured.access.isVerified() ||
    !owner.isScopeCurrent() ||
    getCurrentUserId(client) !== scope.accountId ||
    getIdentitySessionGeneration(client) !== scope.sessionGeneration
  )
    throw new Error("Work item access changed. Retry the operation.");
}

function validateIssueReceipt(issue: TIssue | undefined, owner: IssueListMutationOwner, issueId?: string): TIssue {
  if (!issue?.id || issue.project_id !== owner.scope.projectId || (issueId !== undefined && issue.id !== issueId))
    throw new Error("The saved work item did not match this project and operation.");
  return issue;
}

/** An accepted PATCH may return a canonical row or HTTP 204; both invalidate captured readers. */
export async function patchIssueMutation(
  client: QueryClient,
  owner: IssueListMutationOwner,
  issueId: string,
  data: Partial<TIssue>
): Promise<TIssue | undefined> {
  const captured = captureIssueListMutationOwner(client, owner);
  assertWriteReady(client, owner, captured);
  const service = new IssueService(owner.serviceType);
  const response = await service.patchIssue(owner.scope.workspaceSlug!, owner.scope.projectId!, issueId, data);
  const receipt = response === undefined ? undefined : validateIssueReceipt(response, owner, issueId);
  await reconcileAcceptedWrite(client, owner, captured, { kind: "patch", issueId, data }, receipt);
  return receipt;
}

/** Execute an issue-list write against one captured account, permission, request, and list-query lineage. */
export async function issueListMutation(
  client: QueryClient,
  owner: IssueListMutationOwner,
  mutation: IssueListMutation
): Promise<TIssue | undefined> {
  if (isNoopMutation(mutation)) return undefined;
  const captured = captureIssueListMutationOwner(client, owner);
  assertWriteReady(client, owner, captured);
  const { workspaceSlug, projectId } = owner.scope;
  const service = new IssueService(owner.serviceType);
  let receipt: TIssue | undefined;
  if (isAssignmentMutation(mutation)) await executeAssignmentMutation(workspaceSlug!, projectId!, service, mutation);
  else receipt = await executeEntityMutation(workspaceSlug!, projectId!, owner, service, mutation);
  await reconcileAcceptedWrite(client, owner, captured, mutation, receipt);
  return receipt;
}

type AssignmentMutation = Extract<
  IssueListMutation,
  { kind: "cycle-add" | "cycle-remove" | "module-remove" | "module-add" | "module-change" }
>;

type EntityMutation = Exclude<IssueListMutation, AssignmentMutation>;

function isAssignmentMutation(mutation: IssueListMutation): mutation is AssignmentMutation {
  return (
    mutation.kind === "cycle-add" ||
    mutation.kind === "cycle-remove" ||
    mutation.kind === "module-remove" ||
    mutation.kind === "module-add" ||
    mutation.kind === "module-change"
  );
}

function isNoopMutation(mutation: IssueListMutation) {
  return (
    (mutation.kind === "delete-many" && mutation.issueIds.length === 0) ||
    (mutation.kind === "module-change" && mutation.added.length === 0 && mutation.removed.length === 0)
  );
}

async function executeEntityMutation(
  workspaceSlug: string,
  projectId: string,
  owner: IssueListMutationOwner,
  service: IssueService,
  mutation: EntityMutation
): Promise<TIssue | undefined> {
  switch (mutation.kind) {
    case "create":
      return validateIssueReceipt(await service.createIssue(workspaceSlug, projectId, mutation.data), owner);
    case "patch": {
      const response = await service.patchIssue(workspaceSlug, projectId, mutation.issueId, mutation.data);
      return response === undefined ? undefined : validateIssueReceipt(response, owner, mutation.issueId);
    }
    case "delete":
      await service.deleteIssue(workspaceSlug, projectId, mutation.issueId);
      return undefined;
    case "delete-many":
      await service.bulkDeleteIssues(workspaceSlug, projectId, { issue_ids: mutation.issueIds });
      return undefined;
    case "archive":
      await new IssueArchiveService(owner.serviceType).archiveIssue(workspaceSlug, projectId, mutation.issueId);
      return undefined;
    case "restore":
      await new IssueArchiveService(owner.serviceType).restoreIssue(workspaceSlug, projectId, mutation.issueId);
      return undefined;
    case "dates":
      await service.updateIssueDates(workspaceSlug, projectId, mutation.updates);
      return undefined;
    default:
      throw new Error("Unsupported issue list mutation.");
  }
}

async function executeAssignmentMutation(
  workspaceSlug: string,
  projectId: string,
  service: IssueService,
  mutation: AssignmentMutation
): Promise<void> {
  switch (mutation.kind) {
    case "cycle-add":
      await service.addIssueToCycle(workspaceSlug, projectId, mutation.cycleId, { issues: mutation.issueIds });
      return;
    case "cycle-remove":
      await service.removeIssueFromCycle(workspaceSlug, projectId, mutation.cycleId, mutation.issueId);
      return;
    case "module-remove":
      await new ModuleService().removeIssuesFromModuleBulk(workspaceSlug, projectId, mutation.moduleId, [
        mutation.issueId,
      ]);
      return;
    case "module-add":
      await new ModuleService().addIssuesToModule(workspaceSlug, projectId, mutation.moduleId, {
        issues: mutation.issueIds,
      });
      return;
    case "module-change":
      await new ModuleService().addModulesToIssue(workspaceSlug, projectId, mutation.issueId, {
        modules: mutation.added,
        removed_modules: mutation.removed,
      });
  }
}

/** Build the current authorized target-project scope for a list or detail mutation. */
export function currentIssueListMutationOwner(
  client: QueryClient,
  workspaceSlug: string,
  projectId: string,
  serviceType: TIssueServiceType,
  isScopeCurrent: () => boolean,
  parentStatistics?: IssueListMutationOwner["parentStatistics"]
): IssueListMutationOwner {
  const current = currentParentCatalogueScope(client, workspaceSlug, projectId);
  return {
    scope: current.project,
    listScopes: [current.project, current.workspace],
    serviceType: serviceType ?? EIssueServiceType.ISSUES,
    isScopeCurrent,
    parentStatistics,
  };
}

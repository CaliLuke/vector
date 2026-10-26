/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import type { ICycle, IModule } from "@plane/types";
import { CycleService } from "@/services/cycle.service";
import { ModuleService } from "@/services/module.service";
import { CycleArchiveService } from "@/services/cycle_archive.service";
import { ModuleArchiveService } from "@/services/module_archive.service";
import { isAuthorizationError } from "./authorization-errors";
import { captureWorkspaceFavoritesRefresh } from "./favorites";
import { captureParentAccess, parentScopeKey, parentScopeReady } from "./cycle-module-scope";
import type { ParentCatalogueScope } from "./cycle-module-scope";
import type { ProjectCatalogueScope, WorkspaceCatalogueScope } from "./catalogue-scope";

type Parent = ICycle | IModule;
export type ParentDomain = "cycle" | "module";
export type ParentCatalogueCollection = "active" | "archived" | "universe";
export const parentCatalogueKeys = {
  owner: (domain: ParentDomain, scope: ParentCatalogueScope) =>
    ["cycle-module", domain, scope.accountId, scope.sessionGeneration, scope.workspaceSlug] as const,
  scope: (domain: ParentDomain, scope: ParentCatalogueScope) =>
    [
      ...parentCatalogueKeys.owner(domain, scope),
      "projectId" in scope ? scope.projectId : "workspace",
      ...parentScopeKey(scope).slice(3),
    ] as const,
  list: (domain: ParentDomain, scope: ParentCatalogueScope, collection: ParentCatalogueCollection = "active") =>
    [...parentCatalogueKeys.scope(domain, scope), "list", collection] as const,
  detail: (
    domain: ParentDomain,
    scope: ProjectCatalogueScope,
    id: string | undefined,
    archived = false,
    statisticsRevision?: string
  ) => [...parentCatalogueKeys.scope(domain, scope), statisticsRevision, "detail", id, archived] as const,
};
function verifiedQuery(client: QueryClient, scope: ParentCatalogueScope, key: readonly unknown[], signal: AbortSignal) {
  const access = captureParentAccess(client, scope),
    query = client.getQueryCache().find({ queryKey: key, exact: true });
  const current = () =>
    !signal.aborted &&
    access.isVerified() &&
    query !== undefined &&
    client.getQueryCache().find({ queryKey: key, exact: true }) === query;
  const check = () => {
    if (!current()) throw new Error("Cycle or module access changed.");
  };
  return check;
}
function validateParent<T extends Parent>(row: T, scope: ParentCatalogueScope, id?: string) {
  if (
    !row?.id ||
    ("projectId" in scope
      ? row.project_id !== scope.projectId
      : !scope.projectRoles?.some(([projectId]) => row.project_id === projectId)) ||
    (id !== undefined && row.id !== id)
  )
    throw new Error("The cycle or module response did not match its requested scope.");
  return row;
}
function collectionOptions<T extends Parent>(
  domain: ParentDomain,
  scope: ParentCatalogueScope,
  collection: ParentCatalogueCollection,
  read: (signal: AbortSignal) => Promise<T[]>
) {
  const queryKey = parentCatalogueKeys.list(domain, scope, collection);
  return queryOptions({
    queryKey,
    retry: false,
    queryFn: parentScopeReady(scope)
      ? async ({ client, signal }) => {
          const check = verifiedQuery(client, scope, queryKey, signal);
          check();
          const rows = await read(signal);
          check();
          return rows.map((row) => validateParent(row, scope));
        }
      : skipToken,
  });
}
const cycles = new CycleService(),
  modules = new ModuleService(),
  cycleArchive = new CycleArchiveService(),
  moduleArchive = new ModuleArchiveService();
export function projectCycleOptions(scope: ProjectCatalogueScope, archived = false) {
  return collectionOptions("cycle", scope, archived ? "archived" : "active", (signal) => {
    if (!scope.workspaceSlug || !scope.projectId) throw new Error("A project is required.");
    return archived
      ? cycleArchive.getArchivedCycles(scope.workspaceSlug, scope.projectId, signal)
      : cycles.getWithParams(scope.workspaceSlug, scope.projectId, undefined, signal);
  });
}
export function projectModuleOptions(scope: ProjectCatalogueScope, archived = false) {
  return collectionOptions("module", scope, archived ? "archived" : "active", (signal) => {
    if (!scope.workspaceSlug || !scope.projectId) throw new Error("A project is required.");
    return archived
      ? moduleArchive.getArchivedModules(scope.workspaceSlug, scope.projectId, signal)
      : modules.getModules(scope.workspaceSlug, scope.projectId, signal);
  });
}
export function workspaceCycleOptions(scope: WorkspaceCatalogueScope) {
  return collectionOptions("cycle", scope, "active", (signal) => {
    if (!scope.workspaceSlug) throw new Error("A workspace is required.");
    return cycles.getWorkspaceCycles(scope.workspaceSlug, signal);
  });
}
export function workspaceModuleOptions(scope: WorkspaceCatalogueScope) {
  return collectionOptions("module", scope, "active", (signal) => {
    if (!scope.workspaceSlug) throw new Error("A workspace is required.");
    return modules.getWorkspaceModules(scope.workspaceSlug, signal);
  });
}
/** null is an accepted local delete/archive/restore removal from this endpoint; read failures remain Query errors. */
function detailOptions<T extends Parent>(
  domain: ParentDomain,
  scope: ProjectCatalogueScope,
  id: string | undefined,
  archived: boolean,
  read: (signal: AbortSignal) => Promise<T>,
  statisticsRevision?: string
) {
  const queryKey = parentCatalogueKeys.detail(domain, scope, id, archived, statisticsRevision);
  return queryOptions({
    queryKey,
    retry: false,
    queryFn:
      parentScopeReady(scope) && id
        ? async ({ client, signal }): Promise<T | null> => {
            const check = verifiedQuery(client, scope, queryKey, signal);
            check();
            const row = await read(signal);
            check();
            return validateParent(row, scope, id);
          }
        : skipToken,
  });
}
export function cycleDetailOptions(
  scope: ProjectCatalogueScope,
  id: string | undefined,
  archived = false,
  statisticsRevision?: string
) {
  return detailOptions(
    "cycle",
    scope,
    id,
    archived,
    (signal) => {
      if (!scope.workspaceSlug || !scope.projectId || !id) throw new Error("A cycle is required.");
      return archived
        ? cycleArchive.getArchivedCycleDetails(scope.workspaceSlug, scope.projectId, id, signal)
        : cycles.retrieve(scope.workspaceSlug, scope.projectId, id, signal);
    },
    statisticsRevision
  );
}
export function moduleDetailOptions(
  scope: ProjectCatalogueScope,
  id: string | undefined,
  archived = false,
  statisticsRevision?: string
) {
  return detailOptions(
    "module",
    scope,
    id,
    archived,
    (signal) => {
      if (!scope.workspaceSlug || !scope.projectId || !id) throw new Error("A module is required.");
      return archived
        ? moduleArchive.getArchivedModuleDetails(scope.workspaceSlug, scope.projectId, id, signal)
        : modules.getModuleDetails(scope.workspaceSlug, scope.projectId, id, signal);
    },
    statisticsRevision
  );
}
export type ParentWrite<T extends Parent> =
  | { type: "create"; data: Partial<T> }
  | { type: "patch"; id: string; data: Partial<T> }
  | { type: "delete" | "archive" | "restore"; id: string };

type ParentMutationQueries = { affected: Query[]; snapshots: Set<Query>; source: Query | undefined };
type ParentMutationReceipt<T> = { record?: T; archivedAt?: string | null };

function captureParentMutationQueries<T extends Parent>(
  client: QueryClient,
  key: readonly unknown[],
  input: ParentWrite<T>
): ParentMutationQueries {
  const affected = client.getQueryCache().findAll({ queryKey: key });
  const source = affected.find(
    (query) =>
      query.state.status === "success" &&
      (query.queryKey.at(-2) === "list" ||
        (input.type !== "create" && query.queryKey.at(-3) === "detail" && query.queryKey.at(-2) === input.id))
  );
  if (!source && input.type !== "create") throw new Error("Reload the cycle or module before editing it.");
  return { affected, snapshots: new Set(affected), source };
}

function parentMutationRow<T extends Parent>(
  client: QueryClient,
  queries: readonly Query[],
  input: ParentWrite<T>,
  receipt: ParentMutationReceipt<T>
) {
  if (input.type === "create" || !("archivedAt" in receipt)) return receipt.record;
  for (const query of queries) {
    const existing =
      query.queryKey.at(-2) === "list"
        ? client.getQueryData<T[]>(query.queryKey)?.find((item) => item.id === input.id)
        : query.queryKey.at(-3) === "detail" && query.queryKey.at(-2) === input.id
          ? client.getQueryData<T | null>(query.queryKey)
          : undefined;
    if (existing) return { ...existing, archived_at: receipt.archivedAt ?? null };
  }
  return receipt.record;
}

function parentBelongsInList(row: Parent, query: Query) {
  const collection = query.queryKey.at(-1);
  return collection === "universe" || (row.archived_at ? collection === "archived" : collection === "active");
}

function applyParentListReceipt<T extends Parent>(
  client: QueryClient,
  query: Query,
  input: ParentWrite<T>,
  row: T | undefined
) {
  client.setQueryData<T[]>(query.queryKey, (rows) => {
    if (!rows) return rows;
    if (input.type === "delete") return rows.filter((item) => item.id !== input.id);
    if (!row) return rows;
    return [...rows.filter((item) => item.id !== row.id), ...(parentBelongsInList(row, query) ? [row] : [])];
  });
}

function parentDetailIsRemoved<T extends Parent>(query: Query, input: ParentWrite<T>) {
  return (
    input.type === "delete" ||
    (input.type === "archive" && query.queryKey.at(-1) === false) ||
    (input.type === "restore" && query.queryKey.at(-1) === true)
  );
}

function applyParentDetailReceipt<T extends Parent>(
  client: QueryClient,
  query: Query,
  input: ParentWrite<T>,
  row: T | undefined
) {
  if (query.queryKey.at(-3) !== "detail") return;
  if (input.type !== "create" && query.queryKey.at(-2) === input.id && parentDetailIsRemoved(query, input)) {
    client.setQueryData(query.queryKey, null);
  } else if (row && query.queryKey.at(-2) === row.id) {
    client.setQueryData(query.queryKey, row);
  }
}

function applyParentReceipt<T extends Parent>(
  client: QueryClient,
  queries: readonly Query[],
  input: ParentWrite<T>,
  row: T | undefined
) {
  for (const query of queries) {
    if (client.getQueryCache().get(query.queryHash) !== query) continue;
    if (query.queryKey.at(-2) === "list") applyParentListReceipt(client, query, input, row);
    else applyParentDetailReceipt(client, query, input, row);
  }
}

function parentMutationCurrent(client: QueryClient, accessCurrent: () => boolean, source: Query | undefined) {
  return (
    accessCurrent() &&
    (!source || (!isAuthorizationError(source.state.error) && client.getQueryCache().get(source.queryHash) === source))
  );
}

function parentMutationOptions<T extends Parent>(
  client: QueryClient,
  domain: ParentDomain,
  scope: ProjectCatalogueScope,
  send: (input: ParentWrite<T>, isVerified: () => boolean) => Promise<{ record?: T; archivedAt?: string | null }>
) {
  const key = parentCatalogueKeys.scope(domain, scope);
  return mutationOptions({
    mutationKey: [...key, "write"],
    scope: { id: JSON.stringify([...parentCatalogueKeys.owner(domain, scope), scope.projectId, "write"]) },
    retry: false,
    mutationFn: async (input: ParentWrite<T>) => {
      const access = captureParentAccess(client, scope);
      if (!access.isVerified() || !scope.projectRole || ![15, 20].includes(scope.projectRole))
        throw new Error("Cycle or module editing access is unavailable.");
      const { affected, snapshots, source } = captureParentMutationQueries(client, key, input);
      const refreshFavorites = captureWorkspaceFavoritesRefresh(client, scope.workspaceSlug ?? "");
      const verified = () =>
        access.isVerified() &&
        (!source || (source.state.status === "success" && client.getQueryCache().get(source.queryHash) === source));
      const receipt = await send(input, verified);
      const row = parentMutationRow(client, affected, input, receipt);
      if (row) validateParent(row, scope, input.type === "create" ? undefined : input.id);
      const isCurrent = () => parentMutationCurrent(client, access.isCurrent, source);
      if (isCurrent()) {
        await client.cancelQueries({ predicate: (query) => snapshots.has(query) });
        if (isCurrent()) {
          // Apply the accepted row only to the captured cached list and detail queries.
          applyParentReceipt(client, affected, input, row);
          void Promise.allSettled([
            client.invalidateQueries({
              predicate: (query) =>
                snapshots.has(query) && !(query.queryKey.at(-3) === "detail" && query.state.data === null),
            }),
            client.invalidateQueries({ queryKey: [...parentCatalogueKeys.owner(domain, scope), "workspace"] }),
            ...(input.type === "delete" || input.type === "archive" ? [refreshFavorites()] : []),
          ]);
        }
      }
      return { record: row, captured: { isCurrent } };
    },
  });
}
export function cycleMutationOptions(client: QueryClient, scope: ProjectCatalogueScope) {
  return parentMutationOptions<ICycle>(client, "cycle", scope, async (input, isVerified) => {
    const { workspaceSlug, projectId } = scope;
    if (!workspaceSlug || !projectId) throw new Error("A project is required.");
    if ((input.type === "create" || input.type === "patch") && input.data.start_date && input.data.end_date) {
      const validation = await cycles.validateDates(workspaceSlug, projectId, {
        start_date: input.data.start_date,
        end_date: input.data.end_date,
        ...(input.type === "patch" ? { cycle_id: input.id } : {}),
      });
      if (!isVerified()) throw new Error("Cycle editing access changed.");
      if (!validation.status)
        throw new Error("A cycle already exists on these dates. Choose other dates or remove them to create a draft.");
    }
    if (input.type === "create") return { record: await cycles.create(workspaceSlug, projectId, input.data) };
    if (input.type === "patch") return { record: await cycles.update(workspaceSlug, projectId, input.id, input.data) };
    if (input.type === "delete") await cycles.destroy(workspaceSlug, projectId, input.id);
    else if (input.type === "archive")
      return { archivedAt: (await cycleArchive.archiveCycle(workspaceSlug, projectId, input.id)).archived_at };
    else {
      await cycleArchive.restoreCycle(workspaceSlug, projectId, input.id);
      return { archivedAt: null };
    }
    return {};
  });
}
export function moduleMutationOptions(client: QueryClient, scope: ProjectCatalogueScope) {
  return parentMutationOptions<IModule>(client, "module", scope, async (input) => {
    const { workspaceSlug, projectId } = scope;
    if (!workspaceSlug || !projectId) throw new Error("A project is required.");
    if (input.type === "create") return { record: await modules.createModule(workspaceSlug, projectId, input.data) };
    if (input.type === "patch")
      return { record: await modules.patchModule(workspaceSlug, projectId, input.id, input.data) };
    if (input.type === "delete") await modules.deleteModule(workspaceSlug, projectId, input.id);
    else if (input.type === "archive")
      return { archivedAt: (await moduleArchive.archiveModule(workspaceSlug, projectId, input.id)).archived_at };
    else {
      await moduleArchive.restoreModule(workspaceSlug, projectId, input.id);
      return { archivedAt: null };
    }
    return {};
  });
}

function cycleAnalyticsRead<T>(
  scope: ProjectCatalogueScope,
  id: string | undefined,
  purpose: "progress" | "issues" | "points",
  revision: string | undefined,
  read: (signal: AbortSignal) => Promise<T>
) {
  const queryKey = [...parentCatalogueKeys.scope("cycle", scope), purpose, id, revision] as const;
  return queryOptions({
    queryKey,
    retry: false,
    queryFn:
      parentScopeReady(scope) && id
        ? async ({ client, signal }): Promise<T | null> => {
            const check = verifiedQuery(client, scope, queryKey, signal);
            check();
            const result = await read(signal);
            check();
            return result;
          }
        : skipToken,
  });
}
export function cycleProgressOptions(scope: ProjectCatalogueScope, id: string | undefined, revision?: string) {
  return cycleAnalyticsRead(scope, id, "progress", revision, (signal) => {
    if (!scope.workspaceSlug || !scope.projectId || !id) throw new Error("A cycle is required.");
    return cycles.workspaceActiveCyclesProgress(scope.workspaceSlug, scope.projectId, id, signal);
  });
}
export function cycleIssueAnalyticsOptions(scope: ProjectCatalogueScope, id: string | undefined, revision?: string) {
  return cycleAnalyticsRead(scope, id, "issues", revision, (signal) => {
    if (!scope.workspaceSlug || !scope.projectId || !id) throw new Error("A cycle is required.");
    return cycles.issueAnalytics(scope.workspaceSlug, scope.projectId, id, signal);
  });
}
export function cycleEstimateAnalyticsOptions(scope: ProjectCatalogueScope, id: string | undefined, revision?: string) {
  return cycleAnalyticsRead(scope, id, "points", revision, (signal) => {
    if (!scope.workspaceSlug || !scope.projectId || !id) throw new Error("A cycle is required.");
    return cycles.estimateAnalytics(scope.workspaceSlug, scope.projectId, id, signal);
  });
}

/** The server's issue grouping universe includes archived authorized parents. */
export function projectCycleUniverseOptions(scope: ProjectCatalogueScope) {
  return collectionOptions("cycle", scope, "universe", async (signal) => {
    if (!scope.workspaceSlug || !scope.projectId) throw new Error("A project is required.");
    const [active, archived] = await Promise.all([
      cycles.getWithParams(scope.workspaceSlug, scope.projectId, undefined, signal),
      cycleArchive.getArchivedCycles(scope.workspaceSlug, scope.projectId, signal),
    ]);
    return Array.from(new Map([...active, ...archived].map((row) => [row.id, row])).values());
  });
}
export function projectModuleUniverseOptions(scope: ProjectCatalogueScope) {
  return collectionOptions("module", scope, "universe", async (signal) => {
    if (!scope.workspaceSlug || !scope.projectId) throw new Error("A project is required.");
    const [active, archived] = await Promise.all([
      modules.getModules(scope.workspaceSlug, scope.projectId, signal),
      moduleArchive.getArchivedModules(scope.workspaceSlug, scope.projectId, signal),
    ]);
    return Array.from(new Map([...active, ...archived].map((row) => [row.id, row])).values());
  });
}
export function workspaceCycleUniverseOptions(scope: WorkspaceCatalogueScope) {
  return collectionOptions("cycle", scope, "universe", async (signal) => {
    const { workspaceSlug } = scope;
    if (!workspaceSlug || !scope.projectRoles) throw new Error("A workspace is required.");
    const [active, archived] = await Promise.all([
      cycles.getWorkspaceCycles(workspaceSlug, signal),
      Promise.all(
        scope.projectRoles.map(([projectId]) => cycleArchive.getArchivedCycles(workspaceSlug, projectId, signal))
      ),
    ]);
    return Array.from(new Map([...active, ...archived.flat()].map((row) => [row.id, row])).values());
  });
}
export function workspaceModuleUniverseOptions(scope: WorkspaceCatalogueScope) {
  return collectionOptions("module", scope, "universe", async (signal) => {
    const { workspaceSlug } = scope;
    if (!workspaceSlug || !scope.projectRoles) throw new Error("A workspace is required.");
    const [active, archived] = await Promise.all([
      modules.getWorkspaceModules(workspaceSlug, signal),
      Promise.all(
        scope.projectRoles.map(([projectId]) => moduleArchive.getArchivedModules(workspaceSlug, projectId, signal))
      ),
    ]);
    return Array.from(new Map([...active, ...archived.flat()].map((row) => [row.id, row])).values());
  });
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import type { IProjectUserPropertiesResponse, TPartialProject, TProject, TUserPermissions } from "@plane/types";
import { EUserPermissions, EUserWorkspaceRoles } from "@plane/types";
import { effectiveProjectRole } from "@/helpers/permission-policy";
import { getIdentitySessionGeneration } from "@/queries/identity";
import { captureProjectAccessScope, projectRolesRevision } from "@/queries/project-access-scope";
import { capturePermissionChanges } from "@/queries/permission-mutations";
import { captureWorkspaceFavoritesRefresh } from "@/queries/favorites";
import { invalidateProjectLabelCatalogues } from "@/queries/labels";
import { projectUserPropertiesKeys } from "@/queries/project-user-properties";
import { ProjectService, ProjectArchiveService } from "@/services/project";

export type TProjectCatalogueScope = {
  accountId: string | undefined;
  sessionGeneration: number;
  workspaceSlug: string | undefined;
  workspaceRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
  projectRoles: readonly (readonly [string, TUserPermissions])[] | undefined;
  projectId?: string;
  projectMembershipRole?: TUserPermissions;
  projectRole?: TUserPermissions;
};

type ProjectListKind = "partial" | "full";
export type TProjectWrite =
  | { type: "create"; data: Partial<TProject> }
  | { type: "update"; projectId: string; data: Partial<TProject> }
  | { type: "delete"; projectId: string }
  | { type: "archive"; projectId: string }
  | { type: "restore"; projectId: string };

const projectService = new ProjectService();
const projectArchiveService = new ProjectArchiveService();

export const projectKeys = {
  workspacePrefix: (scope: Pick<TProjectCatalogueScope, "accountId" | "sessionGeneration" | "workspaceSlug">) =>
    ["projects", scope.accountId, scope.sessionGeneration, scope.workspaceSlug] as const,
  workspace: (scope: TProjectCatalogueScope, kind: ProjectListKind) =>
    [
      ...projectKeys.workspacePrefix(scope),
      scope.workspaceRole,
      scope.membershipRevision,
      scope.projectRoles,
      kind,
    ] as const,
  detail: (scope: TProjectCatalogueScope, projectId = scope.projectId) =>
    [
      ...projectKeys.workspacePrefix(scope),
      scope.workspaceRole,
      scope.membershipRevision,
      scope.projectRoles,
      projectId,
      scope.projectMembershipRole,
      scope.projectRole,
      "detail",
    ] as const,
  accessProbe: (scope: TProjectCatalogueScope) =>
    [
      ...projectKeys.workspacePrefix(scope),
      scope.workspaceRole,
      scope.membershipRevision,
      scope.projectRoles,
      scope.projectId,
      "access-probe",
    ] as const,
  writes: (scope: TProjectCatalogueScope) => [...projectKeys.workspacePrefix(scope), "write"] as const,
};

function scopeReady(scope: TProjectCatalogueScope, requireProject: boolean) {
  return Boolean(
    scope.accountId &&
    scope.sessionGeneration >= 0 &&
    scope.workspaceSlug &&
    scope.workspaceRole !== undefined &&
    scope.membershipRevision &&
    scope.projectRoles !== undefined &&
    (!requireProject || (scope.projectId && scope.projectRole !== undefined))
  );
}

function exactQuery(client: QueryClient, key: readonly unknown[]) {
  return client.getQueryCache().find({ queryKey: key, exact: true });
}

export function captureProjectCatalogueScope(client: QueryClient, scope: TProjectCatalogueScope) {
  const access = captureProjectAccessScope(client, {
    accountId: scope.accountId,
    workspaceSlug: scope.workspaceSlug,
    sessionGeneration: scope.sessionGeneration,
    membershipRole: scope.workspaceRole,
    membershipRevision: scope.membershipRevision,
    projectId: scope.projectId,
    projectRole: scope.projectRole,
    projectMembershipRole: scope.projectMembershipRole,
    rolesRevision: scope.projectRoles ? projectRolesRevision(Object.fromEntries(scope.projectRoles)) : undefined,
  });
  return {
    isLineageCurrent: access.isCurrent,
    isCurrent: () => scopeReady(scope, Boolean(scope.projectId)) && access.isCurrent(),
    isVerified: () => scopeReady(scope, Boolean(scope.projectId)) && access.isVerified(),
  };
}

function projectOptions<TData>(
  client: QueryClient,
  scope: TProjectCatalogueScope,
  kind: ProjectListKind,
  enabled: boolean,
  fetch: (workspaceSlug: string, signal: AbortSignal) => Promise<TData>
) {
  const { workspaceSlug } = scope;
  const key = projectKeys.workspace(scope, kind);
  const canRead = scopeReady(scope, false);
  return queryOptions({
    queryKey: key,
    staleTime: 20_000,
    retry: false,
    enabled: canRead && enabled,
    queryFn:
      canRead && workspaceSlug
        ? async ({ signal }) => {
            const captured = captureProjectCatalogueScope(client, scope);
            const query = exactQuery(client, key);
            const isCurrent = () => captured.isVerified() && exactQuery(client, key) === query;
            if (!isCurrent()) throw new Error("Workspace project access changed before loading.");
            const projects = await fetch(workspaceSlug, signal);
            if (!isCurrent()) throw new Error("Workspace project access changed while loading.");
            return projects;
          }
        : skipToken,
  });
}

export function workspaceProjectsOptions(client: QueryClient, scope: TProjectCatalogueScope, enabled = true) {
  return projectOptions(client, scope, "partial", enabled, (slug, signal) =>
    projectService.getProjectsLite(slug, signal)
  );
}

export function workspaceProjectDetailsOptions(client: QueryClient, scope: TProjectCatalogueScope, enabled = true) {
  return projectOptions(client, scope, "full", enabled, (slug, signal) => projectService.getProjects(slug, signal));
}

export function projectDetailOptions(client: QueryClient, scope: TProjectCatalogueScope, enabled = true) {
  const { workspaceSlug, projectId } = scope;
  const key = projectKeys.detail(scope);
  const canRead = scopeReady(scope, true);
  return queryOptions({
    queryKey: key,
    staleTime: 20_000,
    retry: false,
    enabled: canRead && enabled,
    queryFn:
      canRead && workspaceSlug && projectId
        ? async ({ signal }) => {
            const captured = captureProjectCatalogueScope(client, scope);
            const query = exactQuery(client, key);
            const isCurrent = () => captured.isVerified() && exactQuery(client, key) === query;
            if (!isCurrent()) throw new Error("Project access changed before loading.");
            const project = await projectService.getProject(workspaceSlug, projectId, signal);
            if (project.id !== scope.projectId || !isCurrent())
              throw new Error("Project access changed while loading.");
            return project;
          }
        : skipToken,
  });
}

export function projectAccessProbeOptions(client: QueryClient, scope: TProjectCatalogueScope, enabled = true) {
  const { workspaceSlug, projectId } = scope;
  const queryKey = projectKeys.accessProbe(scope);
  const canProbe = Boolean(
    scopeReady(scope, false) && projectId && scope.projectRole === undefined && scope.projectRoles !== undefined
  );
  return queryOptions({
    queryKey,
    staleTime: 20_000,
    retry: false,
    enabled: canProbe && enabled,
    queryFn:
      canProbe && workspaceSlug && projectId
        ? async ({ client: queryClient, signal }) => {
            const access = captureProjectAccessScope(queryClient, {
              accountId: scope.accountId,
              sessionGeneration: scope.sessionGeneration,
              workspaceSlug,
              membershipRole: scope.workspaceRole,
              membershipRevision: scope.membershipRevision,
              rolesRevision: projectRolesRevision(Object.fromEntries(scope.projectRoles ?? [])),
              // The probe verifies workspace access and the full role snapshot;
              // it does not authorize or expose this project's detail record.
              projectId: undefined,
              projectRole: undefined,
            });
            const query = exactQuery(queryClient, queryKey);
            const isCurrent = () => access.isVerified() && exactQuery(queryClient, queryKey) === query;
            if (!isCurrent()) throw new Error("Project access changed before its access check.");
            const project = await projectService.getProject(workspaceSlug, projectId, signal);
            if (project.id !== projectId || !isCurrent())
              throw new Error("Project access changed during its access check.");
            return true;
          }
        : skipToken,
  });
}

type CapturedProjectQueries = {
  scope: TProjectCatalogueScope;
  canReconcile: () => boolean;
  queries: readonly Query[];
};

function captureProjectQueries(client: QueryClient, scope: TProjectCatalogueScope): CapturedProjectQueries {
  const captured = captureProjectCatalogueScope(client, scope);
  const keys = [
    projectKeys.workspace(scope, "partial"),
    projectKeys.workspace(scope, "full"),
    ...(scope.projectId ? [projectKeys.detail(scope)] : []),
  ];
  const queries = keys
    .map((queryKey) => client.getQueryCache().find({ queryKey, exact: true }))
    .filter((query): query is Query => Boolean(query));
  return {
    scope,
    queries,
    canReconcile: () =>
      captured.isCurrent() &&
      queries.every((query) => client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) === query),
  };
}

async function cancelCapturedProjectQueries(client: QueryClient, captured: CapturedProjectQueries) {
  if (!captured.canReconcile()) return false;
  await Promise.all(captured.queries.map((query) => client.cancelQueries({ queryKey: query.queryKey, exact: true })));
  return captured.canReconcile();
}

function applyProjectResponse(client: QueryClient, captured: CapturedProjectQueries, project: TProject) {
  if (!captured.canReconcile()) return;
  for (const query of captured.queries) {
    const kind = query.queryKey.at(-1);
    if (kind === "detail" && query.queryKey.at(-4) === project.id) {
      client.setQueryData(query.queryKey, project);
      continue;
    }
    if (kind === "partial" || kind === "full") {
      client.setQueryData<TProject[] | TPartialProject[]>(query.queryKey, (rows) => {
        if (!rows) return rows;
        const index = rows.findIndex((row) => row.id === project.id);
        if (index < 0) return rows;
        const next = rows.slice();
        next[index] = { ...next[index], ...project };
        return next;
      });
    }
  }
}

export type TProjectCatalogueCountOwner = {
  projectId: string;
  isCurrent: () => boolean;
  cancelReads: () => Promise<boolean>;
  refresh: () => Promise<void>;
};

export function captureProjectCatalogueCount(
  client: QueryClient,
  scope: TProjectCatalogueScope,
  projectId: string
): TProjectCatalogueCountOwner | undefined {
  if (!scopeReady(scope, false)) return undefined;
  if (scope.projectId && scope.projectId !== projectId) return undefined;
  const projectMembershipRole = scope.projectRoles?.find(([id]) => id === projectId)?.[1];
  const projectRole = effectiveProjectRole(
    scope.workspaceRole ?? null,
    scope.projectRoles ? Object.fromEntries(scope.projectRoles) : undefined,
    projectId
  );
  const projectScope = { ...scope, projectId, projectMembershipRole, projectRole };
  if (!scopeReady(projectScope, true)) return undefined;
  const owner = captureProjectCatalogueScope(client, projectScope);
  if (!owner.isCurrent()) return undefined;
  const prefix = projectKeys.workspacePrefix(scope);
  const keys = [
    projectKeys.workspace(scope, "partial"),
    projectKeys.workspace(scope, "full"),
    projectKeys.detail(projectScope),
  ];
  const queries = keys
    .map((queryKey) => client.getQueryCache().find({ queryKey, exact: true }))
    .filter((query): query is Query => Boolean(query));
  for (const query of queries) {
    if (client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) !== query) return undefined;
  }
  const isCurrent = () =>
    owner.isCurrent() &&
    queries.every(
      (query) =>
        prefix.every((part, index) => query.queryKey[index] === part) &&
        client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) === query
    );
  return {
    projectId,
    isCurrent,
    cancelReads: async () => {
      if (!isCurrent()) return false;
      await Promise.all(queries.map((query) => client.cancelQueries({ queryKey: query.queryKey, exact: true })));
      return isCurrent();
    },
    refresh: async () => {
      if (!isCurrent()) return;
      await Promise.all(queries.map((query) => client.cancelQueries({ queryKey: query.queryKey, exact: true })));
      if (!isCurrent()) return;
      await Promise.all(
        queries.map((query) =>
          client.invalidateQueries({ queryKey: query.queryKey, exact: true }, { throwOnError: false })
        )
      );
    },
  };
}

type TProjectSortOrderWrite = { projectId: string; sort_order: number };

type TProjectWriteResult = {
  write: TProjectWrite;
  project: TProject | undefined;
  archivedAt?: string | null;
};

function canCreateProject(scope: TProjectCatalogueScope) {
  return scope.workspaceRole === EUserWorkspaceRoles.ADMIN || scope.workspaceRole === EUserWorkspaceRoles.MEMBER;
}

function assertProjectAdminWrite(scope: TProjectCatalogueScope, write: Exclude<TProjectWrite, { type: "create" }>) {
  if (
    !scope.projectId ||
    write.projectId !== scope.projectId ||
    (scope.projectRole !== EUserPermissions.ADMIN && scope.workspaceRole !== EUserWorkspaceRoles.ADMIN)
  )
    throw new Error("Project administration access is required.");
}

async function createProjectWrite(scope: TProjectCatalogueScope, write: Extract<TProjectWrite, { type: "create" }>) {
  if (!scope.workspaceSlug || !canCreateProject(scope)) throw new Error("Workspace membership cannot create projects.");
  const project = await projectService.createProject(scope.workspaceSlug, write.data);
  if (!project.id || !project.workspace) throw new Error("The server returned a different project.");
  return { write, project };
}

async function executeProjectWrite(
  client: QueryClient,
  scope: TProjectCatalogueScope,
  captured: CapturedProjectQueries,
  write: TProjectWrite
): Promise<TProjectWriteResult> {
  if (!scope.workspaceSlug || !captured.canReconcile() || !captureProjectCatalogueScope(client, scope).isVerified())
    throw new Error("Project access changed before the write.");
  if (write.type === "create") return createProjectWrite(scope, write);
  assertProjectAdminWrite(scope, write);
  if (write.type === "update") {
    const project = await projectService.updateProject(scope.workspaceSlug, write.projectId, write.data);
    if (project.id !== write.projectId || !project.workspace)
      throw new Error("The server returned a different project.");
    return { write, project };
  }
  if (write.type === "delete") {
    await projectService.deleteProject(scope.workspaceSlug, write.projectId);
    return { write, project: undefined };
  }
  if (write.type === "archive") {
    const archived = await projectArchiveService.archiveProject(scope.workspaceSlug, write.projectId);
    const previous = client.getQueryData<TProject>(projectKeys.detail(scope, write.projectId));
    return {
      write,
      project: previous ? { ...previous, archived_at: archived.archived_at } : undefined,
      archivedAt: archived.archived_at,
    };
  }
  await projectArchiveService.restoreProject(scope.workspaceSlug, write.projectId);
  const previous = client.getQueryData<TProject>(projectKeys.detail(scope, write.projectId));
  return {
    write,
    project: previous ? { ...previous, archived_at: null } : undefined,
    archivedAt: null,
  };
}

function removeProjectFromCapturedQueries(client: QueryClient, captured: CapturedProjectQueries, projectId: string) {
  for (const query of captured.queries) {
    if (query.queryKey.at(-1) === "detail" && query.queryKey.at(-4) === projectId)
      client.removeQueries({ queryKey: query.queryKey, exact: true });
    else if (query.queryKey.at(-1) === "partial" || query.queryKey.at(-1) === "full")
      client.setQueryData<TProject[] | TPartialProject[]>(query.queryKey, (rows) =>
        rows?.filter((row) => row.id !== projectId)
      );
  }
}

function setProjectArchivedInCapturedQueries(
  client: QueryClient,
  captured: CapturedProjectQueries,
  projectId: string,
  archivedAt: string | null
) {
  for (const query of captured.queries) {
    if (query.queryKey.at(-1) === "detail" && query.queryKey.at(-4) === projectId) {
      client.setQueryData(query.queryKey, (current: TProject | undefined) =>
        current ? { ...current, archived_at: archivedAt } : current
      );
    } else if (query.queryKey.at(-1) === "partial" || query.queryKey.at(-1) === "full") {
      client.setQueryData<TProject[] | TPartialProject[]>(query.queryKey, (rows) =>
        rows?.map((row) => (row.id === projectId ? { ...row, archived_at: archivedAt } : row))
      );
    }
  }
}

async function applyAcceptedProjectWrite(
  client: QueryClient,
  workspaceSlug: string,
  permissions: ReturnType<typeof capturePermissionChanges> | undefined,
  result: TProjectWriteResult,
  captured: CapturedProjectQueries
) {
  const { write, project, archivedAt } = result;
  if (write.type === "delete") {
    removeProjectFromCapturedQueries(client, captured, write.projectId);
    await invalidateProjectLabelCatalogues(client, workspaceSlug, write.projectId);
    void permissions?.removeProjectRole(write.projectId);
    return;
  }
  if (write.type === "create") {
    await permissions?.refreshProjectRoles();
    return;
  }
  if (project) {
    applyProjectResponse(client, captured, project);
    return;
  }
  if (write.type === "archive" || write.type === "restore") {
    const nextArchivedAt = write.type === "archive" ? (archivedAt ?? null) : null;
    setProjectArchivedInCapturedQueries(client, captured, write.projectId, nextArchivedAt);
    await invalidateProjectLabelCatalogues(client, workspaceSlug, write.projectId);
  }
}

async function reconcileProjectWrite(
  client: QueryClient,
  scope: TProjectCatalogueScope,
  permissions: ReturnType<typeof capturePermissionChanges> | undefined,
  refreshFavorites: ReturnType<typeof captureWorkspaceFavoritesRefresh> | undefined,
  result: TProjectWriteResult,
  captured: CapturedProjectQueries | undefined
) {
  if (!scope.workspaceSlug || !captured?.canReconcile()) return;
  await cancelCapturedProjectQueries(client, captured);
  if (!captured.canReconcile()) return;
  await applyAcceptedProjectWrite(client, scope.workspaceSlug, permissions, result, captured);
  await refreshFavorites?.();
}

export function projectSortOrderMutationOptions(client: QueryClient, scope: TProjectCatalogueScope) {
  const generation = getIdentitySessionGeneration(client);
  return mutationOptions({
    mutationKey: [...projectKeys.workspacePrefix(scope), "sort-order", generation],
    scope: { id: `project-order/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}` },
    retry: false,
    mutationFn: async (write: TProjectSortOrderWrite) => {
      const membershipRole = scope.projectRoles?.find(([id]) => id === write.projectId)?.[1];
      const projectRole = effectiveProjectRole(
        scope.workspaceRole ?? null,
        scope.projectRoles ? Object.fromEntries(scope.projectRoles) : undefined,
        write.projectId
      );
      const projectScope = { ...scope, projectId: write.projectId, projectMembershipRole: membershipRole, projectRole };
      const access = captureProjectCatalogueScope(client, projectScope);
      const capturedLists = captureProjectQueries(client, projectScope);
      const propertiesScope = {
        accountId: scope.accountId,
        workspaceSlug: scope.workspaceSlug,
        projectId: write.projectId,
      };
      const propertiesKey = projectUserPropertiesKeys.detail(propertiesScope);
      const propertiesQuery = exactQuery(client, propertiesKey);
      if (!projectRole || !access.isVerified() || !capturedLists.canReconcile())
        throw new Error("Project membership changed before reordering.");
      await cancelCapturedProjectQueries(client, capturedLists);
      if (propertiesQuery) await client.cancelQueries({ queryKey: propertiesKey, exact: true });
      if (
        !access.isVerified() ||
        !capturedLists.canReconcile() ||
        (propertiesQuery && exactQuery(client, propertiesKey) !== propertiesQuery)
      )
        throw new Error("Project membership changed before reordering.");
      const saved = await projectService.updateProjectUserProperties(scope.workspaceSlug!, write.projectId, {
        sort_order: write.sort_order,
      });
      if (saved.sort_order !== write.sort_order) throw new Error("The server returned a different project order.");
      return { write, saved, propertiesScope, propertiesQuery, capturedLists };
    },
    onSuccess: async ({ saved, propertiesScope, propertiesQuery, capturedLists }) => {
      if (!capturedLists.canReconcile()) return;
      const propertiesKey = projectUserPropertiesKeys.detail(propertiesScope);
      if (propertiesQuery) {
        await client.cancelQueries({ queryKey: propertiesKey, exact: true });
        if (!capturedLists.canReconcile() || exactQuery(client, propertiesKey) !== propertiesQuery) return;
        client.setQueryData<IProjectUserPropertiesResponse>(propertiesKey, saved);
      }
      const listQueries = capturedLists.queries.filter((query) =>
        ["partial", "full"].includes(String(query.queryKey.at(-1)))
      );
      if (!capturedLists.canReconcile()) return;
      await Promise.all(
        listQueries.map((query) =>
          client.invalidateQueries({ queryKey: query.queryKey, exact: true }, { throwOnError: false })
        )
      );
    },
  });
}

export function projectMutationOptions(client: QueryClient, scope: TProjectCatalogueScope) {
  const generation = getIdentitySessionGeneration(client);
  const captured = captureProjectQueries(client, scope);
  const permissions = scope.workspaceSlug
    ? capturePermissionChanges(client, scope.accountId, scope.workspaceSlug)
    : undefined;
  const refreshFavorites = scope.workspaceSlug
    ? captureWorkspaceFavoritesRefresh(client, scope.workspaceSlug)
    : undefined;
  return mutationOptions({
    mutationKey: [...projectKeys.writes(scope), generation],
    scope: { id: `projects/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}` },
    retry: false,
    onMutate: async () => {
      if (!captured.canReconcile()) throw new Error("Project access changed before the write.");
      await cancelCapturedProjectQueries(client, captured);
      if (!captured.canReconcile()) throw new Error("Project access changed before the write.");
      return captured;
    },
    mutationFn: (write: TProjectWrite) => executeProjectWrite(client, scope, captured, write),
    onSuccess: (result, _variables, context) =>
      reconcileProjectWrite(client, scope, permissions, refreshFavorites, result, context),
  });
}

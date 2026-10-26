/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import groupBy from "lodash-es/groupBy.js";
import sortBy from "lodash-es/sortBy.js";
import { PROJECT_STATES, WORKSPACE_STATES } from "@plane/constants";
import { EUserPermissions, STATE_GROUPS } from "@plane/constants";
import type { IState, IUserProjectsRole, TUserPermissions } from "@plane/types";
import { sortStates } from "@plane/utils/work-item";
import { isWorkspaceRole } from "@/helpers/permission-policy";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import { getCurrentUserId, getIdentitySessionGeneration } from "@/queries/identity";
import { isAuthorizationError } from "@/queries/authorization-errors";
import { workspaceCatalogueScopeReady } from "@/queries/catalogue-scope";
import type { ProjectCatalogueScope, WorkspaceCatalogueScope } from "@/queries/catalogue-scope";
import { permissionKeys } from "@/queries/permissions";
import { ProjectStateService } from "@/services/project/project-state.service";

export type TStateProjection = {
  scopeKey: string;
  statesById: Readonly<Record<string, IState>>;
  isCurrent: () => boolean;
};

export type TProjectStateProjection = TStateProjection & {
  workspaceSlug: string;
  projectId: string;
};

export function stateOrderReadiness(
  required: boolean,
  accessPending: boolean,
  accessError: boolean,
  accessDenied: boolean,
  queryPending: boolean,
  queryError: boolean
) {
  return {
    isPending: required && (accessPending || (!accessDenied && queryPending)),
    isError: required && (accessError || queryError),
  };
}

type StateWrite =
  | { type: "create"; data: Partial<IState> }
  | { type: "patch"; stateId: string; data: Partial<IState> }
  | { type: "delete"; stateId: string }
  | { type: "default"; stateId: string }
  | { type: "reorder"; stateId: string; data: Partial<IState> };

const service = new ProjectStateService();
const PROJECT_STATE_READ_ROLES = [EUserPermissions.ADMIN, EUserPermissions.MEMBER, EUserPermissions.GUEST] as const;

export function canReadProjectStates(role: TUserPermissions | undefined) {
  return role !== undefined && PROJECT_STATE_READ_ROLES.includes(role);
}

export const stateKeys = {
  projectPrefix: (workspaceSlug: string | undefined, projectId: string | undefined) =>
    ["PROJECT_STATES", workspaceSlug, projectId] as const,
  project: (scope: ProjectCatalogueScope) =>
    [
      ...PROJECT_STATES(scope.workspaceSlug, scope.projectId, scope.projectRole),
      scope.accountId,
      scope.sessionGeneration,
      scope.workspaceRole,
      scope.membershipRevision,
      scope.projectMembershipRole,
    ] as const,
  intake: (scope: ProjectCatalogueScope) => [...stateKeys.project(scope), "intake"] as const,
  workspacePrefix: (scope: Pick<WorkspaceCatalogueScope, "accountId" | "sessionGeneration" | "workspaceSlug">) =>
    [...WORKSPACE_STATES(scope.workspaceSlug), scope.accountId, scope.sessionGeneration] as const,
  workspace: (scope: WorkspaceCatalogueScope) =>
    [...stateKeys.workspacePrefix(scope), scope.workspaceRole, scope.membershipRevision, scope.projectRoles] as const,
  write: (scope: ProjectCatalogueScope) =>
    [
      ...stateKeys.projectPrefix(scope.workspaceSlug, scope.projectId),
      scope.accountId,
      scope.sessionGeneration,
      "write",
    ] as const,
};

export function projectRoleSnapshot(roles: IUserProjectsRole | undefined) {
  return sortBy(Object.entries(roles ?? {}), ([projectId]) => projectId);
}

export function currentStateRows(query: { isSuccess: boolean; data: readonly IState[] | undefined }) {
  if (!query.isSuccess) return undefined;
  return sortStates([...(query.data ?? [])]) ?? [];
}

export function groupedStateRows(states: readonly IState[] | undefined) {
  if (!states) return undefined;
  const grouped = groupBy(sortStates([...states]) ?? [], "group");
  return Object.fromEntries(Object.keys(STATE_GROUPS).map((group) => [group, grouped[group] ?? []])) as Record<
    string,
    IState[]
  >;
}

export function stateRowsById(states: readonly IState[] | undefined) {
  return states?.reduce<Record<string, IState>>((rows, state) => {
    rows[state.id] = state;
    return rows;
  }, {});
}

export function statePercentageInGroup(states: readonly IState[] | undefined, stateId: string | null | undefined) {
  if (!stateId || !states) return -1;
  const state = states.find((row) => row.id === stateId);
  if (!state) return -1;
  const siblings = states.filter((row) => row.group === state.group);
  const index = siblings.findIndex((row) => row.id === stateId);
  return index < 0 ? undefined : ((index + 1) / siblings.length) * 100;
}

function exactQuery(client: QueryClient, key: readonly unknown[]) {
  return client.getQueryCache().find({ queryKey: key, exact: true });
}

function currentQuery(client: QueryClient, key: readonly unknown[], query: Query | undefined) {
  return Boolean(query && exactQuery(client, key) === query);
}

function activeProjectRolesQuery(client: QueryClient, scope: ProjectCatalogueScope) {
  const key = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.workspaceRole,
    scope.membershipRevision
  );
  const query = exactQuery(client, key);
  return query?.state.status === "success" ? { key, query } : undefined;
}

function projectScopeReady(scope: ProjectCatalogueScope) {
  return Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.projectId &&
    isWorkspaceRole(scope.workspaceRole) &&
    scope.membershipRevision &&
    canReadProjectStates(scope.projectRole)
  );
}

function captureProjectScope(client: QueryClient, scope: ProjectCatalogueScope, queryKey: readonly unknown[]) {
  const query = exactQuery(client, queryKey);
  const roles = activeProjectRolesQuery(client, scope);
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  const membershipQuery = exactQuery(client, membershipKey);
  const access = captureProjectAccessScope(client, {
    accountId: scope.accountId,
    workspaceSlug: scope.workspaceSlug,
    projectId: scope.projectId,
    membershipRole: scope.workspaceRole,
    membershipRevision: scope.membershipRevision,
    projectRole: scope.projectRole,
  });
  const isCurrent = () => {
    const roleMap = roles ? client.getQueryData<IUserProjectsRole>(roles.key) : undefined;
    return Boolean(
      getCurrentUserId(client) === scope.accountId &&
      getIdentitySessionGeneration(client) === scope.sessionGeneration &&
      access.isCurrent() &&
      currentQuery(client, queryKey, query) &&
      currentQuery(client, membershipKey, membershipQuery) &&
      roles &&
      currentQuery(client, roles.key, roles.query) &&
      !isAuthorizationError(roles.query.state.error) &&
      roleMap?.[scope.projectId ?? ""] === scope.projectMembershipRole
    );
  };
  return {
    query,
    isCurrent,
    isVerifiedCurrent: () => isCurrent() && access.isVerified() && roles?.query.state.status === "success",
  };
}

function captureWorkspaceScope(client: QueryClient, scope: WorkspaceCatalogueScope, queryKey: readonly unknown[]) {
  const query = exactQuery(client, queryKey);
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.workspaceRole,
    scope.membershipRevision
  );
  const rolesQuery = exactQuery(client, rolesKey);
  const access = captureProjectAccessScope(client, {
    accountId: scope.accountId,
    workspaceSlug: scope.workspaceSlug,
    projectId: undefined,
    projectRole: undefined,
    membershipRole: scope.workspaceRole,
    membershipRevision: scope.membershipRevision,
  });
  const isCurrent = () => {
    const roles = client.getQueryData<IUserProjectsRole>(rolesKey);
    return (
      getCurrentUserId(client) === scope.accountId &&
      getIdentitySessionGeneration(client) === scope.sessionGeneration &&
      access.isCurrent() &&
      currentQuery(client, queryKey, query) &&
      currentQuery(client, rolesKey, rolesQuery) &&
      !isAuthorizationError(rolesQuery?.state.error) &&
      JSON.stringify(projectRoleSnapshot(roles)) === JSON.stringify(scope.projectRoles)
    );
  };
  return {
    isCurrent,
    isVerifiedCurrent: () => isCurrent() && access.isVerified() && rolesQuery?.state.status === "success",
  };
}

export function projectStateOptions(client: QueryClient, scope: ProjectCatalogueScope) {
  const queryKey = stateKeys.project(scope);
  return queryOptions({
    queryKey,
    queryFn: projectScopeReady(scope)
      ? async ({ signal }) => {
          const captured = captureProjectScope(client, scope, queryKey);
          if (!captured.isVerifiedCurrent()) throw new Error("The project state scope changed or is unverified.");
          const states = await service.getStates(scope.workspaceSlug!, scope.projectId!, signal);
          if (!captured.isVerifiedCurrent()) throw new Error("The project state scope changed or is unverified.");
          return states;
        }
      : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
    retry: false,
  });
}

export function projectIntakeStateOptions(client: QueryClient, scope: ProjectCatalogueScope) {
  const queryKey = stateKeys.intake(scope);
  return queryOptions({
    queryKey,
    queryFn: projectScopeReady(scope)
      ? async ({ signal }) => {
          const captured = captureProjectScope(client, scope, queryKey);
          if (!captured.isVerifiedCurrent())
            throw new Error("The project intake state scope changed or is unverified.");
          const state = await service.getIntakeState(scope.workspaceSlug!, scope.projectId!, signal);
          if (!captured.isVerifiedCurrent())
            throw new Error("The project intake state scope changed or is unverified.");
          return state;
        }
      : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
    retry: false,
  });
}

export function workspaceStateOptions(client: QueryClient, scope: WorkspaceCatalogueScope) {
  const queryKey = stateKeys.workspace(scope);
  const enabled = workspaceCatalogueScopeReady(scope);
  return queryOptions({
    queryKey,
    queryFn: enabled
      ? async ({ signal }) => {
          const captured = captureWorkspaceScope(client, scope, queryKey);
          if (!captured.isVerifiedCurrent()) throw new Error("The workspace state scope changed or is unverified.");
          const states = await service.getWorkspaceStates(scope.workspaceSlug!, signal);
          if (!captured.isVerifiedCurrent()) throw new Error("The workspace state scope changed or is unverified.");
          return states;
        }
      : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: false,
    retry: false,
  });
}

function canWrite(scope: ProjectCatalogueScope) {
  return projectScopeReady(scope) && scope.projectRole === EUserPermissions.ADMIN;
}

function validateStateWriteResult(scope: ProjectCatalogueScope, write: StateWrite, state: IState | undefined) {
  if (write.type === "delete" || write.type === "default") return;
  if (write.type === "create" && !state?.id) throw new Error("The server did not return the created project state.");
  if (state?.project_id !== scope.projectId) throw new Error("The server returned a state from a different project.");
  if (write.type !== "create" && (!state || state.id !== write.stateId))
    throw new Error("The server returned a different project state.");
}

async function executeStateWrite(scope: ProjectCatalogueScope, write: StateWrite) {
  switch (write.type) {
    case "create":
      return service.createState(scope.workspaceSlug!, scope.projectId!, write.data);
    case "patch":
    case "reorder":
      return service.patchState(scope.workspaceSlug!, scope.projectId!, write.stateId, write.data);
    case "delete":
      await service.deleteState(scope.workspaceSlug!, scope.projectId!, write.stateId);
      return undefined;
    case "default":
      await service.markDefault(scope.workspaceSlug!, scope.projectId!, write.stateId);
      return undefined;
  }
  return undefined;
}

export function projectStateMutationOptions(client: QueryClient, scope: ProjectCatalogueScope) {
  return mutationOptions({
    mutationKey: stateKeys.write(scope),
    scope: {
      id: `project-states/${scope.accountId}/${scope.sessionGeneration}/${scope.workspaceSlug}/${scope.projectId}`,
    },
    retry: false,
    mutationFn: async (write: StateWrite) => {
      if (!canWrite(scope)) throw new Error("Project administrator access is required to change states.");
      const captured = captureProjectScope(client, scope, stateKeys.project(scope));
      if (!captured.isVerifiedCurrent()) throw new Error("The project state scope changed or is unverified.");
      const intakeKey = stateKeys.intake(scope);
      const intakeQuery = exactQuery(client, intakeKey);
      const relatedQueries = [
        ...(intakeQuery ? [{ queryKey: intakeKey, query: intakeQuery }] : []),
        ...client
          .getQueryCache()
          .findAll({ queryKey: stateKeys.workspacePrefix(scope) })
          .map((query) => ({ queryKey: query.queryKey, query })),
      ];
      const result = await executeStateWrite(scope, write);
      validateStateWriteResult(scope, write, result);
      return { result, accepted: captured, relatedQueries };
    },
    onSuccess: async (accepted) => {
      const queryKey = stateKeys.project(scope);
      const scopeCurrent = accepted.accepted.isCurrent();
      if (accepted.accepted.query && currentQuery(client, queryKey, accepted.accepted.query)) {
        await client.invalidateQueries(
          scopeCurrent ? { queryKey, exact: true } : { queryKey, exact: true, refetchType: "none" }
        );
      }
      await Promise.all(
        accepted.relatedQueries
          .filter(({ queryKey: relatedQueryKey, query }) => currentQuery(client, relatedQueryKey, query))
          .map(({ queryKey: relatedQueryKey }) =>
            client.invalidateQueries({
              queryKey: relatedQueryKey,
              exact: true,
              refetchType: accepted.accepted.isCurrent() ? "active" : "none",
            })
          )
      );
    },
  });
}

type StateQuerySnapshot = {
  isSuccess: boolean;
  fetchStatus: "fetching" | "paused" | "idle";
  dataUpdatedAt: number;
};

function stateProjection(
  client: QueryClient,
  queryKey: readonly unknown[],
  queryState: StateQuerySnapshot,
  statesById: Readonly<Record<string, IState>> | undefined,
  isAccessCurrent: () => boolean
): TStateProjection | undefined {
  if (!queryState.isSuccess || queryState.fetchStatus !== "idle" || !statesById) return undefined;
  const query = exactQuery(client, queryKey);
  if (
    !query ||
    query.state.status !== "success" ||
    query.state.fetchStatus !== "idle" ||
    query.state.dataUpdatedAt !== queryState.dataUpdatedAt
  )
    return undefined;
  const sourceData = query.state.data;
  return {
    scopeKey: JSON.stringify(queryKey),
    statesById,
    isCurrent: () =>
      currentQuery(client, queryKey, query) &&
      query.state.status === "success" &&
      query.state.fetchStatus === "idle" &&
      query.state.data === sourceData &&
      query.state.dataUpdatedAt === queryState.dataUpdatedAt &&
      isAccessCurrent(),
  };
}

export function projectStateProjection(
  client: QueryClient,
  scope: ProjectCatalogueScope,
  queryState: StateQuerySnapshot,
  statesById: Readonly<Record<string, IState>> | undefined
): TProjectStateProjection | undefined {
  if (!scope.workspaceSlug || !scope.projectId) return undefined;
  const queryKey = stateKeys.project(scope);
  const captured = captureProjectScope(client, scope, queryKey);
  const projection = stateProjection(client, queryKey, queryState, statesById, captured.isVerifiedCurrent);
  return projection ? { ...projection, workspaceSlug: scope.workspaceSlug, projectId: scope.projectId } : undefined;
}

export function workspaceStateProjection(
  client: QueryClient,
  scope: WorkspaceCatalogueScope,
  queryState: StateQuerySnapshot,
  statesById: Readonly<Record<string, IState>> | undefined
): TStateProjection | undefined {
  if (!scope.workspaceSlug) return undefined;
  const queryKey = stateKeys.workspace(scope);
  const captured = captureWorkspaceScope(client, scope, queryKey);
  return stateProjection(client, queryKey, queryState, statesById, captured.isVerifiedCurrent);
}

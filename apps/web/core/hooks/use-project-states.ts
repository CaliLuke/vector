/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useMemo } from "react";
import { useParams } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { IState, IIssueDisplayFilterOptions } from "@plane/types";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useProjectPermissions, useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import {
  currentStateRows,
  stateOrderReadiness,
  canReadProjectStates,
  groupedStateRows,
  projectRoleSnapshot,
  projectStateProjection,
  projectStateMutationOptions,
  projectStateOptions,
  projectIntakeStateOptions,
  stateKeys,
  statePercentageInGroup,
  stateRowsById,
  workspaceStateOptions,
  workspaceStateProjection,
} from "@/queries/states";
import type { ProjectCatalogueScope, WorkspaceCatalogueScope } from "@/queries/catalogue-scope";

type StatePermissions = Pick<
  ReturnType<typeof useWorkspaceProjectRoles>,
  "isPending" | "isError" | "membershipQuery" | "rolesQuery"
>;

function stateReadAccess(
  account: ReturnType<typeof useCurrentUser>,
  permissions: StatePermissions,
  scopeAllowed: boolean
) {
  return {
    canRead: account.isSuccess && !permissions.isPending && !permissions.isError && scopeAllowed,
    isPending: account.isPending || permissions.isPending,
    isError: account.isError || permissions.isError,
    isDenied: account.isSuccess && !permissions.isPending && !permissions.isError && !scopeAllowed,
  };
}

async function retryStateAccess(account: ReturnType<typeof useCurrentUser>, permissions: StatePermissions) {
  if (account.isError) await account.refetch();
  else if (permissions.membershipQuery.isError) await permissions.membershipQuery.refetch();
  else if (permissions.rolesQuery.isError) await permissions.rolesQuery.refetch();
}

function projectStatesFor(
  project: string | undefined,
  target: string | null | undefined,
  states: IState[] | undefined
) {
  if (!target || target !== project) return undefined;
  return states;
}

export function useProjectStates(
  workspaceSlug: string | null | undefined,
  projectId: string | null | undefined,
  includeIntake = false,
  enabled = true
) {
  const workspace = workspaceSlug ?? undefined;
  const project = projectId ?? undefined;
  const client = useQueryClient();
  const account = useCurrentUser();
  const accountId = account.isSuccess ? account.data?.id : undefined;
  const sessionGeneration = useIdentityGeneration(client);
  const permissions = useProjectPermissions(workspace, project);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const projectMembershipRole = permissions.rolesQuery.isSuccess
    ? permissions.rolesQuery.data?.[projectId ?? ""]
    : undefined;
  const scope = useMemo<ProjectCatalogueScope>(
    () => ({
      accountId,
      workspaceSlug: workspace,
      projectId: project,
      workspaceRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
      projectRole: permissions.projectRole,
      projectMembershipRole,
      sessionGeneration,
    }),
    [
      accountId,
      membership?.updated_at,
      permissions.projectRole,
      permissions.workspaceRole,
      project,
      projectMembershipRole,
      sessionGeneration,
      workspace,
    ]
  );
  const query = useQuery({ ...projectStateOptions(client, scope), enabled });
  const intakeQuery = useQuery({ ...projectIntakeStateOptions(client, scope), enabled: includeIntake && enabled });
  const mutation = useMutation(projectStateMutationOptions(client, scope));
  const access = stateReadAccess(account, permissions, canReadProjectStates(scope.projectRole));
  const states = useMemo(() => (access.canRead ? currentStateRows(query) : undefined), [access.canRead, query]);
  const byId = useMemo(() => stateRowsById(states), [states]);
  const groups = useMemo(() => groupedStateRows(states), [states]);
  const projection = useMemo(() => projectStateProjection(client, scope, query, byId), [byId, client, query, scope]);
  const getStateById = useCallback(
    (stateId: string | null | undefined) => (stateId ? byId?.[stateId] : undefined),
    [byId]
  );
  const getProjectStates = useCallback(
    (targetProjectId: string | null | undefined) => projectStatesFor(project, targetProjectId, states),
    [project, states]
  );
  const getProjectStateIds = useCallback(
    (targetProjectId: string | null | undefined) => getProjectStates(targetProjectId)?.map((state) => state.id),
    [getProjectStates]
  );
  const getProjectDefaultStateId = useCallback(
    (targetProjectId: string | null | undefined) =>
      getProjectStates(targetProjectId)?.find((state) => state.default)?.id,
    [getProjectStates]
  );
  const getStatePercentageInGroup = useCallback(
    (stateId: string | null | undefined) => statePercentageInGroup(states, stateId),
    [states]
  );
  const getIntakeStateById = useCallback(
    (stateId: string | null | undefined) =>
      stateId && intakeQuery.isSuccess && intakeQuery.data?.id === stateId ? intakeQuery.data : undefined,
    [intakeQuery.data, intakeQuery.isSuccess]
  );
  const getProjectIntakeStateIds = useCallback(
    (targetProjectId: string | null | undefined) =>
      targetProjectId === project && intakeQuery.isSuccess && intakeQuery.data?.id ? [intakeQuery.data.id] : undefined,
    [intakeQuery.data, intakeQuery.isSuccess, project]
  );
  const getProjectIntakeState = useCallback(
    (targetProjectId: string | null | undefined) =>
      targetProjectId === project && intakeQuery.isSuccess && intakeQuery.data ? intakeQuery.data : undefined,
    [intakeQuery.data, intakeQuery.isSuccess, project]
  );
  const createState = useCallback(
    async (data: Partial<IState>) => {
      const { result: created } = await mutation.mutateAsync({ type: "create", data });
      if (!created) throw new Error("State creation did not return the created state.");
      return created;
    },
    [mutation]
  );
  const updateState = useCallback(
    async (stateId: string, data: Partial<IState>) =>
      (await mutation.mutateAsync({ type: "patch", stateId, data })).result,
    [mutation]
  );
  const deleteState = useCallback(
    (stateId: string) => mutation.mutateAsync({ type: "delete", stateId }).then(() => undefined),
    [mutation]
  );
  const markStateAsDefault = useCallback(
    (stateId: string) => mutation.mutateAsync({ type: "default", stateId }).then(() => undefined),
    [mutation]
  );
  const moveStatePosition = useCallback(
    (stateId: string, data: Partial<IState>) =>
      mutation.mutateAsync({ type: "reorder", stateId, data }).then(() => undefined),
    [mutation]
  );

  return {
    ...query,
    accessPending: access.isPending,
    accessError: access.isError,
    accessDenied: access.isDenied,
    retryAccess: () => retryStateAccess(account, permissions),
    states,
    projectStates: states,
    groupedProjectStates: groups,
    stateById: byId,
    projection,
    getStateById,
    getProjectStates,
    getProjectStateIds,
    getProjectDefaultStateId,
    getStatePercentageInGroup,
    intakeQuery,
    getIntakeStateById,
    getProjectIntakeState,
    getProjectIntakeStateIds,
    createState,
    updateState,
    deleteState,
    markStateAsDefault,
    moveStatePosition,
    mutation,
    scope,
    queryKey: stateKeys.project(scope),
  };
}

export function useRouteProjectStates(projectId: string | null | undefined, includeIntake = false) {
  const { workspaceSlug } = useParams({ strict: false });
  return useProjectStates(workspaceSlug, projectId, includeIntake);
}

export function useWorkspaceStates(workspaceSlug: string | null | undefined, enabled = true) {
  const workspace = workspaceSlug ?? undefined;
  const client = useQueryClient();
  const account = useCurrentUser();
  const accountId = account.isSuccess ? account.data?.id : undefined;
  const sessionGeneration = useIdentityGeneration(client);
  const permissions = useWorkspaceProjectRoles(workspace);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const projectRoles = useMemo<WorkspaceCatalogueScope["projectRoles"]>(
    () => (permissions.rolesQuery.isSuccess ? projectRoleSnapshot(permissions.rolesQuery.data) : undefined),
    [permissions.rolesQuery.data, permissions.rolesQuery.isSuccess]
  );
  const scope = useMemo<WorkspaceCatalogueScope>(
    () => ({
      accountId,
      workspaceSlug: workspace,
      workspaceRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
      projectRoles,
      sessionGeneration,
    }),
    [accountId, membership?.updated_at, permissions.workspaceRole, projectRoles, sessionGeneration, workspace]
  );
  const query = useQuery({ ...workspaceStateOptions(client, scope), enabled });
  const access = stateReadAccess(account, permissions, Boolean(scope.workspaceRole && projectRoles !== undefined));
  const states = useMemo(() => (access.canRead ? currentStateRows(query) : undefined), [access.canRead, query]);
  const byId = useMemo(() => stateRowsById(states), [states]);
  const projection = useMemo(() => workspaceStateProjection(client, scope, query, byId), [byId, client, query, scope]);
  const getProjectStates = useCallback(
    (projectId: string | null | undefined) =>
      projectId ? states?.filter((state) => state.project_id === projectId) : undefined,
    [states]
  );
  const getStateById = useCallback(
    (stateId: string | null | undefined) => (stateId ? byId?.[stateId] : undefined),
    [byId]
  );
  return {
    ...query,
    accessPending: access.isPending,
    accessError: access.isError,
    accessDenied: access.isDenied,
    retryAccess: () => retryStateAccess(account, permissions),
    states,
    workspaceStates: states,
    getProjectStates,
    getStateById,
    stateById: byId,
    projection,
    scope,
  };
}

export function useIssueStateOrder(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  workspaceLevel: boolean,
  orderBy: IIssueDisplayFilterOptions["order_by"]
) {
  const required = orderBy === "state__name" || orderBy === "-state__name";
  const project = useProjectStates(
    workspaceLevel ? undefined : workspaceSlug,
    workspaceLevel ? undefined : projectId,
    false,
    required && !workspaceLevel
  );
  const workspace = useWorkspaceStates(workspaceLevel ? workspaceSlug : undefined, required && workspaceLevel);
  const state = workspaceLevel ? workspace : project;
  const readiness = stateOrderReadiness(
    required,
    state.accessPending,
    state.accessError,
    state.accessDenied,
    state.isPending,
    state.isError
  );
  return {
    projection: state.projection,
    ...readiness,
    retry: async () => {
      if (required && !state.accessDenied) {
        if (state.accessError) await state.retryAccess();
        else await state.refetch();
      }
    },
  };
}

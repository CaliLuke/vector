/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import type { IProjectView } from "@plane/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { effectiveProjectRole, isWorkspaceRole } from "@/helpers/permission-policy";
import { captureProjectAccessScope } from "@/queries/project-access-scope";
import { currentUserOptions, getIdentitySessionGeneration } from "@/queries/identity";
import { workspaceMembershipOptions, workspaceProjectRolesOptions } from "@/queries/permissions";
import { projectViewMutationOptions, projectViewOptions, projectViewsOptions } from "@/queries/project-views";
import type { TProjectViewsScope } from "@/queries/project-views";

function hasActiveWorkspaceMembership(
  permissions: ReturnType<typeof useProjectPermissions>,
  scope: TProjectViewsScope
) {
  return Boolean(
    permissions.membershipQuery.isSuccess && scope.membershipRole !== undefined && scope.membershipRevision
  );
}

function hasProjectRole(permissions: ReturnType<typeof useProjectPermissions>, scope: TProjectViewsScope) {
  return (
    permissions.rolesQuery.isSuccess && scope.projectMembershipRole !== undefined && scope.projectRole !== undefined
  );
}

function isProjectViewAccessDenied(
  currentUser: ReturnType<typeof useCurrentUser>,
  permissions: ReturnType<typeof useProjectPermissions>,
  membership: ReturnType<typeof useProjectPermissions>["membershipQuery"]["data"],
  scope: TProjectViewsScope,
  workspaceSlug: string | undefined,
  projectId: string | undefined
) {
  const identityDenied = currentUser.isSuccess && !currentUser.data?.id;
  const membershipDenied =
    permissions.membershipQuery.isSuccess &&
    Boolean(membership && (!membership.is_active || !isWorkspaceRole(membership.role) || !membership.updated_at));
  const projectDenied =
    permissions.rolesQuery.isSuccess &&
    membership?.is_active === true &&
    (scope.projectMembershipRole === undefined || scope.projectRole === undefined);
  return Boolean(workspaceSlug && projectId && (identityDenied || membershipDenied || projectDenied));
}

function getProjectViewScopeState(
  currentUser: ReturnType<typeof useCurrentUser>,
  permissions: ReturnType<typeof useProjectPermissions>,
  membership: ReturnType<typeof useProjectPermissions>["membershipQuery"]["data"],
  scope: TProjectViewsScope,
  workspaceSlug: string | undefined,
  projectId: string | undefined
) {
  const isError = currentUser.isError || permissions.isError;
  const isReady = Boolean(
    currentUser.isSuccess &&
    scope.accountId &&
    hasActiveWorkspaceMembership(permissions, scope) &&
    hasProjectRole(permissions, scope)
  );
  const isDenied = isProjectViewAccessDenied(currentUser, permissions, membership, scope, workspaceSlug, projectId);
  return {
    isReady,
    isDenied,
    isError,
    isPending: !isError && !isDenied && (!currentUser.isSuccess || permissions.isPending),
  };
}

async function fetchVerifiedQuery<TData>(
  client: ReturnType<typeof useQueryClient>,
  queryKey: readonly unknown[],
  fetch: () => Promise<TData>,
  isCurrent: () => boolean
) {
  const request = fetch();
  const query = client.getQueryCache().find({ queryKey, exact: true });
  const result = await request;
  return isCurrent() &&
    query &&
    client.getQueryCache().find({ queryKey, exact: true }) === query &&
    query.state.status === "success" &&
    client.getQueryData<TData>(queryKey) === result
    ? result
    : undefined;
}

function routeIsCurrent(
  router: ReturnType<typeof useRouter>,
  routePath: string,
  sessionGeneration: number,
  client: ReturnType<typeof useQueryClient>,
  accountId?: string
) {
  return (
    router.state.location.pathname === routePath &&
    sessionGeneration === getIdentitySessionGeneration(client) &&
    (!accountId || client.getQueryData<{ id?: string }>(currentUserOptions().queryKey)?.id === accountId)
  );
}

async function refetchViewAccess(
  client: ReturnType<typeof useQueryClient>,
  router: ReturnType<typeof useRouter>,
  routePath: string,
  sessionGeneration: number,
  workspaceSlug: string | undefined,
  projectId: string | undefined
): Promise<TProjectViewsScope | undefined> {
  const currentUserOptionsValue = currentUserOptions();
  const currentUser = await fetchVerifiedQuery(
    client,
    currentUserOptionsValue.queryKey,
    () => client.fetchQuery({ ...currentUserOptionsValue, staleTime: 0 }),
    () => routeIsCurrent(router, routePath, sessionGeneration, client)
  );
  if (!currentUser?.id || !workspaceSlug || !projectId) return undefined;
  const userKey = currentUserOptionsValue.queryKey;
  const userQuery = client.getQueryCache().find({ queryKey: userKey, exact: true });
  const isCurrent = () =>
    routeIsCurrent(router, routePath, sessionGeneration, client, currentUser.id) &&
    userQuery !== undefined &&
    client.getQueryCache().find({ queryKey: userKey, exact: true }) === userQuery &&
    userQuery.state.status === "success" &&
    client.getQueryData<{ id?: string }>(userKey)?.id === currentUser.id;
  const membershipOptions = workspaceMembershipOptions(currentUser.id, workspaceSlug);
  const membership = await fetchVerifiedQuery(
    client,
    membershipOptions.queryKey,
    () => client.fetchQuery({ ...membershipOptions, staleTime: 0 }),
    isCurrent
  );
  if (!isCurrent() || !membership?.is_active || !membership.updated_at || !isWorkspaceRole(membership.role))
    return undefined;
  const rolesOptions = workspaceProjectRolesOptions(
    currentUser.id,
    workspaceSlug,
    membership.role,
    membership.updated_at
  );
  const roles = await fetchVerifiedQuery(
    client,
    rolesOptions.queryKey,
    () => client.fetchQuery({ ...rolesOptions, staleTime: 0 }),
    isCurrent
  );
  const projectRole = roles && effectiveProjectRole(membership.role, roles, projectId);
  if (!isCurrent() || !roles || !projectRole || roles[projectId] === undefined) return undefined;
  return {
    accountId: currentUser.id,
    workspaceSlug,
    projectId,
    membershipRole: membership.role,
    membershipRevision: membership.updated_at,
    projectMembershipRole: roles[projectId],
    projectRole,
    sessionGeneration,
  };
}

async function removeStaleRetryResult(
  client: ReturnType<typeof useQueryClient>,
  queryKey: readonly unknown[],
  request: Promise<unknown>,
  isCurrent: () => boolean
) {
  const query = client.getQueryCache().find({ queryKey, exact: true });
  await request;
  if (!isCurrent() && query && client.getQueryCache().find({ queryKey, exact: true }) === query) {
    client.removeQueries({ queryKey, exact: true });
  }
}

async function retryProjectViewQuery(
  client: ReturnType<typeof useQueryClient>,
  router: ReturnType<typeof useRouter>,
  routePath: string,
  scope: TProjectViewsScope,
  viewId?: string
) {
  const isCurrent = () => routeIsCurrent(router, routePath, scope.sessionGeneration, client, scope.accountId);
  if (viewId) {
    const options = projectViewOptions(client, scope, viewId);
    await removeStaleRetryResult(client, options.queryKey, client.fetchQuery({ ...options, staleTime: 0 }), isCurrent);
    return;
  }
  const options = projectViewsOptions(client, scope);
  await removeStaleRetryResult(client, options.queryKey, client.fetchQuery({ ...options, staleTime: 0 }), isCurrent);
}

function useProjectViewsScope(workspaceSlug: string | undefined, projectId: string | undefined) {
  const client = useQueryClient();
  const router = useRouter();
  const routePath = router.state.location.pathname;
  const currentUser = useCurrentUser();
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const sessionGeneration = useIdentityGeneration(client);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const rawProjectRole =
    permissions.rolesQuery.isSuccess && projectId ? permissions.rolesQuery.data[projectId] : undefined;
  const scope: TProjectViewsScope = {
    accountId: currentUser.isSuccess ? currentUser.data?.id : undefined,
    workspaceSlug,
    projectId,
    membershipRole: membership?.is_active ? permissions.workspaceRole : undefined,
    membershipRevision: membership?.is_active ? membership.updated_at : undefined,
    projectMembershipRole: rawProjectRole,
    projectRole: permissions.projectRole,
    sessionGeneration,
  };
  const access = captureProjectAccessScope(client, scope);
  const readiness = getProjectViewScopeState(currentUser, permissions, membership, scope, workspaceSlug, projectId);
  const retry = async (viewId?: string) => {
    try {
      const refreshedScope = await refetchViewAccess(
        client,
        router,
        routePath,
        sessionGeneration,
        workspaceSlug,
        projectId
      );
      if (!refreshedScope || !routeIsCurrent(router, routePath, sessionGeneration, client, refreshedScope.accountId))
        return;
      await retryProjectViewQuery(client, router, routePath, refreshedScope, viewId);
    } catch {
      // The current user, membership, role or view query exposes the failure for retry.
    }
  };

  return { client, scope, currentUser, permissions, access, ...readiness, retry };
}

export function useProjectViews(workspaceSlug: string | undefined, projectId: string | undefined) {
  const state = useProjectViewsScope(workspaceSlug, projectId);
  const query = useRecordCollection(projectViewsOptions(state.client, state.scope));
  const views = query.isSuccess && state.access.isVerified() ? query.data : undefined;
  return {
    ...query,
    data: views,
    views,
    scope: state.scope,
    isPending: state.isPending || (state.isReady && query.isPending),
    isError: state.isError || (state.isReady && query.isError),
    isDenied: state.isDenied,
    retry: () => state.retry(),
  };
}

export function useProjectView(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  viewId: string | undefined
) {
  const state = useProjectViewsScope(workspaceSlug, projectId);
  const options = projectViewOptions(state.client, state.scope, viewId);
  const query = useQuery(options);
  const queryIdentity = state.client.getQueryCache().find({ queryKey: options.queryKey, exact: true });
  const view: IProjectView | undefined = query.isSuccess && state.access.isVerified() ? query.data : undefined;
  const isScopeCurrent = () =>
    state.access.isCurrent() &&
    queryIdentity !== undefined &&
    state.client.getQueryCache().find({ queryKey: options.queryKey, exact: true }) === queryIdentity &&
    queryIdentity.state.status === "success" &&
    state.client.getQueryData<IProjectView>(options.queryKey)?.id === viewId;
  const isReadScopeCurrent = () => state.access.isVerified() && isScopeCurrent();
  return {
    ...query,
    data: view,
    view,
    scope: state.scope,
    isScopeCurrent,
    isReadScopeCurrent,
    isPending: state.isPending || (state.isReady && query.isPending),
    isError: state.isError || (state.isReady && query.isError),
    isDenied: state.isDenied,
    retry: () => state.retry(viewId),
  };
}

export function useProjectViewMutations(workspaceSlug: string | undefined, projectId: string | undefined) {
  const state = useProjectViewsScope(workspaceSlug, projectId);
  const create = useMutation(
    projectViewMutationOptions.create(state.client, state.scope, state.scope.sessionGeneration)
  );
  const update = useMutation(
    projectViewMutationOptions.update(state.client, state.scope, state.scope.sessionGeneration)
  );
  const remove = useMutation(
    projectViewMutationOptions.delete(state.client, state.scope, state.scope.sessionGeneration)
  );
  return { create, update, remove };
}

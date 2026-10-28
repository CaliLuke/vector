/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRecordCollection } from "@plane/hooks";
import { useCallback, useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import sortBy from "lodash-es/sortBy.js";
import { isWorkspaceRole } from "@/helpers/permission-policy";
import type { TUserPermissions } from "@plane/types";
import { effectiveProjectRole } from "@/helpers/permission-policy";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { currentUserOptions } from "@/queries/identity";
import { workspaceMembershipOptions, workspaceProjectRolesOptions } from "@/queries/permissions";
import {
  projectAccessProbeOptions,
  projectDetailOptions,
  projectMutationOptions,
  projectSortOrderMutationOptions,
  workspaceProjectsOptions,
  workspaceProjectDetailsOptions,
} from "@/queries/projects";
import type { TProjectCatalogueScope } from "@/queries/projects";

function orderedProjectRoles(roles: Record<string, TUserPermissions> | undefined) {
  return roles ? sortBy(Object.entries(roles), ([projectId]) => projectId) : undefined;
}

function membershipDenied(scope: TProjectCatalogueScope, permissions: ReturnType<typeof useWorkspaceProjectRoles>) {
  return (
    permissions.membershipQuery.isSuccess &&
    (!permissions.workspaceRole ||
      !scope.membershipRevision ||
      !permissions.membershipQuery.data.is_active ||
      !isWorkspaceRole(permissions.membershipQuery.data.role))
  );
}

function canProbeProjectAccess(
  scope: TProjectCatalogueScope,
  accountIsReady: boolean,
  membershipResolved: boolean,
  rolesResolved: boolean,
  isMembershipDenied: boolean
) {
  return Boolean(
    accountIsReady &&
    scope.accountId &&
    membershipResolved &&
    rolesResolved &&
    !isMembershipDenied &&
    scope.projectId &&
    scope.projectRole === undefined
  );
}

function projectAccessReady(
  scope: TProjectCatalogueScope,
  accountIsReady: boolean,
  membershipResolved: boolean,
  rolesResolved: boolean,
  isMembershipDenied: boolean
) {
  return Boolean(
    accountIsReady &&
    scope.accountId &&
    membershipResolved &&
    rolesResolved &&
    !isMembershipDenied &&
    (!scope.projectId || scope.projectRole !== undefined)
  );
}

function projectAccessDenied(
  scope: TProjectCatalogueScope,
  account: ReturnType<typeof useCurrentUser>,
  membershipIsDenied: boolean,
  membershipResolved: boolean,
  rolesResolved: boolean,
  isProjectRoleReady: boolean
) {
  return Boolean(
    account.isSuccess &&
    (account.data?.id === undefined ||
      membershipIsDenied ||
      (membershipResolved && rolesResolved && !isProjectRoleReady && scope.projectId))
  );
}

function projectAccess(
  scope: TProjectCatalogueScope,
  account: ReturnType<typeof useCurrentUser>,
  permissions: ReturnType<typeof useWorkspaceProjectRoles>
) {
  const isError = account.isError || permissions.isError;
  const membershipResolved = permissions.membershipQuery.isSuccess;
  const isMembershipDenied = membershipDenied(scope, permissions);
  const rolesResolved = permissions.rolesQuery.isSuccess;
  const isProjectRoleReady = !scope.projectId || scope.projectRole !== undefined;
  const canProbe = canProbeProjectAccess(
    scope,
    account.isSuccess,
    membershipResolved,
    rolesResolved,
    isMembershipDenied
  );
  const isReady = projectAccessReady(scope, account.isSuccess, membershipResolved, rolesResolved, isMembershipDenied);
  const isDenied = projectAccessDenied(
    scope,
    account,
    isMembershipDenied,
    membershipResolved,
    rolesResolved,
    isProjectRoleReady
  );
  return {
    isReady,
    isError,
    isDenied,
    isPending: !isError && !isDenied && (!account.isSuccess || permissions.isPending),
    canProbeProjectAccess: canProbe,
  };
}

function projectCollectionResult<
  TProjects,
  TQuery extends { data: TProjects | undefined; isError: boolean; isPending: boolean },
>(
  query: TQuery,
  projects: TProjects | undefined,
  access: ReturnType<typeof projectAccess>,
  scope: TProjectCatalogueScope,
  retryAccess: () => Promise<unknown>
) {
  return {
    ...query,
    data: projects,
    projects,
    isError: access.isError || (access.isReady && query.isError),
    isPending: access.isPending || (access.isReady && query.isPending),
    accessDenied: access.isDenied,
    accessError: access.isError,
    scope,
    retryAccess,
  };
}

async function retryProjectDetailRead(
  account: ReturnType<typeof useCurrentUser>,
  permissions: ReturnType<typeof useWorkspaceProjectRoles>,
  probeIsActive: boolean,
  probe: { refetch: () => Promise<unknown> },
  query: { refetch: () => Promise<unknown> }
) {
  if (account.isError) await account.refetch();
  else if (permissions.membershipQuery.isError) await permissions.membershipQuery.refetch();
  else if (permissions.rolesQuery.isError) await permissions.rolesQuery.refetch();
  else if (probeIsActive) await probe.refetch();
  else await query.refetch();
}

function projectDetailStatus(input: {
  accessError: boolean;
  accessReady: boolean;
  accessPending: boolean;
  projectId: string | undefined;
  queryError: boolean;
  queryPending: boolean;
  probeIsActive: boolean;
  probeError: boolean;
  probePending: boolean;
}) {
  const hasProject = Boolean(input.projectId);
  return {
    isError:
      input.accessError ||
      (hasProject && input.accessReady && input.queryError) ||
      (input.probeIsActive && input.probeError),
    isPending:
      hasProject &&
      (input.accessPending || (input.accessReady && input.queryPending) || (input.probeIsActive && input.probePending)),
  };
}

function useProjectScope(workspaceSlug: string | undefined, projectId?: string) {
  const client = useQueryClient();
  const account = useCurrentUser();
  const accountId = account.isSuccess ? account.data?.id : undefined;
  const sessionGeneration = useIdentityGeneration(client);
  const permissions = useWorkspaceProjectRoles(workspaceSlug);
  const roles = permissions.rolesQuery.isSuccess ? permissions.rolesQuery.data : undefined;
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const scope = useMemo<TProjectCatalogueScope>(
    () => ({
      accountId,
      sessionGeneration,
      workspaceSlug,
      workspaceRole: permissions.workspaceRole,
      membershipRevision: membership?.updated_at,
      projectRoles: orderedProjectRoles(roles),
      projectId,
      projectMembershipRole: projectId ? roles?.[projectId] : undefined,
      projectRole: projectId ? effectiveProjectRole(permissions.workspaceRole ?? null, roles, projectId) : undefined,
    }),
    [accountId, membership?.updated_at, permissions.workspaceRole, projectId, roles, sessionGeneration, workspaceSlug]
  );
  return { client, account, permissions, scope, access: projectAccess(scope, account, permissions) };
}

export function useWorkspaceProjects(workspaceSlug: string | undefined, enabled = true) {
  const { client, account, permissions, scope, access } = useProjectScope(workspaceSlug);
  const query = useRecordCollection(workspaceProjectsOptions(client, scope, enabled && access.isReady));
  const projects = access.isReady && query.isSuccess ? query.data : undefined;
  const retryAccess = useCallback(async () => {
    if (account.isError) await client.fetchQuery({ ...currentUserOptions(), staleTime: 0 });
    else if (permissions.membershipQuery.isError)
      await client.fetchQuery({
        ...workspaceMembershipOptions(scope.accountId, scope.workspaceSlug),
        staleTime: 0,
      });
    else if (permissions.rolesQuery.isError && scope.workspaceRole && scope.membershipRevision)
      await client.fetchQuery({
        ...workspaceProjectRolesOptions(
          scope.accountId,
          scope.workspaceSlug,
          scope.workspaceRole,
          scope.membershipRevision
        ),
        staleTime: 0,
      });
    else await query.refetch();
  }, [account.isError, client, permissions.membershipQuery, permissions.rolesQuery, query, scope]);
  return projectCollectionResult(query, projects, access, scope, retryAccess);
}

export function useWorkspaceProjectDetails(workspaceSlug?: string, enabled = true) {
  const route = useParams({ strict: false });
  const slug = workspaceSlug ?? route.workspaceSlug;
  const { client, account, permissions, scope, access } = useProjectScope(slug);
  const query = useRecordCollection(workspaceProjectDetailsOptions(client, scope, enabled && access.isReady));
  const projects = access.isReady && query.isSuccess ? query.data : undefined;
  const retryAccess = useCallback(async () => {
    if (account.isError) await account.refetch();
    else if (permissions.membershipQuery.isError) await permissions.membershipQuery.refetch();
    else if (permissions.rolesQuery.isError) await permissions.rolesQuery.refetch();
    else await query.refetch();
  }, [account, permissions, query]);
  return projectCollectionResult(query, projects, access, scope, retryAccess);
}

export function useProjectDetail(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  enabled = true,
  probeMissingProjectRole = false
) {
  const { client, account, permissions, scope, access } = useProjectScope(workspaceSlug, projectId);
  const query = useQuery(projectDetailOptions(client, scope, enabled && access.isReady));
  const probe = useQuery(
    projectAccessProbeOptions(client, scope, enabled && probeMissingProjectRole && access.canProbeProjectAccess)
  );
  const probeIsActive = probeMissingProjectRole && access.canProbeProjectAccess;
  const project = access.isReady && query.isSuccess && query.data?.id === projectId ? query.data : undefined;
  const retryAccess = useCallback(async () => {
    await retryProjectDetailRead(account, permissions, probeIsActive, probe, query);
  }, [account, permissions, probe, probeIsActive, query]);
  const activeRead = probeIsActive ? probe : query;
  const status = projectDetailStatus({
    accessError: access.isError,
    accessReady: access.isReady,
    accessPending: access.isPending,
    projectId,
    queryError: query.isError,
    queryPending: query.isPending,
    probeIsActive,
    probeError: probe.isError,
    probePending: probe.isPending,
  });
  return {
    ...activeRead,
    data: project,
    project,
    isError: status.isError,
    isPending: status.isPending,
    accessDenied: access.isDenied,
    accessError: access.isError,
    accessPending: access.isPending,
    retryAccess,
    scope,
  };
}

export function useRouteProject(enabled = true) {
  const { workspaceSlug, projectId } = useParams({ strict: false });
  return useProjectDetail(workspaceSlug, projectId, enabled);
}

export function useProjectMutations(workspaceSlug?: string, projectId?: string | null) {
  const route = useParams({ strict: false });
  const scopedProjectId = projectId === null ? undefined : (projectId ?? route.projectId);
  const { client, scope } = useProjectScope(workspaceSlug ?? route.workspaceSlug, scopedProjectId);
  return { ...useMutation(projectMutationOptions(client, scope)), scope };
}

export function useProjectSortOrderMutation(workspaceSlug?: string) {
  const route = useParams({ strict: false });
  const { client, scope } = useProjectScope(workspaceSlug ?? route.workspaceSlug);
  return useMutation(projectSortOrderMutationOptions(client, scope));
}

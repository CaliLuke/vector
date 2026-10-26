/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Query, QueryClient } from "@tanstack/react-query";
import sortBy from "lodash-es/sortBy.js";
import { EUserWorkspaceRoles } from "@plane/types";
import type { IWorkspaceMemberMe, TUserPermissions } from "@plane/types";
import { captureIdentityScope, isIdentityScopeCurrent } from "@/queries/identity";
import { permissionKeys } from "@/queries/permissions";
import { memberRosterKeys } from "@/queries/member-rosters";
import { getIdentitySessionGeneration } from "@/queries/identity";
import { effectiveProjectRole } from "@/helpers/permission-policy";
import { projectKeys } from "@/queries/projects";
import { captureProjectRolesQuery, projectRolesRevision } from "@/queries/project-access-scope";

export type WorkspaceInvitationScope = {
  accountId: string | undefined;
  workspaceSlug: string | undefined;
  membershipRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
};

export const workspaceInvitationKeys = {
  list: (scope: WorkspaceInvitationScope) =>
    [
      "member-rosters",
      scope.accountId,
      scope.workspaceSlug,
      scope.membershipRevision,
      scope.membershipRole,
      "invitations",
    ] as const,
};

export function exactQueryCurrent(client: QueryClient, query: Query) {
  return client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) === query;
}

export function captureWorkspaceScope(client: QueryClient, accountId: string | undefined, slug: string | undefined) {
  const identity = captureIdentityScope(client, accountId);
  const membershipKey = permissionKeys.workspaceMembership(accountId, slug);
  const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
  const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
  const isCurrent = () => {
    const currentMembership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
    return Boolean(
      isIdentityScopeCurrent(client, identity) &&
      membershipQuery &&
      client.getQueryCache().find({ queryKey: membershipKey, exact: true }) === membershipQuery &&
      membershipQuery.state.status === "success" &&
      currentMembership?.is_active === true &&
      currentMembership.role === membership?.role &&
      currentMembership.updated_at === membership?.updated_at
    );
  };
  return { identity, membershipQuery, membership, isCurrent };
}

export function captureWorkspaceRosterQueries(
  client: QueryClient,
  accountId: string | undefined,
  slug: string | undefined,
  membership: IWorkspaceMemberMe | undefined
) {
  const key = memberRosterKeys.workspace({
    accountId,
    workspaceSlug: slug,
    membershipRole: membership?.role ?? undefined,
    membershipRevision: membership?.updated_at,
  });
  return client.getQueryCache().findAll({ queryKey: key, exact: true });
}

export function captureWorkspaceInvitationQueries(
  client: QueryClient,
  accountId: string | undefined,
  slug: string | undefined,
  membership: IWorkspaceMemberMe | undefined
) {
  const queryKey = workspaceInvitationKeys.list({
    accountId,
    workspaceSlug: slug,
    membershipRole: membership?.role ?? undefined,
    membershipRevision: membership?.updated_at,
  });
  return client.getQueryCache().findAll({ queryKey, exact: true });
}

export function captureWorkspaceMemberProjectCascade(
  client: QueryClient,
  accountId: string | undefined,
  slug: string,
  membership: IWorkspaceMemberMe | undefined
) {
  const scope = captureWorkspaceScope(client, accountId, slug);
  const prefix = ["member-rosters", accountId, slug, membership?.updated_at, membership?.role] as const;
  const projectRosters = client
    .getQueryCache()
    .findAll({ queryKey: prefix })
    .filter((query) => query.queryKey.length === 8 && query.queryKey[7] === "project");
  const rolesSnapshot = captureProjectRolesQuery(
    client,
    accountId,
    slug,
    membership?.role ?? undefined,
    membership?.updated_at
  );
  const { queryKey: rolesKey, query: rolesQuery, roles } = rolesSnapshot;
  const catalogueScope = {
    accountId,
    sessionGeneration: getIdentitySessionGeneration(client),
    workspaceSlug: slug,
    workspaceRole: membership?.role ?? undefined,
    membershipRevision: membership?.updated_at,
    projectRoles: roles ? sortBy(Object.entries(roles), ([projectId]) => projectId) : undefined,
  };
  const projectCatalogueKeys = roles
    ? [
        projectKeys.workspace(catalogueScope, "partial"),
        projectKeys.workspace(catalogueScope, "full"),
        ...Object.keys(roles).map((projectId) => {
          const projectMembershipRole = roles[projectId];
          return projectKeys.detail({
            ...catalogueScope,
            projectId,
            projectMembershipRole,
            projectRole: effectiveProjectRole(catalogueScope.workspaceRole ?? null, roles, projectId),
          });
        }),
      ]
    : [];
  const projectQueries = projectCatalogueKeys.flatMap((queryKey) => {
    const query = client.getQueryCache().find({ queryKey, exact: true });
    return query ? [query] : [];
  });
  const rolesRevision = rolesSnapshot.revision;
  const isCurrent = () => {
    const currentRoles = rolesQuery
      ? client.getQueryCache().find({ queryKey: rolesKey, exact: true }) === rolesQuery &&
        rolesQuery.state.status === "success" &&
        projectRolesRevision(client.getQueryData<Record<string, TUserPermissions>>(rolesKey)) === rolesRevision
      : projectQueries.length === 0;
    return (
      scope.isCurrent() &&
      currentRoles &&
      projectQueries.every((query) => client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) === query)
    );
  };
  return {
    isCurrent,
    queries: [...projectRosters, ...projectQueries],
  };
}

export async function invalidateCaptured(client: QueryClient, captured: Query[], isScopeCurrent: () => boolean) {
  await Promise.allSettled(
    captured.map((query) =>
      isScopeCurrent() && exactQueryCurrent(client, query)
        ? client.invalidateQueries({ queryKey: query.queryKey, exact: true })
        : Promise.resolve()
    )
  );
}

export async function cancelAndInvalidateCaptured(
  client: QueryClient,
  captured: Query[],
  isScopeCurrent: () => boolean
) {
  await Promise.allSettled(
    captured.map(async (query) => {
      if (!isScopeCurrent() || !exactQueryCurrent(client, query)) return;
      await client.cancelQueries({ queryKey: query.queryKey, exact: true });
      if (!isScopeCurrent() || !exactQueryCurrent(client, query)) return;
      await client.invalidateQueries({ queryKey: query.queryKey, exact: true });
    })
  );
}

export async function updateCaptured<T>(
  client: QueryClient,
  captured: Query[],
  isScopeCurrent: () => boolean,
  update: (current: T | undefined) => T | undefined
) {
  await Promise.all(
    captured.map(async (query) => {
      if (!isScopeCurrent() || !exactQueryCurrent(client, query)) return;
      await client.cancelQueries({ queryKey: query.queryKey, exact: true });
      if (!isScopeCurrent() || !exactQueryCurrent(client, query)) return;
      client.setQueryData<T>(query.queryKey, update);
    })
  );
}

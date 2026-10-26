/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { EUserWorkspaceRoles, IWorkspaceMemberMe } from "@plane/types";
import { captureIdentityScope, isIdentityScopeVerified } from "@/queries/identity";
import { WorkspaceService } from "@/services/workspace.service";
import { isWorkspaceRole } from "@/helpers/permission-policy";

const service = new WorkspaceService();

export const permissionKeys = {
  workspaceMembership: (userId: string | undefined, slug: string | undefined) =>
    ["permissions", userId, slug, "workspace-membership"] as const,
  workspaceProjectRoles: (
    userId: string | undefined,
    slug: string | undefined,
    role: EUserWorkspaceRoles | undefined,
    membershipUpdatedAt: string | undefined
  ) => ["permissions", userId, slug, "project-roles", role, membershipUpdatedAt] as const,
};

const exactQueryCurrent = (client: QueryClient, key: readonly unknown[], query: object | undefined) =>
  Boolean(query && client.getQueryCache().find({ queryKey: key, exact: true }) === query);

export function workspaceMembershipOptions(userId: string | undefined, slug: string | undefined) {
  const queryKey = permissionKeys.workspaceMembership(userId, slug);
  return queryOptions({
    queryKey,
    queryFn:
      userId && slug
        ? async ({ client, signal }) => {
            const identity = captureIdentityScope(client, userId);
            const query = client.getQueryCache().find({ queryKey, exact: true });
            const isCurrent = () =>
              isIdentityScopeVerified(client, identity) && exactQueryCurrent(client, queryKey, query);
            if (!isCurrent()) throw new Error("The account session changed.");
            const data = await service.workspaceMemberMe(slug, signal);
            if (!isCurrent()) throw new Error("The account session changed.");
            return data;
          }
        : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    retry: false,
  });
}

export function workspaceProjectRolesOptions(
  userId: string | undefined,
  slug: string | undefined,
  role: EUserWorkspaceRoles | undefined,
  membershipUpdatedAt: string | undefined
) {
  const queryKey = permissionKeys.workspaceProjectRoles(userId, slug, role, membershipUpdatedAt);
  const membershipKey = permissionKeys.workspaceMembership(userId, slug);
  return queryOptions({
    queryKey,
    queryFn:
      userId && slug && isWorkspaceRole(role) && membershipUpdatedAt
        ? async ({ client, signal }) => {
            const identity = captureIdentityScope(client, userId);
            const rolesQuery = client.getQueryCache().find({ queryKey, exact: true });
            const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
            const isCurrent = () => {
              const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
              return (
                isIdentityScopeVerified(client, identity) &&
                exactQueryCurrent(client, queryKey, rolesQuery) &&
                exactQueryCurrent(client, membershipKey, membershipQuery) &&
                membershipQuery?.state.status === "success" &&
                membership?.is_active === true &&
                membership.role === role &&
                membership.updated_at === membershipUpdatedAt
              );
            };
            if (!isCurrent()) throw new Error("The workspace permission scope changed.");
            const data = await service.getWorkspaceUserProjectsRole(slug, signal);
            if (!isCurrent()) throw new Error("The workspace permission scope changed.");
            return data;
          }
        : skipToken,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    retry: false,
  });
}

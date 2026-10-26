/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions } from "@tanstack/react-query";
import type { Query, QueryClient } from "@tanstack/react-query";
import type { IUserProjectsRole, IWorkspaceMemberMe } from "@plane/types";
import userService from "@/services/user.service";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "./identity";
import { permissionKeys } from "./permissions";

/** Capture before the durable write; never mutate a replacement query or account. */
export function capturePermissionChanges(client: QueryClient, userId: string | undefined, slug: string) {
  const identity = captureIdentityScope(client, userId);
  const workspace = client
    .getQueryCache()
    .find({ queryKey: permissionKeys.workspaceMembership(userId, slug), exact: true });
  const projectRoles = client.getQueryCache().findAll({ queryKey: ["permissions", userId, slug, "project-roles"] });
  const isCurrent = (query: Query | undefined): query is Query =>
    Boolean(
      query &&
      isIdentityScopeCurrent(client, identity) &&
      client.getQueryCache().find({ queryKey: query.queryKey, exact: true }) === query
    );
  const refresh = (query: Query | undefined) =>
    isCurrent(query) ? client.invalidateQueries({ queryKey: query.queryKey, exact: true }) : Promise.resolve();
  return {
    refreshWorkspace: () => refresh(workspace),
    refreshProjectRoles: () => Promise.all(projectRoles.map(refresh)),
    resetProjectRoles: () =>
      Promise.all(
        projectRoles.map((query) =>
          isCurrent(query) ? client.resetQueries({ queryKey: query.queryKey, exact: true }) : Promise.resolve()
        )
      ),
    removeProjectRole: async (projectId: string) => {
      await Promise.all(
        projectRoles.map(async (query) => {
          if (!isCurrent(query)) return;
          await client.cancelQueries({ queryKey: query.queryKey, exact: true });
          if (!isCurrent(query)) return;
          client.setQueryData<IUserProjectsRole>(query.queryKey, (cached) => {
            if (!cached) return undefined;
            const next = { ...cached };
            delete next[projectId];
            return next;
          });
          await refresh(query);
        })
      );
    },
    leaveWorkspace: async () => {
      if (!isCurrent(workspace)) return;
      await client.cancelQueries({ queryKey: workspace.queryKey, exact: true });
      if (!isCurrent(workspace)) return;
      client.setQueryData<IWorkspaceMemberMe>(workspace.queryKey, (cached) =>
        cached ? { ...cached, is_active: false, role: null } : undefined
      );
    },
  };
}

export function joinProjectOptions(client: QueryClient, userId: string | undefined, slug: string, projectId: string) {
  const identity = captureIdentityScope(client, userId);
  const generation = getIdentitySessionGeneration(client);
  return mutationOptions({
    mutationKey: ["permissions", userId, generation, slug, projectId, "join"],
    scope: { id: `permissions/${userId}/${generation}/${slug}/${projectId}` },
    retry: false,
    onMutate: () => capturePermissionChanges(client, userId, slug),
    mutationFn: async () => {
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      await userService.joinProject(slug, [projectId]);
    },
    onSuccess: async (_data, _variables, changes) => {
      // The acknowledgement has no role. Read canonical permissions; read failures remain query errors.
      await changes.refreshProjectRoles();
    },
  });
}

export function leaveProjectOptions(client: QueryClient, userId: string | undefined, slug: string, projectId: string) {
  const identity = captureIdentityScope(client, userId);
  const generation = getIdentitySessionGeneration(client);
  return mutationOptions({
    mutationKey: ["permissions", userId, generation, slug, projectId, "leave"],
    scope: { id: `permissions/${userId}/${generation}/${slug}/${projectId}` },
    retry: false,
    onMutate: () => capturePermissionChanges(client, userId, slug),
    mutationFn: async () => {
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      await userService.leaveProject(slug, projectId);
    },
    onSuccess: async (_data, _variables, changes) => {
      await changes.removeProjectRole(projectId);
    },
  });
}

export function leaveWorkspaceOptions(client: QueryClient, userId: string | undefined, slug: string) {
  const identity = captureIdentityScope(client, userId);
  const generation = getIdentitySessionGeneration(client);
  return mutationOptions({
    mutationKey: ["permissions", userId, generation, slug, "leave"],
    scope: { id: `permissions/${userId}/${generation}/${slug}` },
    retry: false,
    onMutate: () => capturePermissionChanges(client, userId, slug),
    mutationFn: async () => {
      if (!isIdentityScopeVerified(client, identity)) throw new Error("The account session changed or is unverified.");
      await userService.leaveWorkspace(slug);
    },
    onSuccess: async (_data, _variables, changes) => {
      await changes.leaveWorkspace();
    },
  });
}

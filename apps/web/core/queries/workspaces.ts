/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { IUserSettings, IWorkspace } from "@plane/types";
import { WorkspaceService } from "@/services/workspace.service";
import { captureIdentityScope, isIdentityScopeCurrent, isIdentityScopeVerified } from "./identity";

const service = new WorkspaceService();
export const workspaceKeys = {
  list: (userId: string | undefined) => ["workspaces", userId] as const,
};

export function workspacesOptions(userId: string | undefined) {
  return queryOptions({
    queryKey: workspaceKeys.list(userId),
    staleTime: 5 * 60_000,
    refetchOnMount: true,
    queryFn: userId
      ? async ({ client, signal }) => {
          const scope = captureIdentityScope(client, userId);
          if (!isIdentityScopeVerified(client, scope))
            throw new Error("The account session changed before workspaces could load.");
          const workspaces = await service.userWorkspaces(signal);
          if (!isIdentityScopeVerified(client, scope))
            throw new Error("The account session changed while workspaces loaded.");
          return workspaces;
        }
      : skipToken,
  });
}

export function getWorkspaceDestination(workspaces: IWorkspace[], settings: IUserSettings | undefined) {
  const slug = [settings?.workspace?.last_workspace_slug, settings?.workspace?.fallback_workspace_slug].find(
    (candidate) => candidate && workspaces.some((workspace) => workspace.slug === candidate)
  );
  return slug ? `/${slug}` : "/create-workspace";
}

function workspaceMutationScope(client: QueryClient, userId: string | undefined) {
  const identity = captureIdentityScope(client, userId);
  const queryKey = workspaceKeys.list(userId);
  const findQuery = () => client.getQueryCache().find({ queryKey, exact: true });
  return {
    assertCurrent: () => {
      if (!isIdentityScopeVerified(client, identity))
        throw new Error("The account session has changed or is unverified.");
    },
    capture: findQuery,
    commit: async (query: ReturnType<typeof findQuery>, change: (workspaces: IWorkspace[]) => IWorkspace[]) => {
      const isCurrent = () => isIdentityScopeCurrent(client, identity) && query && query === findQuery();
      if (!isCurrent()) return;
      await client.cancelQueries({ queryKey, exact: true });
      if (!isCurrent()) return;
      // An unloaded list must be fetched in full; a mutation response is not a complete catalogue.
      if (client.getQueryData(queryKey) === undefined) {
        await client.invalidateQueries({ queryKey, exact: true });
        return;
      }
      if (isCurrent()) client.setQueryData<IWorkspace[]>(queryKey, (previous) => previous && change(previous));
    },
  };
}

export function createWorkspaceOptions(client: QueryClient, userId: string | undefined) {
  const scope = workspaceMutationScope(client, userId);
  return mutationOptions({
    mutationKey: ["workspaces", userId, "create"],
    scope: { id: `workspaces/${userId}/create` },
    retry: false,
    onMutate: scope.capture,
    mutationFn: async (change: Partial<IWorkspace>) => {
      scope.assertCurrent();
      return service.createWorkspace(change);
    },
    onSuccess: (saved, _change, query) =>
      scope.commit(query, (previous) => [...previous.filter((workspace) => workspace.id !== saved.id), saved]),
  });
}

export function updateWorkspaceOptions(client: QueryClient, userId: string | undefined, slug: string | undefined) {
  const scope = workspaceMutationScope(client, userId);
  return mutationOptions({
    mutationKey: ["workspaces", userId, slug, "update"],
    scope: { id: `workspaces/${userId}/${slug}` },
    retry: false,
    onMutate: scope.capture,
    mutationFn: async (change: Partial<IWorkspace>) => {
      scope.assertCurrent();
      if (!slug) throw new Error("A workspace is required.");
      const existing = client
        .getQueryData<IWorkspace[]>(workspaceKeys.list(userId))
        ?.find((workspace) => workspace.slug === slug);
      const saved = await service.updateWorkspace(slug, change);
      if (existing && existing.id !== saved.id) throw new Error("The server returned a different workspace.");
      return saved;
    },
    onSuccess: (saved, _change, query) =>
      scope.commit(query, (previous) => previous.map((workspace) => (workspace.id === saved.id ? saved : workspace))),
  });
}

export function deleteWorkspaceOptions(client: QueryClient, userId: string | undefined, slug: string | undefined) {
  const scope = workspaceMutationScope(client, userId);
  return mutationOptions({
    mutationKey: ["workspaces", userId, slug, "delete"],
    scope: { id: `workspaces/${userId}/${slug}` },
    retry: false,
    onMutate: scope.capture,
    mutationFn: async () => {
      scope.assertCurrent();
      if (!slug) throw new Error("A workspace is required.");
      await service.deleteWorkspace(slug);
    },
    onSuccess: (_saved, _change, query) =>
      scope.commit(query, (previous) => previous.filter((workspace) => workspace.slug !== slug)),
  });
}

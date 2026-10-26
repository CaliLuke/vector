/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { EUserWorkspaceRoles, IWorkspaceMemberMe, IWorkspaceView } from "@plane/types";
import {
  captureIdentityScope,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import { permissionKeys } from "@/queries/permissions";
import { WorkspaceService } from "@/services/workspace.service";
import { isWorkspaceRole } from "@/helpers/permission-policy";

const service = new WorkspaceService();

export type WorkspaceViewsScope = {
  accountId: string | undefined;
  workspaceSlug: string | undefined;
  workspaceRole: EUserWorkspaceRoles | undefined;
  membershipRevision: string | undefined;
  sessionGeneration: number;
};

export const workspaceViewKeys = {
  list: (scope: WorkspaceViewsScope) =>
    [
      "workspace-views",
      scope.accountId,
      scope.workspaceSlug,
      scope.workspaceRole,
      scope.membershipRevision,
      scope.sessionGeneration,
      "list",
    ] as const,
  mutations: (scope: WorkspaceViewsScope, generation: number) => [
    "workspace-views",
    scope.accountId,
    scope.workspaceSlug,
    scope.workspaceRole,
    scope.membershipRevision,
    scope.sessionGeneration,
    generation,
  ],
};

function exactQueryCurrent(client: QueryClient, queryKey: readonly unknown[], query: object | undefined) {
  return Boolean(query && client.getQueryCache().find({ queryKey, exact: true }) === query);
}

function workspaceMembershipMatches(
  client: QueryClient,
  scope: WorkspaceViewsScope,
  capturedMembershipQuery: object | undefined
) {
  if (!scope.accountId || !scope.workspaceSlug || !isWorkspaceRole(scope.workspaceRole) || !scope.membershipRevision)
    return false;

  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
  const membership = client.getQueryData<IWorkspaceMemberMe>(membershipKey);
  return (
    membershipQuery === capturedMembershipQuery &&
    membershipQuery?.state.status === "success" &&
    membership?.is_active === true &&
    membership.role === scope.workspaceRole &&
    membership.updated_at === scope.membershipRevision
  );
}

function scopeMembershipIsCurrent(
  client: QueryClient,
  scope: WorkspaceViewsScope,
  identity: ReturnType<typeof captureIdentityScope>,
  capturedMembershipQuery: object | undefined
) {
  return Boolean(
    identity &&
    identity.generation === scope.sessionGeneration &&
    isIdentityScopeCurrent(client, identity) &&
    workspaceMembershipMatches(client, scope, capturedMembershipQuery)
  );
}

function scopeMembershipIsVerified(
  client: QueryClient,
  scope: WorkspaceViewsScope,
  identity: ReturnType<typeof captureIdentityScope>,
  capturedMembershipQuery: object | undefined
) {
  return (
    scopeMembershipIsCurrent(client, scope, identity, capturedMembershipQuery) &&
    isIdentityScopeVerified(client, identity)
  );
}

export function workspaceViewsOptions(client: QueryClient, scope: WorkspaceViewsScope) {
  const { workspaceSlug } = scope;
  const queryKey = workspaceViewKeys.list(scope);
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  return queryOptions({
    queryKey,
    staleTime: 30_000,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    retry: false,
    queryFn:
      scope.accountId &&
      workspaceSlug &&
      isWorkspaceRole(scope.workspaceRole) &&
      scope.membershipRevision &&
      getIdentitySessionGeneration(client) === scope.sessionGeneration
        ? async ({ client: queryClient, signal }) => {
            const identity = captureIdentityScope(queryClient, scope.accountId);
            const query = queryClient.getQueryCache().find({ queryKey, exact: true });
            const membershipQuery = queryClient.getQueryCache().find({ queryKey: membershipKey, exact: true });
            const isCurrent = () =>
              exactQueryCurrent(queryClient, queryKey, query) &&
              scopeMembershipIsVerified(queryClient, scope, identity, membershipQuery);
            if (!isCurrent()) throw new Error("Workspace access changed before views could be loaded.");
            const views = await service.getAllViews(workspaceSlug, signal);
            if (!isCurrent()) throw new Error("Workspace access changed while views were loading.");
            return views;
          }
        : skipToken,
  });
}

type CapturedViewMutation = {
  identity: ReturnType<typeof captureIdentityScope>;
  membershipQuery: object | undefined;
  queryKey: ReturnType<typeof workspaceViewKeys.list>;
  query: object | undefined;
};

function captureViewMutation(client: QueryClient, scope: WorkspaceViewsScope) {
  const queryKey = workspaceViewKeys.list(scope);
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  const identity = captureIdentityScope(client, scope.accountId);
  const membershipQuery = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
  const assertCurrent = () => {
    if (!scopeMembershipIsVerified(client, scope, identity, membershipQuery))
      throw new Error("Workspace access is unavailable or has changed.");
  };
  const capture = (): CapturedViewMutation => ({
    identity,
    membershipQuery,
    queryKey,
    query: client.getQueryCache().find({ queryKey, exact: true }),
  });
  return { assertCurrent, capture };
}

async function reconcileCapturedViewList(
  client: QueryClient,
  scope: WorkspaceViewsScope,
  captured: CapturedViewMutation | undefined,
  change: (views: IWorkspaceView[]) => IWorkspaceView[]
) {
  if (!captured) return;
  try {
    if (!exactQueryCurrent(client, captured.queryKey, captured.query)) return;
    await client.cancelQueries({ queryKey: captured.queryKey, exact: true });
    if (!scopeMembershipIsCurrent(client, scope, captured.identity, captured.membershipQuery)) return;
    if (!exactQueryCurrent(client, captured.queryKey, captured.query)) return;

    const current = client.getQueryData<IWorkspaceView[]>(captured.queryKey);
    if (current === undefined) {
      void client
        .invalidateQueries({ queryKey: captured.queryKey, exact: true, refetchType: "active" })
        .catch(() => {});
      return;
    }
    client.setQueryData<IWorkspaceView[]>(captured.queryKey, change(current));
    // A refresh failure cannot turn an accepted view write into a failed submission.
    void client.invalidateQueries({ queryKey: captured.queryKey, exact: true, refetchType: "active" }).catch(() => {});
  } catch {
    // The server write has already succeeded; cache refresh is best-effort.
  }
}

function createWorkspaceViewOptions(client: QueryClient, scope: WorkspaceViewsScope, generation: number) {
  const mutation = captureViewMutation(client, scope);
  return mutationOptions({
    mutationKey: [...workspaceViewKeys.mutations(scope, generation), "create"],
    scope: { id: `workspace-views/${scope.accountId}/${scope.workspaceSlug}/${generation}` },
    retry: false,
    onMutate: mutation.capture,
    mutationFn: async (data: Partial<IWorkspaceView>) => {
      mutation.assertCurrent();
      if (!scope.workspaceSlug) throw new Error("A workspace is required.");
      return service.createView(scope.workspaceSlug, data);
    },
    onSuccess: (saved, _data, captured) =>
      reconcileCapturedViewList(client, scope, captured, (views) => [
        ...views.filter((view) => view.id !== saved.id),
        saved,
      ]),
  });
}

function updateWorkspaceViewOptions(client: QueryClient, scope: WorkspaceViewsScope, generation: number) {
  const mutation = captureViewMutation(client, scope);
  return mutationOptions({
    mutationKey: [...workspaceViewKeys.mutations(scope, generation), "update"],
    scope: { id: `workspace-views/${scope.accountId}/${scope.workspaceSlug}/${generation}` },
    retry: false,
    onMutate: mutation.capture,
    mutationFn: async ({ viewId, data }: { viewId: string; data: Partial<IWorkspaceView> }) => {
      mutation.assertCurrent();
      if (!scope.workspaceSlug) throw new Error("A workspace is required.");
      return service.updateView(scope.workspaceSlug, viewId, data);
    },
    onSuccess: (saved, { viewId }, captured) =>
      reconcileCapturedViewList(client, scope, captured, (views) =>
        views.map((view) => (view.id === viewId ? saved : view))
      ),
  });
}

function deleteWorkspaceViewOptions(client: QueryClient, scope: WorkspaceViewsScope, generation: number) {
  const mutation = captureViewMutation(client, scope);
  return mutationOptions({
    mutationKey: [...workspaceViewKeys.mutations(scope, generation), "delete"],
    scope: { id: `workspace-views/${scope.accountId}/${scope.workspaceSlug}/${generation}` },
    retry: false,
    onMutate: mutation.capture,
    mutationFn: async (viewId: string) => {
      mutation.assertCurrent();
      if (!scope.workspaceSlug) throw new Error("A workspace is required.");
      await service.deleteView(scope.workspaceSlug, viewId);
    },
    onSuccess: (_saved, viewId, captured) =>
      reconcileCapturedViewList(client, scope, captured, (views) => views.filter((view) => view.id !== viewId)),
  });
}

export const workspaceViewMutationOptions = {
  create: createWorkspaceViewOptions,
  update: updateWorkspaceViewOptions,
  delete: deleteWorkspaceViewOptions,
};

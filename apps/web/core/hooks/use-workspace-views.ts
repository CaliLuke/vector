/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors.
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useMemo, useRef } from "react";
import { useRouter, useRouterState } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { permissionKeys, workspaceMembershipOptions } from "@/queries/permissions";
import {
  captureIdentityScope,
  currentUserOptions,
  getIdentitySessionGeneration,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "@/queries/identity";
import { workspaceViewMutationOptions, workspaceViewsOptions } from "@/queries/global-views";
import type { WorkspaceViewsScope } from "@/queries/global-views";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspacePermissions } from "@/hooks/use-permissions";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { isWorkspaceRole } from "@/helpers/permission-policy";
import type { IWorkspaceMemberMe, IWorkspaceView } from "@plane/types";

function useWorkspaceViewsScope(workspaceSlug: string | undefined) {
  const client = useQueryClient();
  const currentUser = useCurrentUser();
  const permissions = useWorkspacePermissions(workspaceSlug);
  const generation = useIdentityGeneration(client);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const scope: WorkspaceViewsScope = {
    accountId: currentUser.isSuccess ? currentUser.data?.id : undefined,
    workspaceSlug,
    workspaceRole: membership?.is_active === true && isWorkspaceRole(membership.role) ? membership.role : undefined,
    membershipRevision: membership?.is_active === true ? membership.updated_at : undefined,
    sessionGeneration: generation,
  };
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, workspaceSlug);
  const membershipQueryIdentity = client.getQueryCache().find({ queryKey: membershipKey, exact: true });
  const canRead = Boolean(
    currentUser.isSuccess &&
    scope.accountId &&
    workspaceSlug &&
    scope.workspaceRole !== undefined &&
    scope.membershipRevision
  );

  return { client, scope, permissions, currentUser, generation, membershipQueryIdentity, canRead };
}

export function useWorkspaceViews(workspaceSlug: string | undefined) {
  const state = useWorkspaceViewsScope(workspaceSlug);
  const query = useQuery(workspaceViewsOptions(state.client, state.scope));
  const currentMembership = state.permissions.membershipQuery.isSuccess
    ? state.permissions.membershipQuery.data
    : undefined;
  const hasCurrentMembership =
    state.canRead &&
    state.client.getQueryCache().find({
      queryKey: permissionKeys.workspaceMembership(state.scope.accountId, workspaceSlug),
      exact: true,
    }) === state.membershipQueryIdentity &&
    currentMembership?.is_active === true &&
    currentMembership.role === state.scope.workspaceRole &&
    currentMembership.updated_at === state.scope.membershipRevision;
  const views = query.isSuccess && hasCurrentMembership ? query.data : undefined;
  const retry = async () => {
    const { client, generation } = state;
    try {
      const userOptions = currentUserOptions();
      const user =
        client.getQueryState(userOptions.queryKey)?.status === "success"
          ? client.getQueryData(userOptions.queryKey)
          : await client.fetchQuery({ ...userOptions, staleTime: 0 });
      if (!user?.id || getIdentitySessionGeneration(client) !== generation) return;
      const identity = captureIdentityScope(client, user.id);
      const membershipOptions = workspaceMembershipOptions(user.id, workspaceSlug);
      const membership =
        client.getQueryState(membershipOptions.queryKey)?.status === "success"
          ? client.getQueryData<IWorkspaceMemberMe>(membershipOptions.queryKey)
          : await client.fetchQuery({ ...membershipOptions, staleTime: 0 });
      if (
        !isIdentityScopeVerified(client, identity) ||
        membership?.is_active !== true ||
        !isWorkspaceRole(membership.role) ||
        !membership.updated_at
      )
        return;
      // The observer may still hold skipToken from the render before prerequisite recovery.
      await client.fetchQuery({
        ...workspaceViewsOptions(client, {
          accountId: user.id,
          workspaceSlug,
          workspaceRole: membership.role,
          membershipRevision: membership.updated_at,
          sessionGeneration: generation,
        }),
        staleTime: 0,
      });
    } catch {
      // Native query state exposes the failed prerequisite or list read for the next retry.
    }
  };

  return {
    ...query,
    data: views,
    views,
    canRead: state.canRead,
    membershipQuery: state.permissions.membershipQuery,
    currentUserQuery: state.currentUser,
    retry,
  };
}

export function getWorkspaceViewsLoadState(query: ReturnType<typeof useWorkspaceViews>, requireViews = true) {
  return {
    isLoading:
      query.currentUserQuery.isPending ||
      query.membershipQuery.isPending ||
      (requireViews && query.canRead && query.isPending),
    isError: query.currentUserQuery.isError || query.membershipQuery.isError || (requireViews && query.isError),
  };
}

export function useWorkspaceView(workspaceSlug: string | undefined, viewId: string | undefined) {
  const list = useWorkspaceViews(workspaceSlug);
  const view = useMemo<IWorkspaceView | undefined>(
    () => list.views?.find((item) => item.id === viewId),
    [list.views, viewId]
  );
  return { ...list, view };
}

export function useWorkspaceViewMutations(workspaceSlug: string | undefined) {
  const state = useWorkspaceViewsScope(workspaceSlug);
  const create = useMutation(workspaceViewMutationOptions.create(state.client, state.scope, state.generation));
  const update = useMutation(workspaceViewMutationOptions.update(state.client, state.scope, state.generation));
  const remove = useMutation(workspaceViewMutationOptions.delete(state.client, state.scope, state.generation));
  return { create, update, remove };
}

/** Capture feedback for this committed dialog, route, permission and account session. */
export function useWorkspaceViewActionScope(workspaceSlug: string | undefined, isOpen: boolean, viewId?: string) {
  const state = useWorkspaceViewsScope(workspaceSlug);
  const router = useRouter();
  const href = useRouterState({ select: (routerState) => routerState.location.href });
  const token = useMemo(
    () => ({ workspaceSlug, isOpen, viewId, href, accountId: state.scope.accountId, generation: state.generation }),
    [workspaceSlug, isOpen, viewId, href, state.scope.accountId, state.generation]
  );
  const active = useRef<object | undefined>(undefined);
  useLayoutEffect(() => {
    active.current = token;
    return () => {
      active.current = undefined;
    };
  }, [token]);
  return () => {
    const identity = captureIdentityScope(state.client, state.scope.accountId);
    return () => {
      const membership = state.client.getQueryData<IWorkspaceMemberMe>(
        permissionKeys.workspaceMembership(state.scope.accountId, workspaceSlug)
      );
      return (
        isOpen &&
        active.current === token &&
        router.state.location.href === href &&
        isIdentityScopeCurrent(state.client, identity) &&
        state.client.getQueryCache().find({
          queryKey: permissionKeys.workspaceMembership(state.scope.accountId, workspaceSlug),
          exact: true,
        }) === state.membershipQueryIdentity &&
        state.client.getQueryState(permissionKeys.workspaceMembership(state.scope.accountId, workspaceSlug))?.status ===
          "success" &&
        membership?.is_active === true &&
        membership.role === state.scope.workspaceRole &&
        membership.updated_at === state.scope.membershipRevision
      );
    };
  };
}

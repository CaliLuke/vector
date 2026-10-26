/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { IUser } from "@plane/types";
import userService from "@/services/user.service";

export const identityKeys = {
  currentUser: () => ["identity", "current-user"] as const,
  profile: (userId: string | undefined) => ["identity", userId, "profile"] as const,
  settings: (userId: string | undefined) => ["identity", userId, "settings"] as const,
  emailNotificationSettings: (userId: string | undefined) => ["identity", userId, "email-notifications"] as const,
};

const generations = new WeakMap<QueryClient, number>();
const sessionListeners = new WeakMap<QueryClient, Set<() => void>>();

export function subscribeIdentitySession(client: QueryClient, listener: () => void) {
  const listeners = sessionListeners.get(client) ?? new Set<() => void>();
  listeners.add(listener);
  sessionListeners.set(client, listeners);
  return () => listeners.delete(listener);
}

export function getIdentitySessionGeneration(client: QueryClient) {
  return generations.get(client) ?? 0;
}

export function advanceIdentitySession(client: QueryClient) {
  generations.set(client, getIdentitySessionGeneration(client) + 1);
  sessionListeners.get(client)?.forEach((listener) => listener());
}

export function currentUserOptions() {
  return queryOptions({
    queryKey: identityKeys.currentUser(),
    queryFn: ({ signal }) => userService.currentUser(signal),
    staleTime: 5 * 60_000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    retry: false,
  });
}

export function getCurrentUserId(client: QueryClient) {
  return client.getQueryData(currentUserOptions().queryKey)?.id;
}

export function captureIdentityScope(client: QueryClient, userId: string | undefined) {
  const query = client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true });
  if (!userId || getCurrentUserId(client) !== userId || !query) return undefined;
  return { userId, query, generation: getIdentitySessionGeneration(client) };
}

export function isIdentityScopeCurrent(client: QueryClient, scope: ReturnType<typeof captureIdentityScope>) {
  return Boolean(
    scope &&
    scope.generation === getIdentitySessionGeneration(client) &&
    client.getQueryCache().find({ queryKey: identityKeys.currentUser(), exact: true }) === scope.query &&
    getCurrentUserId(client) === scope.userId
  );
}

/** A read or a new write may start only while the cached identity is verified. */
export function isIdentityScopeVerified(client: QueryClient, scope: ReturnType<typeof captureIdentityScope>) {
  return isIdentityScopeCurrent(client, scope) && scope?.query.state.status === "success";
}

export function updateCurrentUserOptions(client: QueryClient, userId: string | undefined) {
  const scope = captureIdentityScope(client, userId);
  return mutationOptions({
    mutationKey: ["identity", userId, "update"],
    scope: { id: `identity/${userId}` },
    retry: false,
    mutationFn: async (change: Partial<IUser>) => {
      if (!isIdentityScopeVerified(client, scope)) throw new Error("The account session has changed or is unverified.");
      const saved = await userService.updateUser(change);
      if (saved.id !== userId) throw new Error("The server returned a different account.");
      return saved;
    },
    onSuccess: async (saved) => {
      if (!isIdentityScopeCurrent(client, scope)) return;
      await client.cancelQueries({ queryKey: identityKeys.currentUser(), exact: true });
      if (isIdentityScopeCurrent(client, scope)) client.setQueryData(currentUserOptions().queryKey, saved);
    },
  });
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { queryOptions, mutationOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { IUserEmailNotificationSettings } from "@plane/types";
import userService from "@/services/user.service";
import { captureIdentityScope, identityKeys, isIdentityScopeCurrent, isIdentityScopeVerified } from "./identity";

export function emailNotificationSettingsOptions(userId: string | undefined) {
  return queryOptions({
    queryKey: identityKeys.emailNotificationSettings(userId),
    staleTime: 5 * 60_000,
    refetchOnMount: false,
    queryFn: userId
      ? async ({ client, signal }) => {
          const scope = captureIdentityScope(client, userId);
          if (!isIdentityScopeVerified(client, scope))
            throw new Error("The account session changed before notification preferences could load.");
          const settings = await userService.currentUserEmailNotificationSettings(signal);
          if (!isIdentityScopeVerified(client, scope))
            throw new Error("The account session changed while notification preferences loaded.");
          return settings;
        }
      : skipToken,
  });
}

export function emailNotificationSettingsMutationOptions(client: QueryClient, userId: string | undefined) {
  const queryKey = identityKeys.emailNotificationSettings(userId);
  const scope = captureIdentityScope(client, userId);
  return mutationOptions({
    mutationKey: queryKey,
    scope: { id: `identity/${userId}/email-notifications` },
    retry: false,
    mutationFn: async (change: Partial<IUserEmailNotificationSettings>) => {
      if (!isIdentityScopeVerified(client, scope)) throw new Error("The account session has changed or is unverified.");
      return userService.updateCurrentUserEmailNotificationSettings(change);
    },
    onMutate: () => client.getQueryCache().find({ queryKey, exact: true }),
    onSuccess: async (saved, _change, query) => {
      if (
        !isIdentityScopeCurrent(client, scope) ||
        !query ||
        client.getQueryCache().find({ queryKey, exact: true }) !== query
      )
        return;
      await client.cancelQueries({ predicate: (candidate) => candidate === query });
      if (!isIdentityScopeCurrent(client, scope) || client.getQueryCache().find({ queryKey, exact: true }) !== query)
        return;
      client.setQueryData(queryKey, saved);
    },
  });
}

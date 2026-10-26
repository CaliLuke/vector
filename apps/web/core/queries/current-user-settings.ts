/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file in the project root for details.
 */

import { queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { IUserSettings } from "@plane/types";
import userService from "@/services/user.service";
import { captureIdentityScope, identityKeys, isIdentityScopeVerified } from "./identity";

export function currentUserSettingsOptions(userId: string | undefined, bustCache = false) {
  return queryOptions({
    queryKey: identityKeys.settings(userId),
    staleTime: bustCache ? 0 : 5 * 60_000,
    refetchOnMount: false,
    queryFn: userId
      ? async ({ client, signal }) => {
          const scope = captureIdentityScope(client, userId);
          if (!isIdentityScopeVerified(client, scope))
            throw new Error("The account session changed before settings could load.");
          const settings = await userService.currentUserSettings(bustCache, signal);
          if (!isIdentityScopeVerified(client, scope))
            throw new Error("The account session changed while settings loaded.");
          if (settings.id !== userId) throw new Error("The server returned settings for a different account.");
          return settings;
        }
      : skipToken,
  });
}

export function refreshCurrentUserSettings(
  client: QueryClient,
  userId: string,
  bustCache = false
): Promise<IUserSettings> {
  return client.fetchQuery({ ...currentUserSettingsOptions(userId, bustCache), staleTime: 0 });
}

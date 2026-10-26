/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions, queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { TUserProfile, IUserTheme, TOnboardingSteps } from "@plane/types";
import userService from "@/services/user.service";
import { currentUserSettingsOptions } from "./current-user-settings";
import { captureIdentityScope, identityKeys, isIdentityScopeCurrent, isIdentityScopeVerified } from "./identity";

export type TUserProfileChange = Partial<TUserProfile>;

export type TFinishUserOnboardingResult = {
  profile: TUserProfile;
  refreshError?: unknown;
};

export function userProfileOptions(userId: string | undefined) {
  return queryOptions({
    queryKey: identityKeys.profile(userId),
    queryFn: userId
      ? async ({ client, signal }) => {
          const scope = captureIdentityScope(client, userId);
          if (!isIdentityScopeVerified(client, scope))
            throw new Error("The account session changed before the profile could load.");
          const profile = await userService.getCurrentUserProfile(signal);
          if (!isIdentityScopeVerified(client, scope))
            throw new Error("The account session changed while the profile loaded.");
          if (profile.user !== userId) throw new Error("The server returned a profile for a different account.");
          return profile;
        }
      : skipToken,
    staleTime: 5 * 60_000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    retry: false,
  });
}

function capturedProfileQuery(client: QueryClient, userId: string | undefined) {
  return client.getQueryCache().find({ queryKey: identityKeys.profile(userId), exact: true });
}

async function updateProfileCache(
  client: QueryClient,
  userId: string | undefined,
  scope: ReturnType<typeof captureIdentityScope>,
  capturedQuery: ReturnType<typeof capturedProfileQuery>,
  profile: TUserProfile
) {
  if (!userId || !isIdentityScopeCurrent(client, scope)) return;
  const key = identityKeys.profile(userId);
  if (capturedQuery) {
    await client.cancelQueries({ predicate: (query) => query === capturedQuery });
    if (!isIdentityScopeCurrent(client, scope)) return;
    if (capturedProfileQuery(client, userId) !== capturedQuery) return;
  } else if (capturedProfileQuery(client, userId)) {
    // A profile query began while this mutation was pending. Its canonical
    // response owns that newer Query instance, so leave it untouched.
    return;
  }
  client.setQueryData(key, profile);
}

export function updateUserProfileOptions(client: QueryClient, userId: string | undefined) {
  const scope = captureIdentityScope(client, userId);
  const query = capturedProfileQuery(client, userId);
  return mutationOptions({
    mutationKey: ["identity", userId, "profile", "update"],
    scope: { id: `identity/${userId}/profile` },
    retry: false,
    mutationFn: async (change: TUserProfileChange) => {
      if (!userId || !isIdentityScopeVerified(client, scope))
        throw new Error("The account session has changed or is unverified.");
      const profile = await userService.updateCurrentUserProfile(change);
      if (profile.user !== userId) throw new Error("The server returned a profile for a different account.");
      return profile;
    },
    onSuccess: (profile) => updateProfileCache(client, userId, scope, query, profile),
  });
}

export function updateUserThemeOptions(client: QueryClient, userId: string | undefined) {
  const scope = captureIdentityScope(client, userId);
  const query = capturedProfileQuery(client, userId);
  return mutationOptions({
    mutationKey: ["identity", userId, "profile", "theme"],
    scope: { id: `identity/${userId}/profile` },
    retry: false,
    mutationFn: async (change: Partial<IUserTheme>) => {
      if (!userId || !isIdentityScopeVerified(client, scope))
        throw new Error("The account session has changed or is unverified.");
      const current = client.getQueryData<TUserProfile>(identityKeys.profile(userId));
      if (!current) throw new Error("The profile is still loading. Try again in a moment.");
      const profile = await userService.updateCurrentUserProfile({
        theme: { ...current.theme, ...change },
      });
      if (profile.user !== userId) throw new Error("The server returned a profile for a different account.");
      return profile;
    },
    onSuccess: (profile) => updateProfileCache(client, userId, scope, query, profile),
  });
}

export function updateUserTourCompletedOptions(client: QueryClient, userId: string | undefined) {
  const scope = captureIdentityScope(client, userId);
  const query = capturedProfileQuery(client, userId);
  return mutationOptions({
    mutationKey: ["identity", userId, "profile", "tour-completed"],
    scope: { id: `identity/${userId}/profile` },
    retry: false,
    mutationFn: async () => {
      if (!userId || !isIdentityScopeVerified(client, scope))
        throw new Error("The account session has changed or is unverified.");
      return userService.updateUserTourCompleted();
    },
    onSuccess: async () => {
      if (!isIdentityScopeCurrent(client, scope)) return;
      if (!query || capturedProfileQuery(client, userId) !== query) return;
      await client.invalidateQueries({ predicate: (current) => current === query });
    },
  });
}

export function finishUserOnboardingOptions(client: QueryClient, userId: string | undefined) {
  const scope = captureIdentityScope(client, userId);
  const query = capturedProfileQuery(client, userId);
  return mutationOptions({
    mutationKey: ["identity", userId, "profile", "finish-onboarding"],
    scope: { id: `identity/${userId}/profile` },
    retry: false,
    mutationFn: async (lastWorkspaceId: string | undefined): Promise<TFinishUserOnboardingResult> => {
      if (!userId || !isIdentityScopeVerified(client, scope))
        throw new Error("The account session has changed or is unverified.");
      const onboardingStep: TOnboardingSteps = {
        profile_complete: true,
        workspace_join: true,
        workspace_create: true,
        workspace_invite: true,
      };
      const profile = await userService.updateCurrentUserProfile({
        onboarding_step: onboardingStep,
        last_workspace_id: lastWorkspaceId,
      });
      if (profile.user !== userId) throw new Error("The server returned a profile for a different account.");
      if (!isIdentityScopeCurrent(client, scope))
        throw new Error("The account session changed while saving onboarding.");

      await userService.updateUserOnBoard();
      const completedProfile = {
        ...profile,
        onboarding_step: onboardingStep,
        last_workspace_id: lastWorkspaceId,
        is_onboarded: true,
      };
      let canonicalProfile = completedProfile;
      let refreshError: unknown;

      // These reads are follow-up synchronization after both durable writes. A
      // failed refresh must not turn completed onboarding into a failed save.
      if (isIdentityScopeCurrent(client, scope)) {
        try {
          canonicalProfile = await client.fetchQuery({ ...userProfileOptions(userId), staleTime: 0 });
        } catch (error) {
          refreshError = error;
        }
      }
      if (isIdentityScopeCurrent(client, scope)) {
        try {
          await client.fetchQuery(currentUserSettingsOptions(userId, true));
        } catch (error) {
          refreshError ??= error;
        }
      }
      return { profile: canonicalProfile, refreshError };
    },
    onSuccess: async (result) => {
      if (!isIdentityScopeCurrent(client, scope)) return;
      await updateProfileCache(client, userId, scope, query, result.profile);
      if (result.refreshError) await client.invalidateQueries({ queryKey: identityKeys.settings(userId), exact: true });
    },
  });
}

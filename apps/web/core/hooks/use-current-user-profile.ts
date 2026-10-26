/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCurrentUser } from "@/hooks/use-current-user";
import { EStartOfTheWeek } from "@plane/types";
import {
  finishUserOnboardingOptions,
  updateUserProfileOptions,
  updateUserThemeOptions,
  updateUserTourCompletedOptions,
  userProfileOptions,
} from "@/queries/user-profile";

export function useCurrentUserProfile() {
  const { data: user } = useCurrentUser();
  return useQuery(userProfileOptions(user?.id));
}

export function useCurrentUserStartOfWeek() {
  const { data: user } = useCurrentUser();
  const { data: startOfWeek } = useQuery({
    ...userProfileOptions(user?.id),
    select: (profile) => profile?.start_of_the_week,
  });
  return startOfWeek ?? EStartOfTheWeek.SUNDAY;
}

export function useUpdateUserProfile() {
  const { data: user } = useCurrentUser();
  const client = useQueryClient();
  return useMutation(updateUserProfileOptions(client, user?.id));
}

export function useUpdateUserTheme() {
  const { data: user } = useCurrentUser();
  const client = useQueryClient();
  return useMutation(updateUserThemeOptions(client, user?.id));
}

export function useUpdateUserTourCompleted() {
  const { data: user } = useCurrentUser();
  const client = useQueryClient();
  return useMutation(updateUserTourCompletedOptions(client, user?.id));
}

export function useFinishUserOnboarding() {
  const { data: user } = useCurrentUser();
  const client = useQueryClient();
  return useMutation(finishUserOnboardingOptions(client, user?.id));
}

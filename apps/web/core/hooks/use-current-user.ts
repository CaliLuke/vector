/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { currentUserOptions, updateCurrentUserOptions } from "@/queries/identity";
import { endSessionOptions, passwordMutationOptions, resetIdentitySession } from "@/queries/identity-session";

export function useCurrentUser() {
  return useQuery(currentUserOptions());
}

export function useCurrentUserId() {
  return useQuery({ ...currentUserOptions(), select: (user) => user?.id }).data;
}

export function useUpdateCurrentUser() {
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  return useMutation(updateCurrentUserOptions(client, user?.id));
}

export function useSignOut() {
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  return useMutation(endSessionOptions(client, user?.id, () => resetIdentitySession(client), "sign-out"));
}

export function useDeactivateAccount() {
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  return useMutation(endSessionOptions(client, user?.id, () => resetIdentitySession(client), "deactivate"));
}

export function usePasswordMutation() {
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  return useMutation(passwordMutationOptions(client, user?.id));
}

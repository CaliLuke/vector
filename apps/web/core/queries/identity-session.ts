/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mutationOptions } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { authService, signOutCurrentSession } from "@/services/auth.service";
import userService from "@/services/user.service";
import {
  advanceIdentitySession,
  captureIdentityScope,
  currentUserOptions,
  isIdentityScopeCurrent,
  isIdentityScopeVerified,
} from "./identity";

export function resetIdentitySession(client: QueryClient) {
  advanceIdentitySession(client);
  client.clear();
  localStorage.setItem("theme", "system");
}

export function endSessionOptions(
  client: QueryClient,
  userId: string | undefined,
  resetSession: () => void,
  action: "sign-out" | "deactivate"
) {
  const scope = captureIdentityScope(client, userId);
  return mutationOptions({
    mutationKey: ["identity", action],
    scope: { id: "identity/session" },
    retry: false,
    gcTime: 0,
    mutationFn: async () => {
      if (action === "sign-out") {
        if (!isIdentityScopeCurrent(client, scope)) throw new Error("The account session changed.");
        await signOutCurrentSession();
      } else {
        if (!isIdentityScopeVerified(client, scope)) throw new Error("The account session changed or is unverified.");
        await userService.deactivateAccount();
      }
      if (isIdentityScopeCurrent(client, scope)) resetSession();
    },
  });
}

export function passwordMutationOptions(client: QueryClient, userId: string | undefined) {
  const scope = captureIdentityScope(client, userId);
  return mutationOptions({
    mutationKey: ["identity", "password"],
    scope: { id: `identity/${userId}` },
    retry: false,
    gcTime: 0,
    mutationFn: async (
      request:
        | { kind: "set"; csrfToken: string; password: string }
        | { kind: "change"; csrfToken: string; old_password?: string; new_password: string }
    ) => {
      if (!isIdentityScopeVerified(client, scope)) throw new Error("The account session changed or is unverified.");
      if (request.kind === "set") await authService.setPassword(request.csrfToken, { password: request.password });
      else
        await userService.changePassword(request.csrfToken, {
          old_password: request.old_password,
          new_password: request.new_password,
        });
    },
    onSuccess: async () => {
      if (!isIdentityScopeCurrent(client, scope)) return;
      await client.cancelQueries({ queryKey: currentUserOptions().queryKey, exact: true });
      if (isIdentityScopeCurrent(client, scope))
        client.setQueryData(currentUserOptions().queryKey, (user) => user && { ...user, is_password_autoset: false });
    },
  });
}

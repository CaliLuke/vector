/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { API_BASE_URL } from "@plane/constants";
import { AuthService } from "@plane/services";
import { installUnauthorizedRedirectInterceptor } from "./api.service";
import userService from "./user.service";

export const authService = new AuthService(API_BASE_URL, installUnauthorizedRedirectInterceptor);

export async function signOutCurrentSession() {
  const { csrf_token: csrfToken } = await authService.requestCSRFToken();
  if (!csrfToken) throw new Error("CSRF token not found");
  await authService.post("/auth/sign-out/", new URLSearchParams({ csrfmiddlewaretoken: csrfToken }));
  // This endpoint redirects on both success and failure. Confirm cookie revocation
  // before clearing private state rather than treating a submitted form as success.
  if (await userService.currentUser()) throw new Error("Unable to sign out. Please try again.");
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import { PasswordScreen } from "@/components/auth-screens/password";
import { EPageTypes } from "@/helpers/authentication.helper";

export const Route = createFileRoute("/accounts/reset-password/")({
  component: () => <PasswordScreen pageType={EPageTypes.NON_AUTHENTICATED} />,
});

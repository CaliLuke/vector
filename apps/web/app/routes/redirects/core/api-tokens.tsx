/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/$workspaceSlug/settings/api-tokens")({
  beforeLoad: () => {
    throw redirect({ to: "/settings/profile/$profileTabId/", params: { profileTabId: "api-tokens" }, replace: true });
  },
});

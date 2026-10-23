/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { createFileRoute, redirect } from "@tanstack/react-router";

export const Route = createFileRoute("/$workspaceSlug/projects/$projectId/settings/$")({
  beforeLoad: ({ params, location }) => {
    const { _splat: splat } = params;
    throw redirect({
      href: `/${params.workspaceSlug}/settings/projects/${params.projectId}/${splat || ""}${location.searchStr}${location.hash ? `#${location.hash}` : ""}`,
      replace: true,
    });
  },
});

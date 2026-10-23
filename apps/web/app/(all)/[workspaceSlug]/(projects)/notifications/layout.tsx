/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { createFileRoute, Outlet } from "@tanstack/react-router";
import { NotificationsSidebarRoot } from "@/components/workspace-notifications/sidebar";

function ProjectInboxIssuesLayout(): React.JSX.Element | null {
  return (
    <div className="relative flex h-full w-full items-center overflow-hidden">
      <NotificationsSidebarRoot />
      <div className="h-full w-full overflow-hidden overflow-y-auto">
        <Outlet />
      </div>
    </div>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/notifications")({
  component: ProjectInboxIssuesLayout,
});

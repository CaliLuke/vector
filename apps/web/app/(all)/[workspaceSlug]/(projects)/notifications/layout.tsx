/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { createFileRoute, Outlet } from "@tanstack/react-router";
import { NotificationsSidebarRoot } from "@/components/workspace-notifications/sidebar";

import { NotificationsProvider } from "@/components/workspace-notifications/provider";

function ProjectInboxIssuesLayout(): React.JSX.Element | null {
  const { workspaceSlug } = Route.useParams();
  return (
    <NotificationsProvider key={workspaceSlug} workspaceSlug={workspaceSlug}>
      <div className="relative flex h-full w-full items-center overflow-hidden">
        <NotificationsSidebarRoot />
        <div className="h-full w-full overflow-hidden overflow-y-auto">
          <Outlet />
        </div>
      </div>
    </NotificationsProvider>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/notifications")({
  component: ProjectInboxIssuesLayout,
});

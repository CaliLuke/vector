/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, Outlet, useLocation, useParams } from "@tanstack/react-router";

import { ProjectsAppPowerKProvider } from "@/components/power-k/projects-app-provider";
import { useCurrentUser } from "@/hooks/use-current-user";
import { WorkItemFilterUiProvider } from "@/providers/work-item-filter-ui";
// plane web components
import { ProjectAppSidebar } from "./_sidebar";
import { ExtendedProjectSidebar } from "./extended-project-sidebar";

function WorkspaceLayout(): React.JSX.Element | null {
  const params = useParams({ strict: false });
  const pathname = useLocation({ select: (location) => location.pathname });
  const { data: currentUser } = useCurrentUser();
  const scopeKey = [
    currentUser?.id,
    params.workspaceSlug,
    params.projectId,
    params.cycleId,
    params.moduleId,
    params.viewId,
    params.globalViewId,
    params.userId,
    pathname,
  ].join(":");
  return (
    <WorkItemFilterUiProvider scopeKey={scopeKey}>
      <>
        <ProjectsAppPowerKProvider />
        <div className="relative flex h-full w-full flex-col overflow-hidden rounded-lg border border-subtle">
          <div id="full-screen-portal" className="absolute inset-0 w-full" />
          <div className="relative flex size-full overflow-hidden">
            <ProjectAppSidebar />
            <ExtendedProjectSidebar />
            <main className="relative flex h-full w-full flex-col overflow-hidden bg-surface-1">
              <Outlet />
            </main>
          </div>
        </div>
      </>
    </WorkItemFilterUiProvider>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects")({
  component: WorkspaceLayout,
});

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { WorkItemTypesPreviewProvider } from "@/components/work-item-types/preview-provider";

import { createFileRoute, useChildMatches, Outlet } from "@tanstack/react-router";

// components
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { getWorkspaceActivePath, pathnameToAccessKey } from "@/components/settings/helper";
import { SettingsMobileNav } from "@/components/settings/mobile/nav";
// plane imports
import { WORKSPACE_SETTINGS_ACCESS } from "@plane/constants";
// components
import { WorkspaceSettingsSidebarRoot } from "@/components/settings/workspace/sidebar";
// hooks
import { useWorkspacePermissions } from "@/hooks/use-permissions";

function WorkspaceSettingLayout(): React.JSX.Element | null {
  // router
  const { workspaceSlug } = Route.useParams();
  // Workspace permissions
  const { workspaceRole, isPending, isError } = useWorkspacePermissions(workspaceSlug);
  // Use the rendered child route while this layout remains mounted during navigation.
  const pathname = useChildMatches({ select: (matches) => matches.at(-1)?.pathname }) ?? "";
  // derived values
  const { accessKey } = pathnameToAccessKey(pathname);
  const isAuthorized =
    !isError &&
    Boolean(
      pathname && workspaceSlug && workspaceRole && WORKSPACE_SETTINGS_ACCESS[accessKey]?.includes(workspaceRole)
    );

  if (isPending) return null;

  return (
    <>
      <SettingsMobileNav
        hamburgerContent={WorkspaceSettingsSidebarRoot}
        activePath={getWorkspaceActivePath(pathname) || ""}
      />
      <div className="inset-y-0 flex h-full w-full flex-row">
        {!isAuthorized ? (
          <NotAuthorizedView section="settings" className="h-auto" />
        ) : (
          <div className="relative flex size-full">
            <div className="hidden h-full md:block">
              <WorkspaceSettingsSidebarRoot />
            </div>
            <WorkItemTypesPreviewProvider key={workspaceSlug}>
              <Outlet />
            </WorkItemTypesPreviewProvider>
          </div>
        )}
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/_workspace")({
  component: WorkspaceSettingLayout,
});

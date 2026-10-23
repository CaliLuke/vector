/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useLocation, Outlet } from "@tanstack/react-router";
import { observer } from "mobx-react";

// components
import { NotAuthorizedView } from "@/components/auth-screens/not-authorized-view";
import { getWorkspaceActivePath, pathnameToAccessKey } from "@/components/settings/helper";
import { SettingsMobileNav } from "@/components/settings/mobile/nav";
// plane imports
import { WORKSPACE_SETTINGS_ACCESS } from "@plane/constants";
import type { EUserWorkspaceRoles } from "@plane/types";
// components
import { WorkspaceSettingsSidebarRoot } from "@/components/settings/workspace/sidebar";
// hooks
import { useUserPermissions } from "@/hooks/store/user";

const WorkspaceSettingLayout = observer(function WorkspaceSettingLayout(): React.JSX.Element | null {
  // router
  const { workspaceSlug } = Route.useParams();
  // store hooks
  const { workspaceUserInfo, getWorkspaceRoleByWorkspaceSlug } = useUserPermissions();
  // next hooks
  const pathname = useLocation({ select: (location) => location.pathname });
  // derived values
  const { accessKey } = pathnameToAccessKey(pathname);
  const userWorkspaceRole = getWorkspaceRoleByWorkspaceSlug(workspaceSlug);

  let isAuthorized: boolean | string = false;
  if (pathname && workspaceSlug && userWorkspaceRole) {
    isAuthorized = WORKSPACE_SETTINGS_ACCESS[accessKey]?.includes(userWorkspaceRole as EUserWorkspaceRoles);
  }

  return (
    <>
      <SettingsMobileNav
        hamburgerContent={WorkspaceSettingsSidebarRoot}
        activePath={getWorkspaceActivePath(pathname) || ""}
      />
      <div className="inset-y-0 flex h-full w-full flex-row">
        {workspaceUserInfo && !isAuthorized ? (
          <NotAuthorizedView section="settings" className="h-auto" />
        ) : (
          <div className="relative flex size-full">
            <div className="hidden h-full md:block">
              <WorkspaceSettingsSidebarRoot />
            </div>
            <Outlet />
          </div>
        )}
      </div>
    </>
  );
});

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/_workspace")({
  component: WorkspaceSettingLayout,
});

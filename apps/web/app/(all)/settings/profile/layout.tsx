/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { createFileRoute, Outlet } from "@tanstack/react-router";
import { ProjectsAppPowerKProvider } from "@/components/power-k/projects-app-provider";
// lib
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";

function ProfileSettingsLayout(): React.JSX.Element | null {
  return (
    <>
      <ProjectsAppPowerKProvider />
      <AuthenticationWrapper>
        <div className="relative flex size-full overflow-hidden bg-canvas p-2">
          <main className="relative flex size-full flex-col overflow-hidden rounded-lg border border-subtle bg-surface-1">
            <div className="size-full overflow-hidden">
              <Outlet />
            </div>
          </main>
        </div>
      </AuthenticationWrapper>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/settings/profile/$profileTabId")({
  component: ProfileSettingsLayout,
});

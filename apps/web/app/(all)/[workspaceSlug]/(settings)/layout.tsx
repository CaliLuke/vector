/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { createFileRoute, Outlet } from "@tanstack/react-router";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ProjectsAppPowerKProvider } from "@/components/power-k/projects-app-provider";

function SettingsLayout(): React.JSX.Element | null {
  return (
    <>
      <ProjectsAppPowerKProvider />
      <div className="relative flex size-full overflow-hidden rounded-lg border border-subtle">
        <main className="relative flex size-full flex-col overflow-hidden">
          {/* Content */}
          <ContentWrapper className="w-full bg-surface-1 md:flex">
            <div className="size-full overflow-hidden">
              <Outlet />
            </div>
          </ContentWrapper>
        </main>
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings")({
  component: SettingsLayout,
});

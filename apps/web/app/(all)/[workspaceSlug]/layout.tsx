/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, Outlet } from "@tanstack/react-router";
import { AuthenticationWrapper } from "@/lib/wrappers/authentication-wrapper";
import { WorkspaceContentWrapper } from "@/components/workspace/content-wrapper";
import { AppRailVisibilityProvider } from "@/lib/app-rail";
import { GlobalModals } from "@/components/common/modal/global";
import { WorkspaceIssuePreferencesProvider } from "@/providers/workspace-issue-preferences";
import { WorkspaceAuthWrapper } from "@/layouts/auth-layout/workspace-wrapper";

function WorkspaceLayout(): React.JSX.Element | null {
  const { workspaceSlug } = Route.useParams();

  return (
    <AuthenticationWrapper>
      <WorkspaceAuthWrapper>
        <WorkspaceIssuePreferencesProvider>
          <AppRailVisibilityProvider>
            <WorkspaceContentWrapper>
              <GlobalModals workspaceSlug={workspaceSlug} />
              <Outlet />
            </WorkspaceContentWrapper>
          </AppRailVisibilityProvider>
        </WorkspaceIssuePreferencesProvider>
      </WorkspaceAuthWrapper>
    </AuthenticationWrapper>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug")({
  component: WorkspaceLayout,
});

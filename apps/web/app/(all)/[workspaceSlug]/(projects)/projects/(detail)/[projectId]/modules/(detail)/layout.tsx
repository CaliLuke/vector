/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ParentIssuePreferencesProvider } from "@/providers/parent-issue-preferences";
// components
import { createFileRoute, Outlet } from "@tanstack/react-router";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { ModuleIssuesHeader } from "./header";
import { ModuleIssuesMobileHeader } from "./mobile-header";

function ProjectModuleIssuesLayout(): React.JSX.Element | null {
  const { workspaceSlug, projectId, moduleId } = Route.useParams();
  return (
    <ParentIssuePreferencesProvider
      domain="module"
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      parentId={moduleId}
    >
      <AppHeader header={<ModuleIssuesHeader />} mobileHeader={<ModuleIssuesMobileHeader />} />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </ParentIssuePreferencesProvider>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/modules/$moduleId")({
  component: ProjectModuleIssuesLayout,
});

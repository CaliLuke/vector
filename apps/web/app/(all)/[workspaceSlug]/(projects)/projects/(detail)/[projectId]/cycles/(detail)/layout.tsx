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
import { CycleIssuesHeader } from "./header";
import { CycleIssuesMobileHeader } from "./mobile-header";

function ProjectCycleIssuesLayout(): React.JSX.Element | null {
  const { workspaceSlug, projectId, cycleId } = Route.useParams();
  return (
    <ParentIssuePreferencesProvider
      domain="cycle"
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      parentId={cycleId}
    >
      <AppHeader header={<CycleIssuesHeader />} mobileHeader={<CycleIssuesMobileHeader />} />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </ParentIssuePreferencesProvider>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/cycles/$cycleId")({
  component: ProjectCycleIssuesLayout,
});

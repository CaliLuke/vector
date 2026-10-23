/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, Outlet } from "@tanstack/react-router";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
// local components
import { ProjectViewIssuesHeader } from "./[viewId]/header";

function ProjectViewIssuesLayout(): React.JSX.Element | null {
  return (
    <>
      <AppHeader header={<ProjectViewIssuesHeader />} />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/views/$viewId")({
  component: ProjectViewIssuesLayout,
});

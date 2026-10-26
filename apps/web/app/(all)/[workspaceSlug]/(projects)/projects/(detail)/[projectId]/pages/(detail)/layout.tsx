/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// component
import { createFileRoute, Outlet } from "@tanstack/react-router";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
import { PageRecordProvider } from "@/components/pages/page-record-context";
// local components
import { PageDetailsHeader } from "./header";

function ProjectPageDetailsLayout(): React.JSX.Element | null {
  const { workspaceSlug, projectId, pageId } = Route.useParams();
  return (
    <PageRecordProvider
      key={`${workspaceSlug}/${projectId}/${pageId}`}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      pageId={pageId}
    >
      <AppHeader header={<PageDetailsHeader />} />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </PageRecordProvider>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/pages/$pageId")({
  component: ProjectPageDetailsLayout,
});

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// component
import { createFileRoute, Outlet } from "@tanstack/react-router";
import useSWR from "swr";
import { AppHeader } from "@/components/core/app-header";
import { ContentWrapper } from "@/components/core/content-wrapper";
// plane web hooks
import { EPageStoreType, usePageStore } from "@/hooks/store";
// local components
import { PageDetailsHeader } from "./header";

function ProjectPageDetailsLayout(): React.JSX.Element | null {
  const { workspaceSlug, projectId } = Route.useParams();
  const { fetchPagesList } = usePageStore(EPageStoreType.PROJECT);
  // fetching pages list
  useSWR(`PROJECT_PAGES_${projectId}`, () => fetchPagesList(workspaceSlug, projectId));
  return (
    <>
      <AppHeader header={<PageDetailsHeader />} />
      <ContentWrapper>
        <Outlet />
      </ContentWrapper>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/pages/$pageId")({
  component: ProjectPageDetailsLayout,
});

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useLocation } from "@tanstack/react-router";

import { useTheme } from "next-themes";
// plane imports
import { EUserPermissions } from "@plane/constants";
import type { TPageNavigationTabs } from "@plane/types";
// assets
import darkPagesAsset from "@/app/assets/empty-state/disabled-feature/pages-dark.webp?url";
import lightPagesAsset from "@/app/assets/empty-state/disabled-feature/pages-light.webp?url";
// components
import { PageHead } from "@/components/core/page-title";
import { ProjectFeatureDisabledState } from "@/components/empty-state/project-feature-disabled";
import { PagesListRoot } from "@/components/pages/list/root";
import { PagesListView } from "@/components/pages/pages-list-view";
// hooks
import { useRouteProject, useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";

// plane web hooks

const getPageType = (pageType?: string | null): TPageNavigationTabs => {
  if (pageType === "private") return "private";
  if (pageType === "archived") return "archived";
  return "public";
};

function ProjectPagesPage(): React.JSX.Element | null {
  // router
  const searchParams = new URLSearchParams(useLocation({ select: (location) => location.searchStr }));
  const type = searchParams.get("type");
  const { workspaceSlug, projectId } = Route.useParams();
  // theme hook
  const { resolvedTheme } = useTheme();
  // store hooks
  const projectQuery = useRouteProject();
  const projectCatalogue = useWorkspaceProjectDetails(workspaceSlug);
  const currentProjectDetails = projectQuery.project;
  const { can } = useProjectPermissions(workspaceSlug, projectId);
  // derived values
  const project = projectCatalogue.projects?.find((row) => row.id === projectId);
  const pageTitle = project?.name ? `${project?.name} - Pages` : undefined;
  const canPerformEmptyStateActions = can([EUserPermissions.ADMIN]);
  const resolvedPath = resolvedTheme === "light" ? lightPagesAsset : darkPagesAsset;
  const pageType = getPageType(type);

  // No access to cycle
  if (currentProjectDetails?.page_view === false)
    return (
      <ProjectFeatureDisabledState
        feature="page"
        assetPath={resolvedPath}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        isManageDisabled={!canPerformEmptyStateActions}
      />
    );
  return (
    <>
      <PageHead title={pageTitle} />
      <PagesListView pageType={pageType} projectId={projectId} workspaceSlug={workspaceSlug}>
        <PagesListRoot pageType={pageType} workspaceSlug={workspaceSlug} />
      </PagesListView>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/pages/")({
  component: ProjectPagesPage,
});

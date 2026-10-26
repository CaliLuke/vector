/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports

import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import { EUserProjectRoles } from "@plane/types";
import { getViewName, orderViews, shouldFilterView } from "@plane/utils/project-views";
// components
import { ListLayout } from "@/components/core/list";
import { ViewListLoader } from "@/components/ui/loader/view-list-loader";
// hooks
import { useCommandPalette } from "@/hooks/use-command-palette";
import { useProjectViews } from "@/hooks/use-project-views";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useProjectViewListFilters } from "@/providers/project-view-list-filters";
// local imports
import { ProjectViewListItem } from "./view-list-item";

export function ProjectViewsList() {
  const { projectId } = useParams({ from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId" });
  // plane hooks
  const { t } = useTranslation();
  // store hooks
  const { toggleCreateViewModal } = useCommandPalette();
  const { workspaceSlug } = useParams({ strict: false });
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const { filters } = useProjectViewListFilters();
  const viewsQuery = useProjectViews(workspaceSlug, projectId);
  // derived values
  const projectViews = viewsQuery.views;
  const filteredProjectViews = projectViews
    ? orderViews(
        projectViews.filter(
          (view) =>
            getViewName(view.name).toLowerCase().includes(filters.searchQuery.toLowerCase()) &&
            shouldFilterView(view, filters.filters)
        ),
        filters.sortKey,
        filters.sortBy
      )
    : undefined;
  const canPerformEmptyStateActions = canProject([
    EUserProjectRoles.ADMIN,
    EUserProjectRoles.MEMBER,
    EUserProjectRoles.GUEST,
  ]);

  if (viewsQuery.isError || viewsQuery.isDenied) {
    return (
      <div role="alert" className="flex flex-col items-center gap-3 p-8 text-13 text-secondary">
        <p>{viewsQuery.isDenied ? "Project views are unavailable." : t("common.something_went_wrong")}</p>
        {!viewsQuery.isDenied && (
          <button type="button" className="text-accent-primary" onClick={() => void viewsQuery.retry()}>
            {t("common.retry")}
          </button>
        )}
      </div>
    );
  }

  if (viewsQuery.isPending || !projectViews || !filteredProjectViews) return <ViewListLoader />;

  if (filteredProjectViews.length === 0 && projectViews.length > 0) {
    return (
      <EmptyStateDetailed
        assetKey="search"
        title={t("common_empty_state.search.title")}
        description={t("common_empty_state.search.description")}
      />
    );
  }

  return (
    <>
      {filteredProjectViews.length > 0 ? (
        <div className="flex h-full w-full flex-col">
          <ListLayout>
            {filteredProjectViews.length > 0 ? (
              filteredProjectViews.map((view) => <ProjectViewListItem key={view.id} view={view} />)
            ) : (
              <p className="mt-10 text-center text-13 text-tertiary">No results found</p>
            )}
          </ListLayout>
        </div>
      ) : (
        <EmptyStateDetailed
          assetKey="view"
          title={t("project_empty_state.views.title")}
          description={t("project_empty_state.views.description")}
          actions={[
            {
              label: t("project_empty_state.views.cta_primary"),
              onClick: () => toggleCreateViewModal(true),
              disabled: !canPerformEmptyStateActions,
              variant: "primary",
            },
          ]}
        />
      )}
    </>
  );
}

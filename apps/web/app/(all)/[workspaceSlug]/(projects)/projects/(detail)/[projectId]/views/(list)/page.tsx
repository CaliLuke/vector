/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute, useRouter } from "@tanstack/react-router";
import { useCallback } from "react";
import { observer } from "mobx-react";
import { useTheme } from "next-themes";
// plane imports
import { EUserPermissionsLevel } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import type { EViewAccess, TViewFilterProps } from "@plane/types";
import { EUserProjectRoles } from "@plane/types";
import { Header, EHeaderVariant } from "@plane/ui";
import { calculateTotalFilters } from "@plane/utils";
// assets
import darkViewsAsset from "@/app/assets/empty-state/disabled-feature/views-dark.webp?url";
import lightViewsAsset from "@/app/assets/empty-state/disabled-feature/views-light.webp?url";
// components
import { PageHead } from "@/components/core/page-title";
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
import { ViewAppliedFiltersList } from "@/components/views/applied-filters";
import { ProjectViewsList } from "@/components/views/views-list";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useProjectView } from "@/hooks/store/use-project-view";
import { useUserPermissions } from "@/hooks/store/user";

function ProjectViewsPage(): React.JSX.Element | null {
  // router
  const router = useRouter();
  const { workspaceSlug, projectId } = Route.useParams();
  // theme hook
  const { resolvedTheme } = useTheme();
  // plane hooks
  const { t } = useTranslation();
  // store
  const { getProjectById, currentProjectDetails } = useProject();
  const { filters, updateFilters, clearAllFilters } = useProjectView();
  const { allowPermissions } = useUserPermissions();
  // derived values
  const project = getProjectById(projectId);
  const pageTitle = project?.name ? `${project?.name} - Views` : undefined;
  const canPerformEmptyStateActions = allowPermissions([EUserProjectRoles.ADMIN], EUserPermissionsLevel.PROJECT);
  const resolvedPath = resolvedTheme === "light" ? lightViewsAsset : darkViewsAsset;

  const handleRemoveFilter = useCallback(
    (key: keyof TViewFilterProps, value: string | EViewAccess | null) => {
      let newValues = filters.filters?.[key];

      if (key === "favorites") {
        newValues = !!value;
      }
      if (Array.isArray(newValues)) {
        if (!value) newValues = [];
        else newValues = newValues.filter((val) => val !== value) as string[];
      }

      updateFilters("filters", { [key]: newValues });
    },
    [filters.filters, updateFilters]
  );

  const isFiltersApplied = calculateTotalFilters(filters?.filters ?? {}) !== 0;

  // No access to
  if (currentProjectDetails?.issue_views_view === false)
    return (
      <div className="flex h-full w-full items-center justify-center">
        <DetailedEmptyState
          title={t("disabled_project.empty_state.view.title")}
          description={t("disabled_project.empty_state.view.description")}
          assetPath={resolvedPath}
          primaryButton={{
            text: t("disabled_project.empty_state.view.primary_button.text"),
            onClick: () => {
              void router.navigate({ href: `/${workspaceSlug}/settings/projects/${projectId}/features` });
            },
            disabled: !canPerformEmptyStateActions,
          }}
        />
      </div>
    );

  return (
    <>
      <PageHead title={pageTitle} />
      {isFiltersApplied && (
        <Header variant={EHeaderVariant.TERNARY}>
          <ViewAppliedFiltersList
            appliedFilters={filters.filters ?? {}}
            handleClearAllFilters={clearAllFilters}
            handleRemoveFilter={handleRemoveFilter}
            alwaysAllowEditing
          />
        </Header>
      )}
      <ProjectViewsList />
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/views/")({
  component: observer(ProjectViewsPage),
});

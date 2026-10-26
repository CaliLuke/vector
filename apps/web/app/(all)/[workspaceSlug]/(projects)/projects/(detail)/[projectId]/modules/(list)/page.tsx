/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import { useCallback } from "react";

import { useTheme } from "next-themes";
// plane imports

import type { TModuleFilters } from "@plane/types";
import { EUserProjectRoles } from "@plane/types";
import { calculateTotalFilters } from "@plane/utils/filter";
// assets
import darkModulesAsset from "@/app/assets/empty-state/disabled-feature/modules-dark.webp?url";
import lightModulesAsset from "@/app/assets/empty-state/disabled-feature/modules-light.webp?url";
// components
import { PageHead } from "@/components/core/page-title";
import { ProjectFeatureDisabledState } from "@/components/empty-state/project-feature-disabled";
import { ModuleAppliedFiltersList, ModulesListView } from "@/components/modules";
// hooks
import { useModuleCataloguePreferences } from "@/providers/cycle-module-preferences";
import { useRouteProject } from "@/hooks/use-projects";
import { useProjectPermissions } from "@/hooks/use-permissions";

function ProjectModulesPage(): React.JSX.Element | null {
  // router
  const { workspaceSlug, projectId } = Route.useParams();
  // theme hook
  const { resolvedTheme } = useTheme();
  // store
  const { project: currentProjectDetails } = useRouteProject();
  const preferences = useModuleCataloguePreferences(workspaceSlug, projectId);
  const { filters: currentProjectFilters, display: currentProjectDisplayFilters } = preferences;
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  // derived values
  const project = currentProjectDetails;
  const pageTitle = project?.name ? `${project?.name} - Modules` : undefined;
  const canPerformEmptyStateActions = canProject([EUserProjectRoles.ADMIN]);
  const resolvedPath = resolvedTheme === "light" ? lightModulesAsset : darkModulesAsset;

  const handleRemoveFilter = useCallback(
    (key: keyof TModuleFilters, value: string | null) => {
      let newValues = currentProjectFilters[key] ?? [];

      if (!value) newValues = [];
      else newValues = newValues.filter((val) => val !== value);

      preferences.update({ filters: { ...preferences.filters, [key]: newValues } });
    },
    [currentProjectFilters, preferences]
  );

  // No access to
  if (currentProjectDetails?.module_view === false)
    return (
      <ProjectFeatureDisabledState
        feature="module"
        assetPath={resolvedPath}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        isManageDisabled={!canPerformEmptyStateActions}
      />
    );

  return (
    <>
      <PageHead title={pageTitle} />
      <div className="flex h-full w-full flex-col">
        {(calculateTotalFilters(currentProjectFilters) !== 0 || currentProjectDisplayFilters?.favorites) && (
          <ModuleAppliedFiltersList
            appliedFilters={currentProjectFilters}
            isFavoriteFilterApplied={currentProjectDisplayFilters?.favorites ?? false}
            handleClearAllFilters={() =>
              preferences.update({ filters: {}, display: { ...preferences.display, favorites: false } })
            }
            handleRemoveFilter={handleRemoveFilter}
            handleDisplayFiltersUpdate={(val) => preferences.update({ display: { ...preferences.display, ...val } })}
            alwaysAllowEditing
          />
        )}
        <ModulesListView />
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/modules/")({
  component: ProjectModulesPage,
});

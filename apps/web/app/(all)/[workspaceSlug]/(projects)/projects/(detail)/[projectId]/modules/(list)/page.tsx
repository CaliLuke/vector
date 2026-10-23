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
import type { TModuleFilters } from "@plane/types";
import { EUserProjectRoles } from "@plane/types";
import { calculateTotalFilters } from "@plane/utils";
// assets
import darkModulesAsset from "@/app/assets/empty-state/disabled-feature/modules-dark.webp?url";
import lightModulesAsset from "@/app/assets/empty-state/disabled-feature/modules-light.webp?url";
// components
import { PageHead } from "@/components/core/page-title";
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
import { ModuleAppliedFiltersList, ModulesListView } from "@/components/modules";
// hooks
import { useModuleFilter } from "@/hooks/store/use-module-filter";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";

function ProjectModulesPage(): React.JSX.Element | null {
  // router
  const router = useRouter();
  const { workspaceSlug, projectId } = Route.useParams();
  // theme hook
  const { resolvedTheme } = useTheme();
  // plane hooks
  const { t } = useTranslation();
  // store
  const { getProjectById, currentProjectDetails } = useProject();
  const {
    currentProjectFilters = {},
    currentProjectDisplayFilters,
    clearAllFilters,
    updateFilters,
    updateDisplayFilters,
  } = useModuleFilter();
  const { allowPermissions } = useUserPermissions();
  // derived values
  const project = getProjectById(projectId);
  const pageTitle = project?.name ? `${project?.name} - Modules` : undefined;
  const canPerformEmptyStateActions = allowPermissions([EUserProjectRoles.ADMIN], EUserPermissionsLevel.PROJECT);
  const resolvedPath = resolvedTheme === "light" ? lightModulesAsset : darkModulesAsset;

  const handleRemoveFilter = useCallback(
    (key: keyof TModuleFilters, value: string | null) => {
      let newValues = currentProjectFilters[key] ?? [];

      if (!value) newValues = [];
      else newValues = newValues.filter((val) => val !== value);

      updateFilters(projectId, { [key]: newValues });
    },
    [currentProjectFilters, projectId, updateFilters]
  );

  // No access to
  if (currentProjectDetails?.module_view === false)
    return (
      <div className="flex h-full w-full items-center justify-center">
        <DetailedEmptyState
          title={t("disabled_project.empty_state.module.title")}
          description={t("disabled_project.empty_state.module.description")}
          assetPath={resolvedPath}
          primaryButton={{
            text: t("disabled_project.empty_state.module.primary_button.text"),
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
      <div className="flex h-full w-full flex-col">
        {(calculateTotalFilters(currentProjectFilters) !== 0 || currentProjectDisplayFilters?.favorites) && (
          <ModuleAppliedFiltersList
            appliedFilters={currentProjectFilters}
            isFavoriteFilterApplied={currentProjectDisplayFilters?.favorites ?? false}
            handleClearAllFilters={() => clearAllFilters(projectId)}
            handleRemoveFilter={handleRemoveFilter}
            handleDisplayFiltersUpdate={(val) => updateDisplayFilters(projectId, val)}
            alwaysAllowEditing
          />
        )}
        <ModulesListView />
      </div>
    </>
  );
}

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/$projectId/modules/")({
  component: observer(ProjectModulesPage),
});

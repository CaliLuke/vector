/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React, { useCallback } from "react";

// plane imports
import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import type { TModuleFilters } from "@plane/types";
// components
import { calculateTotalFilters } from "@plane/utils/filter";
import { ArchivedModulesView, ModuleAppliedFiltersList } from "@/components/modules";
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// helpers
// hooks
import { useProjectModules } from "@/hooks/use-modules";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { useModuleCataloguePreferences } from "@/providers/cycle-module-preferences";

export const ArchivedModuleLayoutRoot = function ArchivedModuleLayoutRoot() {
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // plane hooks
  const { t } = useTranslation();
  // hooks
  const catalogue = useProjectModules(workspaceSlug, projectId, true);
  const preferences = useModuleCataloguePreferences(workspaceSlug, projectId);
  const { archivedFilters: currentProjectArchivedFilters, update } = preferences;
  // derived values
  const totalArchivedModules = catalogue.modules?.length ?? 0;

  const handleRemoveFilter = useCallback(
    (key: keyof TModuleFilters, value: string | null) => {
      if (!projectId) return;
      let newValues = currentProjectArchivedFilters?.[key] ?? [];

      if (!value) newValues = [];
      else newValues = newValues.filter((val) => val !== value);

      update({ archivedFilters: { ...currentProjectArchivedFilters, [key]: newValues } });
    },
    [currentProjectArchivedFilters, projectId, update]
  );

  if (!workspaceSlug || !projectId) return <></>;

  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="modules" retry={catalogue.retry} />;
  if (catalogue.isPending) {
    return <CycleModuleListLayoutLoader />;
  }

  return (
    <>
      {calculateTotalFilters(currentProjectArchivedFilters ?? {}) !== 0 && (
        <div className="border-b border-subtle px-5 py-3">
          <ModuleAppliedFiltersList
            appliedFilters={currentProjectArchivedFilters ?? {}}
            handleClearAllFilters={() =>
              update({ archivedFilters: {}, display: { ...preferences.display, favorites: false } })
            }
            handleRemoveFilter={handleRemoveFilter}
            alwaysAllowEditing
            isArchived
          />
        </div>
      )}
      {totalArchivedModules === 0 ? (
        <div className="h-full place-items-center">
          <EmptyStateDetailed
            assetKey="archived-module"
            title={t("workspace_empty_state.archive_modules.title")}
            description={t("workspace_empty_state.archive_modules.description")}
          />
        </div>
      ) : (
        <div className="relative h-full w-full overflow-auto">
          <ArchivedModulesView workspaceSlug={workspaceSlug.toString()} projectId={projectId.toString()} />
        </div>
      )}
    </>
  );
};

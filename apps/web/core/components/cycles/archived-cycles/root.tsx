/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

// plane imports
import { useTranslation } from "@plane/i18n";
import { EmptyStateDetailed } from "@plane/propel/empty-state";
import type { TCycleFilters } from "@plane/types";
import { calculateTotalFilters } from "@plane/utils/filter";
// components
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { useCycleCataloguePreferences } from "@/providers/cycle-module-preferences";
// local imports
import { CycleAppliedFiltersList } from "../applied-filters";
import { ArchivedCyclesView } from "./view";

export const ArchivedCycleLayoutRoot = function ArchivedCycleLayoutRoot() {
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // plane hooks
  const { t } = useTranslation();
  // hooks
  const catalogue = useProjectCycles(workspaceSlug, projectId, true);
  // cycle filters hook
  const preferences = useCycleCataloguePreferences(workspaceSlug, projectId);
  const { archivedFilters: currentProjectArchivedFilters } = preferences;
  // derived values
  const totalArchivedCycles = catalogue.cycles?.length ?? 0;

  const handleRemoveFilter = (key: keyof TCycleFilters, value: string | null) => {
    if (!projectId) return;
    let newValues = currentProjectArchivedFilters?.[key] ?? [];

    if (!value) newValues = [];
    else newValues = newValues.filter((val) => val !== value);

    preferences.update({ archivedFilters: { ...currentProjectArchivedFilters, [key]: newValues } });
  };

  if (!workspaceSlug || !projectId) return <></>;

  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="cycles" retry={catalogue.retry} />;
  if (catalogue.isPending) {
    return <CycleModuleListLayoutLoader />;
  }

  return (
    <>
      {calculateTotalFilters(currentProjectArchivedFilters ?? {}) !== 0 && (
        <div className="border-b border-subtle px-5 py-3">
          <CycleAppliedFiltersList
            appliedFilters={currentProjectArchivedFilters ?? {}}
            handleClearAllFilters={() => preferences.update({ archivedFilters: {} })}
            handleRemoveFilter={handleRemoveFilter}
          />
        </div>
      )}
      {totalArchivedCycles === 0 ? (
        <div className="h-full place-items-center">
          <EmptyStateDetailed
            assetKey="archived-cycle"
            title={t("workspace_empty_state.archive_cycles.title")}
            description={t("workspace_empty_state.archive_cycles.description")}
          />
        </div>
      ) : (
        <div className="relative h-full w-full overflow-auto">
          <ArchivedCyclesView workspaceSlug={workspaceSlug} projectId={projectId} />
        </div>
      )}
    </>
  );
};

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// assets
import AllFiltersImage from "@/app/assets/empty-state/cycle/all-filters.svg?url";
import NameFilterImage from "@/app/assets/empty-state/cycle/name-filter.svg?url";
// components
import { CyclesList } from "@/components/cycles/list";
// ui
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { visibleArchivedCycles } from "@/queries/cycle-module-selectors";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { useCycleCataloguePreferences } from "@/providers/cycle-module-preferences";

export interface IArchivedCyclesView {
  workspaceSlug: string;
  projectId: string;
}

export const ArchivedCyclesView = function ArchivedCyclesView(props: IArchivedCyclesView) {
  const { workspaceSlug, projectId } = props;
  // store hooks
  const catalogue = useProjectCycles(workspaceSlug, projectId, true);
  const preferences = useCycleCataloguePreferences(workspaceSlug, projectId);
  const { archivedSearch: archivedCyclesSearchQuery } = preferences;
  // derived values
  const cycles = visibleArchivedCycles(catalogue.cycles, preferences.archivedFilters, preferences.archivedSearch);

  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="archived cycles" retry={catalogue.retry} />;
  if (catalogue.isPending || !cycles) return <CycleModuleListLayoutLoader />;

  if (cycles.length === 0)
    return (
      <div className="grid h-full w-full place-items-center">
        <div className="text-center">
          <img
            src={archivedCyclesSearchQuery.trim() === "" ? AllFiltersImage : NameFilterImage}
            className="mx-auto h-36 w-36 sm:h-48 sm:w-48"
            alt="No matching cycles"
          />
          <h5 className="mt-7 mb-1 text-18 font-medium">No matching cycles</h5>
          <p className="text-14 text-placeholder">
            {archivedCyclesSearchQuery.trim() === ""
              ? "Remove the filters to see all cycles"
              : "Remove the search criteria to see all cycles"}
          </p>
        </div>
      </div>
    );

  return (
    <CyclesList completedCycles={[]} cycles={cycles} workspaceSlug={workspaceSlug} projectId={projectId} isArchived />
  );
};

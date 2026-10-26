/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { useTranslation } from "@plane/i18n";
// assets
import AllFiltersImage from "@/app/assets/empty-state/cycle/all-filters.svg?url";
import NameFilterImage from "@/app/assets/empty-state/cycle/name-filter.svg?url";
// components
import { CyclesList } from "@/components/cycles/list";
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// hooks
import { useProjectCycles } from "@/hooks/use-cycles";
import { visibleCycles, visibleCompletedCycles, activeCycle } from "@/queries/cycle-module-selectors";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { useCycleCataloguePreferences } from "@/providers/cycle-module-preferences";

export interface ICyclesView {
  workspaceSlug: string;
  projectId: string;
}

export const CyclesView = function CyclesView(props: ICyclesView) {
  const { workspaceSlug, projectId } = props;
  // store hooks
  const catalogue = useProjectCycles(workspaceSlug, projectId);
  const preferences = useCycleCataloguePreferences(workspaceSlug, projectId);
  const { search: searchQuery } = preferences;
  const { t } = useTranslation();
  // derived values
  const cycles = visibleCycles(catalogue.cycles, preferences.filters, preferences.search);
  const completed = visibleCompletedCycles(catalogue.cycles, preferences.filters, preferences.search);
  const active = activeCycle(catalogue.cycles);
  const upcoming = cycles?.filter((cycle) => cycle.id !== active?.id);
  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="cycles" retry={catalogue.retry} />;
  if (catalogue.isPending || !cycles || !completed) return <CycleModuleListLayoutLoader />;

  if (cycles.length === 0 && completed.length === 0)
    return (
      <div className="grid h-full w-full place-items-center">
        <div className="text-center">
          <img
            src={searchQuery.trim() === "" ? AllFiltersImage : NameFilterImage}
            className="mx-auto h-36 w-36 object-contain sm:h-48 sm:w-48"
            alt="No matching cycles"
          />
          <h5 className="mt-7 mb-1 text-18 font-medium">{t("project_cycles.no_matching_cycles")}</h5>
          <p className="text-14 text-placeholder">
            {searchQuery.trim() === ""
              ? t("project_cycles.remove_filters_to_see_all_cycles")
              : t("project_cycles.remove_search_criteria_to_see_all_cycles")}
          </p>
        </div>
      </div>
    );

  return (
    <CyclesList
      completedCycles={completed}
      upcomingCycles={upcoming}
      cycles={cycles}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
    />
  );
};

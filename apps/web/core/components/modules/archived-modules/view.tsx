/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// assets
import AllFiltersImage from "@/app/assets/empty-state/module/all-filters.svg?url";
import NameFilterImage from "@/app/assets/empty-state/module/name-filter.svg?url";
// components
import { ModuleListItem, ModulePeekOverview } from "@/components/modules";
// ui
import { CycleModuleListLayoutLoader } from "@/components/ui/loader/cycle-module-list-loader";
// hooks
import { useProjectModules } from "@/hooks/use-modules";
import { useFavorites } from "@/hooks/use-favorites";
import { favoriteModuleIds, visibleModules } from "@/queries/cycle-module-selectors";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
import { useModuleCataloguePreferences } from "@/providers/cycle-module-preferences";

export interface IArchivedModulesView {
  workspaceSlug: string;
  projectId: string;
}

function ArchivedModulesEmptyState(props: { search: string }) {
  const hasSearch = props.search.trim() !== "";
  return (
    <div className="grid h-full w-full place-items-center">
      <div className="text-center">
        <img
          src={hasSearch ? NameFilterImage : AllFiltersImage}
          className="mx-auto h-36 w-36 sm:h-48 sm:w-48"
          alt="No matching modules"
        />
        <h5 className="mt-7 mb-1 text-18 font-medium">No matching modules</h5>
        <p className="text-14 text-placeholder">
          {hasSearch ? "Remove the search criteria to see all modules" : "Remove the filters to see all modules"}
        </p>
      </div>
    </div>
  );
}

export const ArchivedModulesView = function ArchivedModulesView(props: IArchivedModulesView) {
  const { workspaceSlug, projectId } = props;
  // store hooks
  const catalogue = useProjectModules(workspaceSlug, projectId, true);
  const favorites = useFavorites(workspaceSlug);
  const preferences = useModuleCataloguePreferences(workspaceSlug, projectId);
  const { archivedSearch: archivedModulesSearchQuery } = preferences;
  // derived values
  const favoriteIds = favoriteModuleIds(favorites.favorites);
  const modules = visibleModules(
    catalogue.modules,
    preferences.display,
    preferences.archivedFilters,
    preferences.archivedSearch,
    favoriteIds
  );

  if (preferences.display.favorites && favorites.isError)
    return <CatalogueReadFailure label="favorites" retry={favorites.retry} />;
  if (catalogue.isError || catalogue.isAccessDenied)
    return <CatalogueReadFailure label="archived modules" retry={catalogue.retry} />;
  if ((preferences.display.favorites && !favorites.favorites) || catalogue.isPending || !modules)
    return <CycleModuleListLayoutLoader />;

  if (modules.length === 0) return <ArchivedModulesEmptyState search={archivedModulesSearchQuery} />;

  return (
    <div className="h-full overflow-y-auto">
      <div className="flex h-full w-full justify-between">
        <div className="vertical-scrollbar flex scrollbar-lg h-full w-full flex-col overflow-y-auto">
          {modules.map((module) => (
            <ModuleListItem key={module.id} module={module} />
          ))}
        </div>
        <ModulePeekOverview projectId={projectId ?? ""} workspaceSlug={workspaceSlug ?? ""} isArchived />
      </div>
    </div>
  );
};

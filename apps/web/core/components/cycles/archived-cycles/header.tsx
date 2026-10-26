/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// icons
// plane helpers
// types
import type { TCycleFilters } from "@plane/types";
import { calculateTotalFilters } from "@plane/utils/filter";
// components
import { ArchiveCatalogueHeader } from "@/components/archives";
// hooks
import { useCycleCataloguePreferences } from "@/providers/cycle-module-preferences";
import { useArchivedFilterUpdate } from "../../cycles-modules/use-archived-filter-update";
// local imports
import { CycleFiltersSelection } from "../dropdowns";

export const ArchivedCyclesHeader = function ArchivedCyclesHeader() {
  // router
  const { projectId, workspaceSlug } = useParams({ strict: false });
  // hooks
  const preferences = useCycleCataloguePreferences(workspaceSlug, projectId);
  const {
    archivedFilters: currentProjectArchivedFilters,
    archivedSearch: archivedCyclesSearchQuery,
    update,
  } = preferences;
  const handleFilters = useArchivedFilterUpdate<keyof TCycleFilters>(projectId, currentProjectArchivedFilters, update);

  const isFiltersApplied = calculateTotalFilters(currentProjectArchivedFilters ?? {}) !== 0;

  return (
    <ArchiveCatalogueHeader
      searchQuery={archivedCyclesSearchQuery}
      onSearchChange={(archivedSearch) => update({ archivedSearch })}
      isFiltersApplied={isFiltersApplied}
      filters={
        <CycleFiltersSelection
          filters={currentProjectArchivedFilters ?? {}}
          handleFiltersUpdate={handleFilters}
          isArchived
        />
      }
    />
  );
};

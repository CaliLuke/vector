/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// icons
// plane helpers
// types
import type { TModuleFilters } from "@plane/types";
import { calculateTotalFilters } from "@plane/utils/filter";
// components
import { ArchiveCatalogueHeader } from "@/components/archives";
import { ModuleFiltersSelection, ModuleOrderByDropdown } from "@/components/modules";
// helpers
// hooks
import { useModuleCataloguePreferences } from "@/providers/cycle-module-preferences";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { workspaceRosterMemberIds } from "@/helpers/member-rosters";
import { useArchivedFilterUpdate } from "../../cycles-modules/use-archived-filter-update";

export const ArchivedModulesHeader = function ArchivedModulesHeader() {
  // router
  const { projectId, workspaceSlug } = useParams({ strict: false });
  // hooks
  const preferences = useModuleCataloguePreferences(workspaceSlug, projectId);
  const {
    archivedFilters: currentProjectArchivedFilters,
    display: currentProjectDisplayFilters,
    archivedSearch: archivedModulesSearchQuery,
    update,
  } = preferences;
  const { data: currentUser } = useCurrentUser();
  const { members } = useWorkspaceMemberRoster(workspaceSlug);
  const workspaceMemberIds = workspaceRosterMemberIds(members, currentUser?.id);
  const handleFilters = useArchivedFilterUpdate<keyof TModuleFilters>(projectId, currentProjectArchivedFilters, update);

  const isFiltersApplied = calculateTotalFilters(currentProjectArchivedFilters ?? {}) !== 0;

  return (
    <ArchiveCatalogueHeader
      searchQuery={archivedModulesSearchQuery}
      onSearchChange={(archivedSearch) => update({ archivedSearch })}
      isFiltersApplied={isFiltersApplied}
      extraControls={
        <ModuleOrderByDropdown
          value={currentProjectDisplayFilters?.order_by}
          onChange={(val) => {
            if (!projectId || val === currentProjectDisplayFilters?.order_by) return;
            update({
              display: {
                ...preferences.display,
                order_by: val,
              },
            });
          }}
        />
      }
      filters={
        <ModuleFiltersSelection
          displayFilters={currentProjectDisplayFilters ?? {}}
          filters={currentProjectArchivedFilters ?? {}}
          handleDisplayFiltersUpdate={(val) => {
            if (!projectId) return;
            preferences.update({ display: { ...preferences.display, ...val } });
          }}
          handleFiltersUpdate={handleFilters}
          memberIds={workspaceMemberIds}
          workspaceSlug={workspaceSlug}
          isArchived
        />
      }
    />
  );
};

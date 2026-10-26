/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { TModuleDisplayFilters, TModuleFilters } from "@plane/types";
// components
import { FilterOption } from "@/components/issues/issue-layouts/filters";
import { DateRangeFilters } from "@/components/core/filters/date-range-filters";
import { FilterSelectionPanel } from "@/components/core/filters/filter-selection-panel";
import { FilterLead } from "./lead";
import { FilterMembers } from "./members";
import { FilterStatus } from "./status";

type Props = {
  displayFilters: TModuleDisplayFilters;
  filters: TModuleFilters;
  handleDisplayFiltersUpdate: (updatedDisplayProperties: Partial<TModuleDisplayFilters>) => void;
  handleFiltersUpdate: (key: keyof TModuleFilters, value: string | string[]) => void;
  memberIds?: string[] | undefined;
  workspaceSlug: string | undefined;
  isArchived?: boolean;
};

export const ModuleFiltersSelection = function ModuleFiltersSelection(props: Props) {
  const {
    displayFilters,
    filters,
    handleDisplayFiltersUpdate,
    handleFiltersUpdate,
    memberIds,
    workspaceSlug,
    isArchived = false,
  } = props;

  return (
    <FilterSelectionPanel>
      {(searchQuery) => (
        <>
          {!isArchived && (
            <div className="py-2">
              <FilterOption
                isChecked={!!displayFilters.favorites}
                onClick={() =>
                  handleDisplayFiltersUpdate({
                    favorites: !displayFilters.favorites,
                  })
                }
                title="Favorites"
              />
            </div>
          )}

          {/* status */}
          {!isArchived && (
            <div className="py-2">
              <FilterStatus
                appliedFilters={filters.status ?? null}
                handleUpdate={(val) => handleFiltersUpdate("status", val)}
                searchQuery={searchQuery}
              />
            </div>
          )}

          {/* lead */}
          <div className="py-2">
            <FilterLead
              appliedFilters={filters.lead ?? null}
              handleUpdate={(val) => handleFiltersUpdate("lead", val)}
              searchQuery={searchQuery}
              memberIds={memberIds}
              workspaceSlug={workspaceSlug}
            />
          </div>

          {/* members */}
          <div className="py-2">
            <FilterMembers
              appliedFilters={filters.members ?? null}
              handleUpdate={(val) => handleFiltersUpdate("members", val)}
              searchQuery={searchQuery}
              memberIds={memberIds}
              workspaceSlug={workspaceSlug}
            />
          </div>

          <DateRangeFilters
            startDate={filters.start_date ?? null}
            dueDate={filters.target_date ?? null}
            handleStartDateUpdate={(value) => handleFiltersUpdate("start_date", value)}
            handleDueDateUpdate={(value) => handleFiltersUpdate("target_date", value)}
            searchQuery={searchQuery}
          />
        </>
      )}
    </FilterSelectionPanel>
  );
};

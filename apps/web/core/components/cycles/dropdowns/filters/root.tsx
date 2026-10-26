/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { TCycleFilters } from "@plane/types";
// local imports
import { DateRangeFilters } from "@/components/core/filters/date-range-filters";
import { FilterSelectionPanel } from "@/components/core/filters/filter-selection-panel";
import { FilterStatus } from "./status";

type Props = {
  filters: TCycleFilters;
  handleFiltersUpdate: (key: keyof TCycleFilters, value: string | string[]) => void;
  isArchived?: boolean;
};

export const CycleFiltersSelection = function CycleFiltersSelection(props: Props) {
  const { filters, handleFiltersUpdate, isArchived = false } = props;

  return (
    <FilterSelectionPanel>
      {(searchQuery) => (
        <>
          {!isArchived && (
            <div className="py-2">
              <FilterStatus
                appliedFilters={filters.status ?? null}
                handleUpdate={(value) => handleFiltersUpdate("status", value)}
                searchQuery={searchQuery}
              />
            </div>
          )}
          <DateRangeFilters
            startDate={filters.start_date ?? null}
            dueDate={filters.end_date ?? null}
            handleStartDateUpdate={(value) => handleFiltersUpdate("start_date", value)}
            handleDueDateUpdate={(value) => handleFiltersUpdate("end_date", value)}
            searchQuery={searchQuery}
          />
        </>
      )}
    </FilterSelectionPanel>
  );
};

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import concat from "lodash-es/concat.js";
import uniq from "lodash-es/uniq.js";
import { PAST_DURATION_FILTER_OPTIONS } from "@plane/constants";
import type { TInboxIssueFilterDateKeys } from "@plane/types";
// components
import { DateFilterModal } from "@/components/core/filters/date-filter-modal";
import { FilterHeader, FilterOption } from "@/components/issues/issue-layouts/filters";
// constants
// hooks
import { useIntakePreferences } from "@/providers/intake-preferences";

type Props = {
  filterKey: TInboxIssueFilterDateKeys;
  label?: string;
  searchQuery: string;
};

const isDate = (date: string) => {
  const datePattern = /^\d{4}-\d{2}-\d{2}$/;
  return datePattern.test(date);
};

export const FilterDate = function FilterDate(props: Props) {
  const { filterKey, label, searchQuery } = props;
  // hooks
  const { filters, setFilter } = useIntakePreferences();
  // state
  const [previewEnabled, setPreviewEnabled] = useState(true);
  const [isDateFilterModalOpen, setIsDateFilterModalOpen] = useState(false);
  // derived values
  const filterValue: string[] = filters?.[filterKey] || [];
  const appliedFiltersCount = filterValue?.length ?? 0;
  const filteredOptions = PAST_DURATION_FILTER_OPTIONS.filter((d) =>
    d.name.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const handleFilterValue = (value: string): string[] => (filterValue?.includes(value) ? [] : uniq(concat(value)));

  const isCustomDateSelected = () => {
    const isValidDateSelected = filterValue?.filter((f) => isDate(f.split(";")[0])) || [];
    return isValidDateSelected.length > 0 ? true : false;
  };

  const handleCustomDate = () => {
    if (isCustomDateSelected()) {
      const updateAppliedFilters = filterValue?.filter((f) => !isDate(f.split(";")[0])) || [];
      setFilter(filterKey, updateAppliedFilters);
    } else {
      setIsDateFilterModalOpen(true);
    }
  };

  return (
    <>
      {isDateFilterModalOpen && (
        <DateFilterModal
          handleClose={() => setIsDateFilterModalOpen(false)}
          isOpen={isDateFilterModalOpen}
          onSelect={(val) => setFilter(filterKey, val)}
          title="Created date"
        />
      )}
      <FilterHeader
        title={`${label || "Created date"}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
        isPreviewEnabled={previewEnabled}
        handleIsPreviewEnabled={() => setPreviewEnabled(!previewEnabled)}
      />
      {previewEnabled && (
        <div>
          {filteredOptions.length > 0 ? (
            <>
              {filteredOptions.map((option) => (
                <FilterOption
                  key={option.value}
                  isChecked={filterValue?.includes(option.value) ? true : false}
                  onClick={() => setFilter(filterKey, handleFilterValue(option.value))}
                  title={option.name}
                  multiple={false}
                />
              ))}
              <FilterOption
                isChecked={isCustomDateSelected()}
                onClick={handleCustomDate}
                title="Custom"
                multiple={false}
              />
            </>
          ) : (
            <p className="text-11 text-placeholder italic">No matches found</p>
          )}
        </div>
      )}
    </>
  );
};

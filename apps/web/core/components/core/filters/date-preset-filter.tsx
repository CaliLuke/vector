/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";

import { DATE_AFTER_FILTER_OPTIONS } from "@plane/constants";
import { isInDateFormat } from "@plane/utils/datetime";
import { DateFilterModal } from "@/components/core/filters/date-filter-modal";
import { FilterHeader } from "@/components/issues/issue-layouts/filters/header/helpers/filter-header";
import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";

type DatePresetFilterProps = {
  appliedFilters: string[] | null;
  handleUpdate: (value: string | string[]) => void;
  searchQuery: string;
  title: string;
  options?: ReadonlyArray<{ value: string; name: string }>;
  multiple?: boolean;
};

export const DatePresetFilter = function DatePresetFilter(props: DatePresetFilterProps) {
  const {
    appliedFilters,
    handleUpdate,
    searchQuery,
    title,
    options = DATE_AFTER_FILTER_OPTIONS,
    multiple = true,
  } = props;
  const [previewEnabled, setPreviewEnabled] = useState(true);
  const [isDateFilterModalOpen, setIsDateFilterModalOpen] = useState(false);

  const appliedFiltersCount = appliedFilters?.length ?? 0;
  const filteredOptions = options.filter((option) => option.name.toLowerCase().includes(searchQuery.toLowerCase()));

  const isCustomDateSelected = () => {
    const isValidDateSelected = appliedFilters?.filter((filter) => isInDateFormat(filter.split(";")[0])) || [];
    return isValidDateSelected.length > 0;
  };

  const handleCustomDate = () => {
    if (isCustomDateSelected()) {
      const updateAppliedFilters = appliedFilters?.filter((filter) => filter.includes("-")) || [];
      handleUpdate(updateAppliedFilters);
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
          onSelect={(value) => handleUpdate(value)}
          title={title}
        />
      )}
      <FilterHeader
        title={`${title}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
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
                  isChecked={Boolean(appliedFilters?.includes(option.value))}
                  onClick={() => handleUpdate(option.value)}
                  title={option.name}
                  multiple={multiple}
                />
              ))}
              <FilterOption
                isChecked={isCustomDateSelected()}
                onClick={handleCustomDate}
                title="Custom"
                multiple={multiple}
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

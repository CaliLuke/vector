/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// constants
import { ISSUE_DISPLAY_FILTERS_BY_LAYOUT } from "@plane/constants";
// components
import { FiltersDropdown } from "@/components/issues/filters/helpers/dropdown";
import { FilterSelection } from "@/components/issues/filters/selection";
// helpers
// hooks
import { usePublicIssueView } from "@/hooks/use-public-issue-view";
// types
import type { TIssueFilterKeys } from "@/types/issue";

type IssueFiltersDropdownProps = {
  anchor: string;
};

export const IssueFiltersDropdown = function IssueFiltersDropdown(props: IssueFiltersDropdownProps) {
  const { anchor } = props;
  const { layout: activeLayout, filters, setFilter } = usePublicIssueView(anchor);
  const handleFilters = (key: TIssueFilterKeys, value: string | string[]) => {
    if (Array.isArray(value)) return setFilter(key, value);
    const selected = filters[key];
    setFilter(key, selected.includes(value) ? selected.filter((item) => item !== value) : [...selected, value]);
  };

  return (
    <div className="relative">
      <FiltersDropdown title="Filters" placement="bottom-end">
        <FilterSelection
          filters={filters}
          handleFilters={handleFilters}
          layoutDisplayFiltersOptions={activeLayout ? ISSUE_DISPLAY_FILTERS_BY_LAYOUT?.[activeLayout]?.filters : []}
        />
      </FiltersDropdown>
    </div>
  );
};

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import type { ReactNode } from "react";
import { DATE_BEFORE_FILTER_OPTIONS } from "@plane/constants";
import { FilterSelectionSearch } from "@/components/common/filters/filter-selection-search";
import { DatePresetFilter } from "@/components/core/filters/date-preset-filter";
import { FilterOption } from "@/components/issues/issue-layouts/filters";
import { usePlatformOS } from "@/hooks/use-platform-os";

type Props = {
  filters: { favorites?: boolean; created_at?: string[] | null };
  onChange: (filters: { favorites?: boolean; created_at?: string[] | null }) => void;
  searchClassName?: string;
  searchInputClassName?: string;
  children: (searchQuery: string) => ReactNode;
};

export function FavoritesCreatedDateFilters(props: Props) {
  const { filters, onChange, searchClassName, searchInputClassName, children } = props;
  const [searchQuery, setSearchQuery] = useState("");
  const { isMobile } = usePlatformOS();
  const toggleCreatedAt = (value: string | string[]) => {
    const values = [...(filters.created_at ?? [])];
    for (const selected of Array.isArray(value) ? value : [value]) {
      const index = values.indexOf(selected);
      if (index === -1) values.push(selected);
      else values.splice(index, 1);
    }
    onChange({ created_at: values });
  };

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      <FilterSelectionSearch
        value={searchQuery}
        onChange={setSearchQuery}
        // oxlint-disable-next-line jsx_a11y/no-autofocus
        autoFocus={!isMobile}
        className={searchClassName}
        inputClassName={searchInputClassName}
      />
      <div className="vertical-scrollbar scrollbar-sm h-full w-full divide-y divide-subtle-1 overflow-y-auto px-2.5">
        <div className="py-2">
          <FilterOption
            isChecked={!!filters.favorites}
            onClick={() => onChange({ favorites: !filters.favorites })}
            title="Favorites"
          />
        </div>
        <div className="py-2">
          <DatePresetFilter
            title="Created date"
            options={DATE_BEFORE_FILTER_OPTIONS}
            appliedFilters={filters.created_at ?? null}
            handleUpdate={toggleCreatedAt}
            searchQuery={searchQuery}
          />
        </div>
        {children(searchQuery)}
      </div>
    </div>
  );
}

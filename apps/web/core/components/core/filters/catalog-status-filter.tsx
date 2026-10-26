/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";

import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";
import { FilterOptionSection } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option-section";

type CatalogStatusFilterOption = {
  value: string;
  title: string;
  icon?: ReactNode;
};

export type CatalogStatusFilterProps = {
  appliedFilters: readonly string[] | null;
  handleUpdate: (value: string) => void;
  searchQuery: string;
};

type Props = CatalogStatusFilterProps & {
  options: readonly CatalogStatusFilterOption[];
  title: string;
};

export function CatalogStatusFilter({ appliedFilters, handleUpdate, options, searchQuery, title }: Props) {
  const filteredOptions = options.filter((option) => option.value.includes(searchQuery.toLowerCase()));
  const appliedFiltersCount = appliedFilters?.length ?? 0;

  return (
    <FilterOptionSection
      title={`${title}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
      optionCount={filteredOptions.length}
      renderOptions={() =>
        filteredOptions.map((option) => (
          <FilterOption
            key={option.value}
            isChecked={appliedFilters?.includes(option.value) ?? false}
            onClick={() => handleUpdate(option.value)}
            icon={option.icon}
            title={option.title}
          />
        ))
      }
    />
  );
}

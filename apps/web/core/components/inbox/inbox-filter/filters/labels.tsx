/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IIssueLabel } from "@plane/types";
// components
import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";
import { FilterOptionSection } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option-section";
// hooks
import { useIntakePreferences } from "@/providers/intake-preferences";

function LabelIcons({ color }: { color: string }) {
  return <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: color }} />;
}

type Props = {
  labels: IIssueLabel[] | undefined;
  searchQuery: string;
};

export const FilterLabels = function FilterLabels(props: Props) {
  const { labels, searchQuery } = props;

  const { filters, setFilter } = useIntakePreferences();

  const filterValue = filters?.labels || [];

  const appliedFiltersCount = filterValue?.length ?? 0;

  const filteredOptions = labels?.filter((label) => label.name.toLowerCase().includes(searchQuery.toLowerCase()));

  const handleFilterValue = (value: string): string[] =>
    filterValue?.includes(value) ? filterValue.filter((v) => v !== value) : [...filterValue, value];

  return (
    <FilterOptionSection
      title={`Label${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
      optionCount={filteredOptions?.length ?? 0}
      state={filteredOptions === undefined ? { kind: "loading" } : undefined}
      renderOptions={(itemsToRender) =>
        filteredOptions
          ?.slice(0, itemsToRender)
          .map((label) => (
            <FilterOption
              key={label?.id}
              isChecked={filterValue.includes(label.id)}
              onClick={() => setFilter("labels", handleFilterValue(label.id))}
              icon={<LabelIcons color={label.color} />}
              title={label.name}
            />
          ))
      }
    />
  );
};

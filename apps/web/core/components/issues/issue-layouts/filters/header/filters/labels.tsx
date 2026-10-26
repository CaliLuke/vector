/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo } from "react";
import sortBy from "lodash-es/sortBy.js";

import type { IIssueLabel } from "@plane/types";
// components
import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";
import { FilterOptionSection } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option-section";

// ui
// types

function LabelIcons({ color }: { color: string }) {
  return <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: color }} />;
}

type Props = {
  appliedFilters: string[] | null;
  handleUpdate: (val: string) => void;
  labels: IIssueLabel[] | undefined;
  searchQuery: string;
};

export const FilterLabels = function FilterLabels(props: Props) {
  const { appliedFilters, handleUpdate, labels, searchQuery } = props;

  const appliedFiltersCount = appliedFilters?.length ?? 0;

  const sortedOptions = useMemo(() => {
    const filteredOptions = (labels || []).filter((label) =>
      label.name.toLowerCase().includes(searchQuery.toLowerCase())
    );

    return sortBy(filteredOptions, [
      (label) => !(appliedFilters ?? []).includes(label.id),
      (label) => label.name.toLowerCase(),
    ]);
  }, [appliedFilters, labels, searchQuery]);

  return (
    <FilterOptionSection
      title={`Label${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
      optionCount={sortedOptions.length}
      renderOptions={(itemsToRender) =>
        sortedOptions
          .slice(0, itemsToRender)
          .map((label) => (
            <FilterOption
              key={label?.id}
              isChecked={appliedFilters?.includes(label.id) ?? false}
              onClick={() => handleUpdate(label?.id)}
              icon={<LabelIcons color={label.color} />}
              title={label.name}
            />
          ))
      }
    />
  );
};

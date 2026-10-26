/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import { EIconSize } from "@plane/constants";
import { StateGroupIcon } from "@plane/propel/icons";
import type { IState } from "@plane/types";
// components
import { FilterOption, FilterOptionSection } from "@/components/issues/issue-layouts/filters";
// hooks
import { useIntakePreferences } from "@/providers/intake-preferences";

type Props = {
  states: IState[] | undefined;
  searchQuery: string;
};

export const FilterState = function FilterState(props: Props) {
  const { states, searchQuery } = props;

  const { filters, setFilter } = useIntakePreferences();

  const filterValue = filters?.state || [];

  const appliedFiltersCount = filterValue?.length ?? 0;

  const filteredOptions = useMemo(
    () => states?.filter((state) => state.name.toLowerCase().includes(searchQuery.toLowerCase())),
    [searchQuery, states]
  );

  const handleFilterValue = (value: string): string[] =>
    filterValue?.includes(value) ? filterValue.filter((v) => v !== value) : [...filterValue, value];

  return (
    <FilterOptionSection
      title={`State${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
      optionCount={filteredOptions?.length ?? 0}
      state={states === undefined ? { kind: "loading" } : undefined}
      renderOptions={(itemsToRender) =>
        (filteredOptions ?? [])
          .slice(0, itemsToRender)
          .map((state) => (
            <FilterOption
              key={state.id}
              isChecked={filterValue.includes(state.id)}
              onClick={() => setFilter("state", handleFilterValue(state.id))}
              icon={
                <StateGroupIcon
                  color={state.color}
                  stateGroup={state.group}
                  size={EIconSize.SM}
                  percentage={state.order}
                />
              }
              title={state.name}
            />
          ))
      }
    />
  );
};

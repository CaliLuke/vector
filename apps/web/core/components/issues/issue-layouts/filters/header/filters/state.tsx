/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useMemo } from "react";
import sortBy from "lodash-es/sortBy.js";

import { EIconSize } from "@plane/constants";
import { StateGroupIcon } from "@plane/propel/icons";
import type { IState } from "@plane/types";
// components
import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";
import { FilterOptionSection } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option-section";
// ui
// types

type Props = {
  appliedFilters: string[] | null;
  handleUpdate: (val: string) => void;
  searchQuery: string;
  states: IState[] | undefined;
};

export const FilterState = function FilterState(props: Props) {
  const { appliedFilters, handleUpdate, searchQuery, states } = props;

  const appliedFiltersCount = appliedFilters?.length ?? 0;

  const sortedOptions = useMemo(() => {
    const filteredOptions = (states ?? []).filter((s) => s.name.toLowerCase().includes(searchQuery.toLowerCase()));

    return sortBy(filteredOptions, [(s) => !(appliedFilters ?? []).includes(s.id)]);
  }, [appliedFilters, searchQuery, states]);

  return (
    <FilterOptionSection
      title={`State${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
      optionCount={sortedOptions.length}
      renderOptions={(itemsToRender) =>
        sortedOptions
          .slice(0, itemsToRender)
          .map((state) => (
            <FilterOption
              key={state.id}
              isChecked={appliedFilters?.includes(state.id) ?? false}
              onClick={() => handleUpdate(state.id)}
              icon={
                <StateGroupIcon
                  stateGroup={state.group}
                  color={state.color}
                  size={EIconSize.MD}
                  percentage={state?.order}
                />
              }
              title={state.name}
            />
          ))
      }
    />
  );
};

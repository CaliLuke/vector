/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React, { useMemo } from "react";
import sortBy from "lodash-es/sortBy.js";

import { CycleGroupIcon } from "@plane/propel/icons";
import { cycleGroup } from "@/queries/cycle-module-selectors";
// components
import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";
import { FilterOptionSection } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option-section";
import { useProjectCycles } from "@/hooks/use-cycles";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
// ui
// types

type Props = {
  appliedFilters: string[] | null;
  handleUpdate: (val: string) => void;
  searchQuery: string;
};

export const FilterCycle = function FilterCycle(props: Props) {
  const { appliedFilters, handleUpdate, searchQuery } = props;

  // hooks
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const catalogue = useProjectCycles(workspaceSlug, projectId);
  const cycles = catalogue.cycles;

  const appliedFiltersCount = appliedFilters?.length ?? 0;

  const sortedOptions = useMemo(() => {
    const filteredOptions = (cycles || []).filter((cycle) =>
      cycle.name.toLowerCase().includes(searchQuery.toLowerCase())
    );

    return sortBy(filteredOptions, [
      (cycle) => !appliedFilters?.includes(cycle.id),
      (cycle) => cycle.name.toLowerCase(),
    ]);
  }, [searchQuery, cycles, appliedFilters]);

  return (
    <FilterOptionSection
      title={`Cycle ${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
      optionCount={sortedOptions.length}
      state={
        catalogue.isError || catalogue.isAccessDenied
          ? { kind: "error", content: <CatalogueReadFailure label="cycles" retry={catalogue.retry} /> }
          : catalogue.isPending
            ? { kind: "loading", content: <output>Loading filters…</output> }
            : undefined
      }
      renderOptions={(itemsToRender) =>
        sortedOptions
          .slice(0, itemsToRender)
          .map((cycle) => (
            <FilterOption
              key={cycle.id}
              isChecked={appliedFilters?.includes(cycle.id) ?? false}
              onClick={() => handleUpdate(cycle.id)}
              icon={<CycleGroupIcon cycleGroup={cycleGroup(cycle?.status)} className="h-3.5 w-3.5 flex-shrink-0" />}
              title={cycle.name}
              activePulse={cycleGroup(cycle?.status) === "current"}
            />
          ))
      }
    />
  );
};

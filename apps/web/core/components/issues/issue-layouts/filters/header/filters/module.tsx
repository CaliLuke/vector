/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React, { useMemo } from "react";
import sortBy from "lodash-es/sortBy.js";

// components
import { ModuleIcon } from "@plane/propel/icons";
import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";
import { FilterOptionSection } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option-section";
import { useProjectModules } from "@/hooks/use-modules";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
// ui

type Props = {
  appliedFilters: string[] | null;
  handleUpdate: (val: string) => void;
  searchQuery: string;
};

export const FilterModule = function FilterModule(props: Props) {
  const { appliedFilters, handleUpdate, searchQuery } = props;
  // hooks
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const catalogue = useProjectModules(workspaceSlug, projectId);
  const modules = catalogue.modules;
  const appliedFiltersCount = appliedFilters?.length ?? 0;

  const sortedOptions = useMemo(() => {
    const filteredOptions = (modules || []).filter((module) =>
      module.name.toLowerCase().includes(searchQuery.toLowerCase())
    );

    return sortBy(filteredOptions, [
      (module) => !appliedFilters?.includes(module.id),
      (module) => module.name.toLowerCase(),
    ]);
  }, [searchQuery, modules, appliedFilters]);

  return (
    <FilterOptionSection
      title={`Module ${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
      optionCount={sortedOptions.length}
      state={
        catalogue.isError || catalogue.isAccessDenied
          ? { kind: "error", content: <CatalogueReadFailure label="modules" retry={catalogue.retry} /> }
          : catalogue.isPending
            ? { kind: "loading", content: <output>Loading filters…</output> }
            : undefined
      }
      renderOptions={(itemsToRender) =>
        sortedOptions
          .slice(0, itemsToRender)
          .map((module) => (
            <FilterOption
              key={module.id}
              isChecked={appliedFilters?.includes(module.id) ?? false}
              onClick={() => handleUpdate(module.id)}
              icon={<ModuleIcon className="h-3 w-3 flex-shrink-0" />}
              title={module.name}
            />
          ))
      }
    />
  );
};

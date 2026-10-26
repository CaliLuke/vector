/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TPageFilterProps, TPageFilters } from "@plane/types";
// components
import { WorkspaceMemberFilter } from "@/components/core/filters/workspace-member-filter";
import { FavoritesCreatedDateFilters } from "@/components/core/filters/favorites-created-date-filters";

type Props = {
  filters: TPageFilters;
  handleFiltersUpdate: <T extends keyof TPageFilters>(filterKey: T, filterValue: TPageFilters[T]) => void;
  memberIds?: string[] | undefined;
  workspaceSlug: string | undefined;
};

export const PageFiltersSelection = function PageFiltersSelection(props: Props) {
  const { filters, handleFiltersUpdate, memberIds, workspaceSlug } = props;
  // states

  const handleFilters = (key: keyof TPageFilterProps, value: boolean | string | string[]) => {
    const currentValues = filters.filters?.[key] ?? [];
    const newValues = Array.isArray(currentValues) ? [...currentValues] : currentValues;

    if (typeof newValues === "boolean" && typeof value === "boolean") return;

    if (Array.isArray(newValues)) {
      if (Array.isArray(value))
        value.forEach((val) => {
          if (!newValues.includes(val)) newValues.push(val);
          else newValues.splice(newValues.indexOf(val), 1);
        });
      else if (typeof value === "string") {
        if (newValues?.includes(value)) newValues.splice(newValues.indexOf(value), 1);
        else newValues.push(value);
      }
    }

    handleFiltersUpdate("filters", {
      ...filters.filters,
      [key]: newValues,
    });
  };

  return (
    <FavoritesCreatedDateFilters
      filters={filters.filters ?? {}}
      onChange={(updated) =>
        handleFiltersUpdate("filters", {
          ...filters.filters,
          ...updated,
        })
      }
      searchClassName="bg-layer-transparent"
      searchInputClassName="bg-transparent"
    >
      {(filtersSearchQuery) => (
        <div className="py-2">
          <WorkspaceMemberFilter
            title="Created by"
            appliedFilters={filters.filters?.created_by ?? null}
            handleUpdate={(val) => handleFilters("created_by", val)}
            searchQuery={filtersSearchQuery}
            memberIds={memberIds}
            workspaceSlug={workspaceSlug}
          />
        </div>
      )}
    </FavoritesCreatedDateFilters>
  );
};

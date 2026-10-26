/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TViewFilterProps, TViewFilters } from "@plane/types";
import { EViewAccess } from "@plane/types";
// components
import { WorkspaceMemberFilter } from "@/components/core/filters/workspace-member-filter";
import { FavoritesCreatedDateFilters } from "@/components/core/filters/favorites-created-date-filters";

type Props = {
  filters: TViewFilters;
  handleFiltersUpdate: <T extends keyof TViewFilters>(filterKey: T, filterValue: TViewFilters[T]) => void;
  memberIds?: string[] | undefined;
  workspaceSlug: string | undefined;
};

export const ViewFiltersSelection = function ViewFiltersSelection(props: Props) {
  const { filters, handleFiltersUpdate, memberIds, workspaceSlug } = props;
  // states
  // handles filter update
  const handleFilters = (key: keyof TViewFilterProps, value: boolean | string | EViewAccess | string[]) => {
    const currentValue = filters.filters?.[key];

    if (typeof currentValue === "boolean" && typeof value === "boolean") return;

    const currValues = Array.isArray(currentValue) ? [...currentValue] : [];
    if (Array.isArray(value)) {
      value.forEach((val) => {
        if (!currValues.includes(val)) currValues.push(val);
        else currValues.splice(currValues.indexOf(val), 1);
      });
    } else if (typeof value !== "boolean") {
      if (currValues?.includes(value)) currValues.splice(currValues.indexOf(value), 1);
      else currValues.push(value);
    }

    handleFiltersUpdate("filters", {
      ...filters.filters,
      [key]: currValues,
    });
  };

  return (
    <FavoritesCreatedDateFilters
      filters={filters.filters ?? {}}
      onChange={(updated) => handleFiltersUpdate("filters", { ...filters.filters, ...updated })}
    >
      {(filtersSearchQuery) => (
        <div className="py-2">
          <WorkspaceMemberFilter
            title="Created by"
            appliedFilters={filters.filters?.owned_by ?? null}
            handleUpdate={(val) => handleFilters("owned_by", val)}
            searchQuery={filtersSearchQuery}
            memberIds={memberIds}
            workspaceSlug={workspaceSlug}
          />
        </div>
      )}
    </FavoritesCreatedDateFilters>
  );
};

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import type { EViewAccess, TViewFilterProps } from "@plane/types";
import { AppliedFiltersRow } from "@/components/common/applied-filters/tag";
// components
import { AppliedDateFilters } from "@/components/common/applied-filters/date";
import { AppliedMembersFilters } from "@/components/common/applied-filters/members";
// local imports
import { AppliedAccessFilters } from "./access";

const FILTER_KEYS = [
  "created_at",
  "owned_by",
  "favorites",
  "view_type",
] as const satisfies readonly (keyof TViewFilterProps)[];

type Props = {
  appliedFilters: TViewFilterProps;
  handleClearAllFilters: () => void;
  handleRemoveFilter: (key: keyof TViewFilterProps, value: string | EViewAccess | null) => void;
  alwaysAllowEditing?: boolean;
};

export function ViewAppliedFiltersList(props: Props) {
  const { workspaceSlug } = useParams({ strict: false });
  const { appliedFilters, handleClearAllFilters, handleRemoveFilter, alwaysAllowEditing } = props;

  const isEditingAllowed = alwaysAllowEditing;

  return (
    <AppliedFiltersRow
      filters={appliedFilters}
      keys={FILTER_KEYS}
      onRemove={isEditingAllowed ? (key) => handleRemoveFilter(key, null) : undefined}
      onClearAll={isEditingAllowed ? handleClearAllFilters : undefined}
    >
      {(filterKey) => (
        <>
          {filterKey === "view_type" && (
            <AppliedAccessFilters
              editable={isEditingAllowed}
              handleRemove={(val) => handleRemoveFilter(filterKey, val)}
              values={appliedFilters.view_type ?? []}
            />
          )}
          {filterKey === "created_at" && (
            <AppliedDateFilters
              editable={isEditingAllowed}
              handleRemove={(val) => handleRemoveFilter(filterKey, val)}
              values={appliedFilters.created_at ?? []}
            />
          )}
          {filterKey === "owned_by" && (
            <AppliedMembersFilters
              workspaceSlug={workspaceSlug}
              editable={isEditingAllowed}
              handleRemove={(val) => handleRemoveFilter(filterKey, val)}
              values={appliedFilters.owned_by ?? []}
            />
          )}
        </>
      )}
    </AppliedFiltersRow>
  );
}

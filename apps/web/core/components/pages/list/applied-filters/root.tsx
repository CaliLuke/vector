/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ClearAppliedFilters, RemoveAppliedFilter } from "@/components/common/applied-filters/tag";
import { orderedFilterKeys } from "@/components/common/applied-filters/keys";
import { useParams } from "@tanstack/react-router";
// plane imports
import type { TPageFilterProps } from "@plane/types";
import { Tag } from "@plane/ui";
import { replaceUnderscoreIfSnakeCase } from "@plane/utils/string";
// components
import { AppliedDateFilters } from "@/components/common/applied-filters/date";
import { AppliedMembersFilters } from "@/components/common/applied-filters/members";

const FILTER_KEYS = [
  "created_at",
  "created_by",
  "favorites",
  "labels",
] as const satisfies readonly (keyof TPageFilterProps)[];

type Props = {
  appliedFilters: TPageFilterProps;
  handleClearAllFilters: () => void;
  handleRemoveFilter: (key: keyof TPageFilterProps, value: string | null) => void;
  alwaysAllowEditing?: boolean;
};

const MEMBERS_FILTERS = new Set(["created_by"]);
const DATE_FILTERS = new Set(["created_at"]);

export function PageAppliedFiltersList(props: Props) {
  const { workspaceSlug } = useParams({ strict: false });
  const { appliedFilters, handleClearAllFilters, handleRemoveFilter, alwaysAllowEditing } = props;

  if (!appliedFilters) return null;
  if (Object.keys(appliedFilters).length === 0) return null;

  const isEditingAllowed = alwaysAllowEditing;

  return (
    <div className="flex flex-wrap items-stretch gap-2">
      {orderedFilterKeys(appliedFilters, FILTER_KEYS).map((filterKey) => {
        const value = appliedFilters[filterKey];

        if (!value) return undefined;
        if (Array.isArray(value) && value.length === 0) return undefined;

        return (
          <Tag key={filterKey}>
            <div className="flex flex-wrap items-center gap-1.5">
              <span className="text-11 text-tertiary">{replaceUnderscoreIfSnakeCase(filterKey)}</span>
              {DATE_FILTERS.has(filterKey) && (
                <AppliedDateFilters
                  editable={isEditingAllowed}
                  handleRemove={(val) => handleRemoveFilter(filterKey, val)}
                  values={Array.isArray(value) ? value : []}
                />
              )}
              {MEMBERS_FILTERS.has(filterKey) && (
                <AppliedMembersFilters
                  workspaceSlug={workspaceSlug}
                  editable={isEditingAllowed}
                  handleRemove={(val) => handleRemoveFilter(filterKey, val)}
                  values={Array.isArray(value) ? value : []}
                />
              )}
              {isEditingAllowed && (
                <RemoveAppliedFilter filterKey={filterKey} onRemove={() => handleRemoveFilter(filterKey, null)} />
              )}
            </div>
          </Tag>
        );
      })}
      {isEditingAllowed && <ClearAppliedFilters onClear={handleClearAllFilters} />}
    </div>
  );
}

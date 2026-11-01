/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { EUserPermissions, DATE_AFTER_FILTER_OPTIONS } from "@plane/constants";
import type { TCycleFilters } from "@plane/types";
import { AppliedFiltersRow } from "@/components/common/applied-filters/tag";
// hooks
import { useProjectPermissions } from "@/hooks/use-permissions";
// local imports
import { AppliedDateFilters } from "@/components/common/applied-filters/date";
import { AppliedStatusFilters } from "./status";

const FILTER_KEYS = ["end_date", "start_date", "status"] as const satisfies readonly (keyof TCycleFilters)[];

type Props = {
  appliedFilters: TCycleFilters;
  handleClearAllFilters: () => void;
  handleRemoveFilter: (key: keyof TCycleFilters, value: string | null) => void;
  alwaysAllowEditing?: boolean;
};

const DATE_FILTERS = new Set<string>(["start_date", "end_date"]);

export const CycleAppliedFiltersList = function CycleAppliedFiltersList(props: Props) {
  const { appliedFilters, handleClearAllFilters, handleRemoveFilter, alwaysAllowEditing } = props;
  // store hooks
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);

  const isEditingAllowed = alwaysAllowEditing || canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  return (
    <AppliedFiltersRow
      filters={appliedFilters}
      keys={FILTER_KEYS}
      onRemove={isEditingAllowed ? (key) => handleRemoveFilter(key, null) : undefined}
      onClearAll={isEditingAllowed ? handleClearAllFilters : undefined}
      valuesClassName="flex flex-wrap items-center gap-1"
    >
      {(filterKey) => (
        <>
          {filterKey === "status" && (
            <AppliedStatusFilters
              editable={isEditingAllowed}
              handleRemove={(val) => handleRemoveFilter("status", val)}
              values={appliedFilters[filterKey] ?? []}
            />
          )}
          {DATE_FILTERS.has(filterKey) && (
            <AppliedDateFilters
              options={DATE_AFTER_FILTER_OPTIONS}
              className="bg-layer-3"
              editable={isEditingAllowed}
              handleRemove={(val) => handleRemoveFilter(filterKey, val)}
              values={appliedFilters[filterKey] ?? []}
            />
          )}
        </>
      )}
    </AppliedFiltersRow>
  );
};

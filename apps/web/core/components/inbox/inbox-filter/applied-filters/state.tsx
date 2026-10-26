/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { EIconSize } from "@plane/constants";
import { StateGroupIcon } from "@plane/propel/icons";
// hooks
import { useIntakePreferences } from "@/providers/intake-preferences";
import { useRouteProjectStates } from "@/hooks/use-project-states";
import { InboxAppliedFilterGroup, toggleAppliedFilterValue } from "./shared";

export const InboxIssueAppliedFiltersState = function InboxIssueAppliedFiltersState() {
  // hooks
  const { filters, setFilter } = useIntakePreferences();
  const { projectId } = useParams({ strict: false });
  const { getStateById } = useRouteProjectStates(projectId);
  // derived values
  const filteredValues = filters?.state || [];
  const currentOptionDetail = (stateId: string) => getStateById(stateId) || undefined;

  const clearFilter = () => setFilter("state", undefined);
  const options = filteredValues.flatMap((value) => {
    const optionDetail = currentOptionDetail(value);
    if (!optionDetail) return [];
    return [
      {
        key: value,
        leading: (
          <div className="relative flex h-3 w-3 flex-shrink-0 items-center justify-center overflow-hidden">
            <StateGroupIcon color={optionDetail.color} stateGroup={optionDetail.group} size={EIconSize.SM} />
          </div>
        ),
        label: optionDetail.name,
        onRemove: () => setFilter("state", toggleAppliedFilterValue(filteredValues, optionDetail.id)),
      },
    ];
  });

  return (
    <InboxAppliedFilterGroup
      label="State"
      options={options}
      hasFilters={filteredValues.length > 0}
      onClearAll={clearFilter}
    />
  );
};

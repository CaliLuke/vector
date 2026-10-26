/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { PAST_DURATION_FILTER_OPTIONS } from "@plane/constants";
import type { TInboxIssueFilterDateKeys } from "@plane/types";
// helpers
import { renderFormattedDate } from "@plane/utils/datetime";
// constants
// hooks
import { useIntakePreferences } from "@/providers/intake-preferences";
import { InboxAppliedFilterGroup, toggleAppliedFilterValue } from "./shared";

type TInboxIssueAppliedFiltersDateProps = {
  filterKey: TInboxIssueFilterDateKeys;
  label: string;
};

function getDateOptionDetail(date: string) {
  const currentDate = PAST_DURATION_FILTER_OPTIONS.find((option) => option.value === date);
  if (currentDate) return currentDate;
  const dateSplit = date.split(";");
  return {
    name: `${dateSplit[1].charAt(0).toUpperCase() + dateSplit[1].slice(1)} ${renderFormattedDate(dateSplit[0])}`,
    value: date,
  };
}

export const InboxIssueAppliedFiltersDate = function InboxIssueAppliedFiltersDate(
  props: TInboxIssueAppliedFiltersDateProps
) {
  const { filterKey, label } = props;
  // hooks
  const { filters, setFilter } = useIntakePreferences();
  // derived values
  const filteredValues = filters?.[filterKey] || [];
  const clearFilter = () => setFilter(filterKey, undefined);
  const options = filteredValues.map((value) => {
    const optionDetail = getDateOptionDetail(value);
    return {
      key: value,
      label: optionDetail.name,
      onRemove: () => setFilter(filterKey, toggleAppliedFilterValue(filteredValues, optionDetail.value)),
    };
  });

  return <InboxAppliedFilterGroup label={label} options={options} onClearAll={clearFilter} />;
};

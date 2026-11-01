/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { PAST_DURATION_FILTER_OPTIONS } from "@plane/constants";
import type { TInboxIssueFilterDateKeys } from "@plane/types";
import { DatePresetFilter } from "@/components/core/filters/date-preset-filter";
import { useIntakePreferences } from "@/providers/intake-preferences";

type Props = { filterKey: TInboxIssueFilterDateKeys; label?: string; searchQuery: string };

export function FilterDate({ filterKey, label = "Created date", searchQuery }: Props) {
  const { filters, setFilter } = useIntakePreferences();
  const values = filters?.[filterKey] ?? [];
  return (
    <DatePresetFilter
      title={label}
      appliedFilters={values}
      searchQuery={searchQuery}
      options={PAST_DURATION_FILTER_OPTIONS}
      multiple={false}
      handleUpdate={(value) =>
        setFilter(filterKey, Array.isArray(value) ? value : values.includes(value) ? [] : [value])
      }
    />
  );
}

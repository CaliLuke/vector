/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ISSUE_PRIORITIES } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { PriorityIcon } from "@plane/propel/icons";
import type { TIssuePriorities } from "@plane/types";
// plane constants
// components
import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";
import { FilterOptionSection } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option-section";
// hooks
import { useIntakePreferences } from "@/providers/intake-preferences";

type Props = {
  searchQuery: string;
};

export const FilterPriority = function FilterPriority(props: Props) {
  const { searchQuery } = props;
  // hooks
  const { t } = useTranslation();
  const { filters, setFilter } = useIntakePreferences();
  // derived values
  const filterValue = filters?.priority || [];
  const appliedFiltersCount = filterValue?.length ?? 0;
  const filteredOptions = ISSUE_PRIORITIES.filter((p) => p.key.includes(searchQuery.toLowerCase()));

  const handleFilterValue = (value: TIssuePriorities): TIssuePriorities[] =>
    filterValue?.includes(value) ? filterValue.filter((v) => v !== value) : [...filterValue, value];

  return (
    <FilterOptionSection
      title={`${t("common.priority")}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
      optionCount={filteredOptions.length}
      emptyMessage={<p className="text-11 text-placeholder italic">{t("common.search.no_matches_found")}</p>}
      renderOptions={(itemsToRender) =>
        filteredOptions
          .slice(0, itemsToRender)
          .map((priority) => (
            <FilterOption
              key={priority.key}
              isChecked={filterValue.includes(priority.key)}
              onClick={() => setFilter("priority", handleFilterValue(priority.key))}
              icon={<PriorityIcon priority={priority.key} className="h-3.5 w-3.5" />}
              title={priority.title}
            />
          ))
      }
    />
  );
};

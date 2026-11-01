/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { TActivityFilters, TActivityFilterOption } from "@plane/constants";
import { ACTIVITY_FILTER_TYPE_OPTIONS, EActivityFilterType } from "@plane/constants";
// components
import { ActivityFilter } from "@/components/issues/issue-detail/issue-activity";

export type TActivityFilterRoot = {
  selectedFilters: TActivityFilters[];
  toggleFilter: (filter: TActivityFilters) => void;
  projectId: string;
  isIntakeIssue?: boolean;
};

export function ActivityFilterRoot(props: TActivityFilterRoot) {
  const { selectedFilters, toggleFilter } = props;

  const filters: TActivityFilterOption[] = Object.values(EActivityFilterType)
    .filter((key) => key !== EActivityFilterType.DEFAULT)
    .map((filterKey) => {
      const value = ACTIVITY_FILTER_TYPE_OPTIONS[filterKey];
      return {
        key: filterKey,
        labelTranslationKey: value.labelTranslationKey,
        isSelected: selectedFilters.includes(filterKey),
        onClick: () => toggleFilter(filterKey),
      };
    });

  return <ActivityFilter selectedFilters={selectedFilters} filterOptions={filters} />;
}

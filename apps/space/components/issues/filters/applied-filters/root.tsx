/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// hooks
import { usePublicIssueView } from "@/hooks/use-public-issue-view";
import type { TIssueFilterKeys } from "@/types/issue";
// components
import { AppliedFiltersList } from "./filters-list";

type TIssueAppliedFilters = {
  anchor: string;
};

export const IssueAppliedFilters = function IssueAppliedFilters(props: TIssueAppliedFilters) {
  const { anchor } = props;
  const { filters, setFilter, clearFilters } = usePublicIssueView(anchor);
  const appliedFilters = Object.fromEntries(Object.entries(filters).filter(([, values]) => values.length));
  const handleFilters = (key: TIssueFilterKeys, value: string | null) => {
    setFilter(key, value === null ? [] : filters[key].filter((item) => item !== value));
  };

  if (Object.keys(appliedFilters).length === 0) return null;

  return (
    <div className="border-b border-subtle bg-surface-1 p-4">
      <AppliedFiltersList
        appliedFilters={appliedFilters || {}}
        handleRemoveFilter={handleFilters}
        handleRemoveAllFilters={clearFilters}
      />
    </div>
  );
};

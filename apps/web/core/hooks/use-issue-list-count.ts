/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useIssueFilterPreferences } from "@/hooks/use-issue-filter-preferences";
import { useIssueListPaginationOptions } from "@/hooks/use-issue-list-pagination-options";
import { useIssueListPresentation } from "@/hooks/use-issue-list-presentation";

/** Read the count from the same grouped/list query used by the active layout. */
export function useIssueListCount(storeType: Parameters<typeof useIssueFilterPreferences>[0]) {
  const preferences = useIssueFilterPreferences(storeType);
  const displayFilters = preferences.issueFilters?.displayFilters;
  const pagination = useIssueListPaginationOptions(storeType, displayFilters);
  const issueList = useIssueListPresentation(storeType, pagination, displayFilters);
  return {
    count: issueList.getGroupIssueCount(undefined, undefined, false),
    isPending: issueList.isPending,
    isError: issueList.isError,
    retry: issueList.retry,
  };
}

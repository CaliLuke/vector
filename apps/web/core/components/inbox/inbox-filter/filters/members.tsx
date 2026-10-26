/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TInboxIssueFilterMemberKeys } from "@plane/types";
import { WorkspaceMemberFilter } from "@/components/core/filters/workspace-member-filter";
import { useIntakePreferences } from "@/providers/intake-preferences";

const EMPTY_MEMBER_FILTER: string[] = [];

type Props = {
  filterKey: TInboxIssueFilterMemberKeys;
  label?: string;
  memberIds: string[] | undefined;
  searchQuery: string;
  workspaceSlug: string | undefined;
};

export const FilterMember = function FilterMember(props: Props) {
  const { filterKey, label = "Members", memberIds, searchQuery, workspaceSlug } = props;
  const { filters, setFilter } = useIntakePreferences();
  const appliedFilters = filters?.[filterKey] ?? EMPTY_MEMBER_FILTER;
  const handleUpdate = (memberId: string) => {
    const nextValue = appliedFilters.includes(memberId)
      ? appliedFilters.filter((value) => value !== memberId)
      : [...appliedFilters, memberId];
    setFilter(filterKey, nextValue);
  };

  return (
    <WorkspaceMemberFilter
      appliedFilters={appliedFilters}
      handleUpdate={handleUpdate}
      memberIds={memberIds}
      searchQuery={searchQuery}
      title={label}
      workspaceSlug={workspaceSlug}
    />
  );
};

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane types
import type { TInboxIssueFilterMemberKeys } from "@plane/types";
// plane ui
import { Avatar } from "@plane/ui";
// helpers
import { getFileURL } from "@plane/utils/file";
// hooks
import { useIntakePreferences } from "@/providers/intake-preferences";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { InboxAppliedFilterGroup, toggleAppliedFilterValue } from "./shared";

type TInboxIssueAppliedFiltersMemberProps = {
  filterKey: TInboxIssueFilterMemberKeys;
  label: string;
};

export const InboxIssueAppliedFiltersMember = function InboxIssueAppliedFiltersMember(
  props: TInboxIssueAppliedFiltersMemberProps
) {
  const { filterKey, label } = props;
  const { workspaceSlug } = useParams({ strict: false });
  // hooks
  const { filters, setFilter } = useIntakePreferences();
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  // derived values
  const filteredValues = filters?.[filterKey] || [];
  const currentOptionDetail = (memberId: string) => membersById?.get(memberId)?.member;

  const clearFilter = () => setFilter(filterKey, undefined);
  const options = filteredValues.flatMap((value) => {
    const optionDetail = currentOptionDetail(value);
    if (!optionDetail) return [];
    return [
      {
        key: value,
        leading: (
          <div className="relative flex flex-shrink-0 items-center justify-center overflow-hidden">
            <Avatar
              name={optionDetail.display_name}
              src={getFileURL(optionDetail.avatar_url)}
              showTooltip={false}
              size="sm"
            />
          </div>
        ),
        label: optionDetail.display_name,
        onRemove: () => setFilter(filterKey, toggleAppliedFilterValue(filteredValues, value)),
      },
    ];
  });

  return (
    <InboxAppliedFilterGroup
      label={label}
      options={options}
      hasFilters={filteredValues.length > 0}
      onClearAll={clearFilter}
    />
  );
};

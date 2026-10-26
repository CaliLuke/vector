/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMemo } from "react";
import sortBy from "lodash-es/sortBy.js";
import { Avatar } from "@plane/ui";
import { getFileURL } from "@plane/utils/file";
import { FilterOption } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option";
import { FilterOptionSection } from "@/components/issues/issue-layouts/filters/header/helpers/filter-option-section";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";

export type WorkspaceMemberFilterProps = {
  appliedFilters: string[] | null;
  handleUpdate: (value: string) => void;
  memberIds: string[] | undefined;
  searchQuery: string;
  title: string;
  workspaceSlug: string | undefined;
};

export function WorkspaceMemberFilter(props: WorkspaceMemberFilterProps) {
  const { appliedFilters, handleUpdate, memberIds, searchQuery, title, workspaceSlug } = props;
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const { data: currentUser } = useCurrentUser();
  const getUserDetails = (memberId: string) => membersById?.get(memberId)?.member;

  const appliedFiltersCount = appliedFilters?.length ?? 0;

  const filteredOptions = useMemo(
    () =>
      (memberIds ?? []).filter((memberId) =>
        membersById?.get(memberId)?.member?.display_name.toLowerCase().includes(searchQuery.toLowerCase())
      ),
    [memberIds, membersById, searchQuery]
  );

  const sortedOptions = useMemo(
    () =>
      sortBy(filteredOptions, [
        (memberId) => !(appliedFilters ?? []).includes(memberId),
        (memberId) => memberId !== currentUser?.id,
        (memberId) => membersById?.get(memberId)?.member?.display_name.toLowerCase(),
      ]),
    [appliedFilters, currentUser?.id, filteredOptions, membersById]
  );

  return (
    <FilterOptionSection
      title={`${title}${appliedFiltersCount > 0 ? ` (${appliedFiltersCount})` : ""}`}
      optionCount={sortedOptions.length}
      renderOptions={(itemsToRender) =>
        sortedOptions.slice(0, itemsToRender).map((memberId) => {
          const member = getUserDetails(memberId);
          if (!member) return null;
          return (
            <FilterOption
              key={member.id}
              isChecked={Boolean(appliedFilters?.includes(member.id))}
              onClick={() => handleUpdate(member.id)}
              icon={
                <Avatar name={member.display_name} src={getFileURL(member.avatar_url)} showTooltip={false} size="md" />
              }
              title={currentUser?.id === member.id ? "You" : member.display_name}
            />
          );
        })
      }
    />
  );
}

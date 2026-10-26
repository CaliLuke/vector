/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback } from "react";
import { ListFilter } from "lucide-react";
// plane imports
import { useTranslation } from "@plane/i18n";
import type { TPageFilterProps, TPageNavigationTabs } from "@plane/types";
import { Header, EHeaderVariant } from "@plane/ui";
import { calculateTotalFilters } from "@plane/utils/filter";
// components
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { ControlledExpandableSearch } from "@/components/common/controlled-expandable-search";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { workspaceRosterMemberIds } from "@/helpers/member-rosters";
import { usePageList } from "../page-record-context";
// local imports
import { PageAppliedFiltersList } from "../list/applied-filters";
import { PageFiltersSelection } from "../list/filters";
import { PageOrderByDropdown } from "../list/order-by";
import { PageTabNavigation } from "../list/tab-navigation";

type Props = {
  pageType: TPageNavigationTabs;
  projectId: string;
  workspaceSlug: string;
};

export function PagesListHeaderRoot(props: Props) {
  const { pageType, projectId, workspaceSlug } = props;
  const { t } = useTranslation();
  const { data: currentUser } = useCurrentUser();
  // store hooks
  const { filters, updateFilters, clearAllFilters } = usePageList();
  const { members } = useWorkspaceMemberRoster(workspaceSlug);
  const workspaceMemberIds = workspaceRosterMemberIds(members, currentUser?.id);

  const handleRemoveFilter = useCallback(
    (key: keyof TPageFilterProps, value: string | null) => {
      let newValues = filters.filters?.[key];

      if (key === "favorites") newValues = !!value;
      if (Array.isArray(newValues)) {
        if (!value) newValues = [];
        else newValues = newValues.filter((val) => val !== value);
      }

      updateFilters("filters", { [key]: newValues });
    },
    [filters.filters, updateFilters]
  );

  const isFiltersApplied = calculateTotalFilters(filters?.filters ?? {}) !== 0;

  return (
    <>
      <Header variant={EHeaderVariant.SECONDARY}>
        <Header.LeftItem>
          <PageTabNavigation workspaceSlug={workspaceSlug} projectId={projectId} pageType={pageType} />
        </Header.LeftItem>
        <Header.RightItem className="items-center">
          <ControlledExpandableSearch
            value={filters.searchQuery}
            onChange={(searchQuery) => updateFilters("searchQuery", searchQuery)}
            placeholder="Search pages"
            className="flex"
            triggerClassName="my-auto -mr-1 shrink-0"
            expandedFieldClassName="w-64 border-subtle px-2.5 py-1.5"
            inputClassName="ml-2"
            searchIconClassName="h-3.5 w-3.5"
          />
          <PageOrderByDropdown
            sortBy={filters.sortBy}
            sortKey={filters.sortKey}
            onChange={(val) => {
              if (val.key) updateFilters("sortKey", val.key);
              if (val.order) updateFilters("sortBy", val.order);
            }}
          />
          <FiltersDropdown
            icon={<ListFilter className="h-3 w-3" />}
            title={t("common.filters")}
            placement="bottom-end"
            isFiltersApplied={isFiltersApplied}
          >
            <PageFiltersSelection
              filters={filters}
              handleFiltersUpdate={updateFilters}
              memberIds={workspaceMemberIds}
              workspaceSlug={workspaceSlug}
            />
          </FiltersDropdown>
        </Header.RightItem>
      </Header>
      {calculateTotalFilters(filters?.filters ?? {}) !== 0 && (
        <Header variant={EHeaderVariant.TERNARY}>
          <PageAppliedFiltersList
            appliedFilters={filters.filters ?? {}}
            handleClearAllFilters={clearAllFilters}
            handleRemoveFilter={handleRemoveFilter}
            alwaysAllowEditing
          />
        </Header>
      )}
    </>
  );
}

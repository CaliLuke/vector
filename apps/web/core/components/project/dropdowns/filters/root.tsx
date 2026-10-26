/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
// plane imports
import type { TProjectDisplayFilters, TProjectFilters } from "@plane/types";
// components
import { FilterOption } from "@/components/issues/issue-layouts/filters";
import { FilterSelectionSearch } from "@/components/common/filters/filter-selection-search";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { FilterAccess } from "./access";
import { DatePresetFilter } from "@/components/core/filters/date-preset-filter";
import { PROJECT_CREATED_AT_FILTER_OPTIONS } from "@plane/constants";
import { FilterLead } from "./lead";
import { FilterMembers } from "./members";

type Props = {
  displayFilters: TProjectDisplayFilters;
  filters: TProjectFilters;
  handleFiltersUpdate: (key: keyof TProjectFilters, value: string | string[]) => void;
  handleDisplayFiltersUpdate: (updatedDisplayProperties: Partial<TProjectDisplayFilters>) => void;
  memberIds?: string[] | undefined;
  workspaceSlug: string | undefined;
};

export const ProjectFiltersSelection = function ProjectFiltersSelection(props: Props) {
  const { displayFilters, filters, handleFiltersUpdate, handleDisplayFiltersUpdate, memberIds, workspaceSlug } = props;
  // states
  const [filtersSearchQuery, setFiltersSearchQuery] = useState("");
  // store
  const { isMobile } = usePlatformOS();

  return (
    <div className="flex h-full w-full flex-col overflow-hidden">
      <FilterSelectionSearch value={filtersSearchQuery} onChange={setFiltersSearchQuery} autoFocus={!isMobile} />
      <div className="vertical-scrollbar scrollbar-sm h-full w-full divide-y divide-subtle-1 overflow-y-auto px-2.5">
        <div className="py-2">
          <FilterOption
            isChecked={!!displayFilters.my_projects}
            onClick={() =>
              handleDisplayFiltersUpdate({
                my_projects: !displayFilters.my_projects,
              })
            }
            title="My projects"
          />
        </div>

        {/* access */}
        <div className="py-2">
          <FilterAccess
            appliedFilters={filters.access ?? null}
            handleUpdate={(val) => handleFiltersUpdate("access", val)}
            searchQuery={filtersSearchQuery}
          />
        </div>

        {/* lead */}
        <div className="py-2">
          <FilterLead
            appliedFilters={filters.lead ?? null}
            handleUpdate={(val) => handleFiltersUpdate("lead", val)}
            searchQuery={filtersSearchQuery}
            memberIds={memberIds}
            workspaceSlug={workspaceSlug}
          />
        </div>

        {/* members */}
        <div className="py-2">
          <FilterMembers
            appliedFilters={filters.members ?? null}
            handleUpdate={(val) => handleFiltersUpdate("members", val)}
            searchQuery={filtersSearchQuery}
            memberIds={memberIds}
            workspaceSlug={workspaceSlug}
          />
        </div>

        {/* created date */}
        <div className="py-2">
          <DatePresetFilter
            title="Created date"
            options={PROJECT_CREATED_AT_FILTER_OPTIONS}
            multiple={false}
            appliedFilters={filters.created_at ?? null}
            handleUpdate={(val) => handleFiltersUpdate("created_at", val)}
            searchQuery={filtersSearchQuery}
          />
        </div>
      </div>
    </div>
  );
};

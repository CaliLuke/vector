/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { ListFilter } from "lucide-react";
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { ControlledExpandableSearch } from "@/components/common/controlled-expandable-search";
import { ArchiveTabsList } from "./archive-tabs-list";

type Props = {
  extraControls?: ReactNode;
  filters: ReactNode;
  isFiltersApplied: boolean;
  onSearchChange: (value: string) => void;
  searchQuery: string;
};

export function ArchiveCatalogueHeader(props: Props) {
  const { extraControls, filters, isFiltersApplied, onSearchChange, searchQuery } = props;
  return (
    <div className="group relative flex border-b border-subtle">
      <div className="horizontal-scrollbar flex scrollbar-sm w-full items-center gap-2 overflow-x-auto px-4">
        <ArchiveTabsList />
      </div>
      <div className="flex h-full items-center gap-3 self-end px-8">
        <ControlledExpandableSearch
          value={searchQuery}
          onChange={onSearchChange}
          placeholder="Search"
          initiallyOpen={searchQuery !== ""}
          className="flex items-center"
          triggerClassName="-mr-5 rounded-sm p-2 hover:bg-layer-1"
          fieldClassName="ml-auto gap-1 bg-surface-1"
          expandedFieldClassName="w-64 border-subtle px-2.5 py-1.5"
          searchIconClassName="h-3.5 w-3.5"
        />
        {extraControls}
        <FiltersDropdown
          icon={<ListFilter className="h-3 w-3" />}
          title="Filters"
          placement="bottom-end"
          isFiltersApplied={isFiltersApplied}
        >
          {filters}
        </FiltersDropdown>
      </div>
    </div>
  );
}

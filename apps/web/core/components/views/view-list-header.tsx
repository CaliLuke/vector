/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useRef, useState } from "react";
import { useParams } from "@tanstack/react-router";
// icons
import { ListFilter } from "lucide-react";
import { useOutsideClickDetector } from "@plane/hooks";
import { SearchIcon, CloseIcon } from "@plane/propel/icons";
// plane helpers
// helpers
import { cn } from "@plane/utils/common";
// hooks
import { useCurrentUser } from "@/hooks/use-current-user";
import { useProjectViewListFilters } from "@/providers/project-view-list-filters";
import { useProjectMemberRoster, useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { projectRosterMemberIds } from "@/helpers/member-rosters";
import { FiltersDropdown } from "../issues/issue-layouts/filters";
import { ViewFiltersSelection } from "./filters/filter-selection";
import { ViewOrderByDropdown } from "./filters/order-by";
import { IconButton } from "@plane/propel/icon-button";

export function ViewListHeader() {
  // states
  const [searchRequested, setIsSearchOpen] = useState(false);
  // refs
  const inputRef = useRef<HTMLInputElement>(null);
  // store hooks
  const { filters, updateFilters } = useProjectViewListFilters();
  const isSearchOpen = searchRequested || filters.searchQuery.trim() !== "";
  const { workspaceSlug, projectId } = useParams({ strict: false });
  const { data: currentUser } = useCurrentUser();
  const workspaceRoster = useWorkspaceMemberRoster(workspaceSlug);
  const projectRoster = useProjectMemberRoster(workspaceSlug, projectId);
  const projectMemberIds = projectId
    ? projectRosterMemberIds(projectRoster.memberships, workspaceRoster.membersById, currentUser?.id, false)
    : undefined;

  // handlers
  const handleInputKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Escape") {
      if (filters?.searchQuery && filters?.searchQuery.trim() !== "") {
        updateFilters("searchQuery", "");
      } else {
        setIsSearchOpen(false);
        inputRef.current?.blur();
      }
    }
  };

  // outside click detector hook
  useOutsideClickDetector(inputRef, () => {
    if (isSearchOpen && filters?.searchQuery.trim() === "") setIsSearchOpen(false);
  });

  return (
    <div className="flex h-full items-center gap-2">
      <div className="flex items-center">
        {!isSearchOpen && (
          <IconButton
            variant="ghost"
            size="lg"
            className="-mr-1"
            onClick={() => {
              setIsSearchOpen(true);
              inputRef.current?.focus();
            }}
            icon={SearchIcon}
          />
        )}
        <div
          className={cn(
            "ml-auto flex w-0 items-center justify-start gap-1 overflow-hidden rounded-md border border-transparent bg-surface-1 text-placeholder opacity-0 transition-[width] ease-linear",
            {
              "w-64 border-subtle px-2.5 py-1.5 opacity-100": isSearchOpen,
            }
          )}
        >
          <SearchIcon className="h-3.5 w-3.5" />
          <input
            ref={inputRef}
            className="w-full max-w-[234px] border-none bg-transparent text-13 text-primary placeholder:text-placeholder focus:outline-none"
            placeholder="Search"
            value={filters?.searchQuery}
            onChange={(e) => updateFilters("searchQuery", e.target.value)}
            onKeyDown={handleInputKeyDown}
          />
          {isSearchOpen && (
            <button
              type="button"
              className="grid place-items-center"
              onClick={() => {
                updateFilters("searchQuery", "");
                setIsSearchOpen(false);
              }}
            >
              <CloseIcon className="h-3 w-3" />
            </button>
          )}
        </div>
      </div>
      <div className="hidden items-center gap-2 md:flex">
        <ViewOrderByDropdown
          sortBy={filters.sortBy}
          sortKey={filters.sortKey}
          onChange={(val) => {
            if (val.key) updateFilters("sortKey", val.key);
            if (val.order) updateFilters("sortBy", val.order);
          }}
        />
        <FiltersDropdown
          icon={<ListFilter className="h-3 w-3" />}
          title="Filters"
          placement="bottom-end"
          isFiltersApplied={false}
        >
          <ViewFiltersSelection
            filters={filters}
            handleFiltersUpdate={updateFilters}
            memberIds={projectMemberIds ?? undefined}
            workspaceSlug={workspaceSlug}
          />
        </FiltersDropdown>
      </div>
    </div>
  );
}

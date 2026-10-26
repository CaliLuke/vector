/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useCallback } from "react";

import { ListFilter } from "lucide-react";
// plane helpers
import { MODULE_VIEW_LAYOUTS } from "@plane/constants";
// types
import { useTranslation } from "@plane/i18n";
import { Tooltip } from "@plane/propel/tooltip";
import type { TModuleFilters } from "@plane/types";
// ui
import { cn } from "@plane/utils/common";
import { calculateTotalFilters } from "@plane/utils/filter";
// plane utils
// components
import { FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { ControlledExpandableSearch } from "@/components/common/controlled-expandable-search";
import { ModuleFiltersSelection, ModuleOrderByDropdown } from "@/components/modules/dropdowns";
// constants
// helpers
// hooks
import { useModuleCataloguePreferences } from "@/providers/cycle-module-preferences";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { workspaceRosterMemberIds } from "@/helpers/member-rosters";
import { ModuleLayoutIcon } from "./module-layout-icon";
import { toggleCatalogueFilterValues } from "../cycles-modules/toggle-catalogue-filter-values";
// i18n

export const ModuleViewHeader = function ModuleViewHeader() {
  // router
  const { projectId, workspaceSlug } = useParams({ strict: false });
  // hooks
  const { isMobile } = usePlatformOS();
  // store hooks
  const { data: currentUser } = useCurrentUser();
  const { members } = useWorkspaceMemberRoster(workspaceSlug);
  const workspaceMemberIds = workspaceRosterMemberIds(members, currentUser?.id);
  const preferences = useModuleCataloguePreferences(workspaceSlug, projectId);
  const { display: displayFilters, filters, search: searchQuery, update } = preferences;
  const { t } = useTranslation();

  // handlers
  const handleFilters = useCallback(
    (key: keyof TModuleFilters, value: string | string[]) => {
      if (!projectId) return;
      const newValues = toggleCatalogueFilterValues(filters?.[key], value);

      update({ filters: { ...filters, [key]: newValues } });
    },
    [filters, projectId, update]
  );

  const isFiltersApplied = calculateTotalFilters(filters ?? {}) !== 0 || displayFilters?.favorites;

  return (
    <div className="hidden h-full items-center gap-2 self-end sm:flex">
      <div className="flex items-center">
        <ControlledExpandableSearch
          value={searchQuery}
          onChange={(search) => update({ search })}
          placeholder="Search"
          initiallyOpen={searchQuery !== ""}
          clearOnClose={false}
          className="flex items-center"
          triggerClassName="p- -mr-1"
          fieldClassName="ml-auto gap-1 bg-surface-1"
          expandedFieldClassName="w-64 border-subtle px-2.5 py-1.5"
          searchIconClassName="h-3.5 w-3.5"
        />
      </div>
      <ModuleOrderByDropdown
        value={displayFilters?.order_by}
        onChange={(val) => {
          if (!projectId || val === displayFilters?.order_by) return;
          update({
            display: {
              ...preferences.display,
              order_by: val,
            },
          });
        }}
      />
      <FiltersDropdown
        icon={<ListFilter className="h-3 w-3" />}
        title="Filters"
        placement="bottom-end"
        isFiltersApplied={isFiltersApplied}
      >
        <ModuleFiltersSelection
          displayFilters={displayFilters ?? {}}
          filters={filters ?? {}}
          handleDisplayFiltersUpdate={(val) => {
            if (!projectId) return;
            update({ display: { ...preferences.display, ...val } });
          }}
          handleFiltersUpdate={handleFilters}
          memberIds={workspaceMemberIds}
          workspaceSlug={workspaceSlug}
        />
      </FiltersDropdown>
      <div className="hidden items-center gap-1 rounded-sm bg-layer-3 p-1 md:flex">
        {MODULE_VIEW_LAYOUTS.map((layout) => (
          <Tooltip key={layout.key} tooltipContent={t(layout.i18n_title)} isMobile={isMobile}>
            <button
              type="button"
              className={cn(
                "group grid h-5.5 w-7 place-items-center overflow-hidden rounded-sm transition-all hover:bg-layer-transparent-hover",
                {
                  "bg-layer-transparent-active hover:bg-layer-transparent-active":
                    displayFilters?.layout === layout.key,
                }
              )}
              onClick={() => {
                if (!projectId) return;
                update({ display: { ...preferences.display, layout: layout.key } });
              }}
            >
              <ModuleLayoutIcon layoutType={layout.key} />
            </button>
          </Tooltip>
        ))}
      </div>
    </div>
  );
};

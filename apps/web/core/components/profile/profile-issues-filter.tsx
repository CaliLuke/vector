/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// i18n
import { useTranslation } from "@plane/i18n";
import { EIssueLayoutTypes } from "@plane/types";
// components
import { DisplayFiltersSelection, FiltersDropdown, LayoutSelection } from "@/components/issues/issue-layouts/filters";
import { WorkItemFiltersToggle } from "@/components/work-item-filters/filters-toggle";
// hooks
import { useProfileIssueDisplayPreferences } from "@/hooks/use-profile-issue-display-preferences";

export function ProfileIssuesFilter() {
  // i18n
  const { t } = useTranslation();
  const {
    activeLayout,
    handleDisplayFilters,
    handleDisplayProperties,
    handleLayoutChange,
    issueFilters,
    layoutDisplayFiltersOptions,
    userId,
  } = useProfileIssueDisplayPreferences();

  return (
    <div className="relative flex items-center justify-end gap-2">
      <LayoutSelection
        layouts={[EIssueLayoutTypes.LIST, EIssueLayoutTypes.KANBAN]}
        onChange={handleLayoutChange}
        selectedLayout={activeLayout}
      />
      {userId && <WorkItemFiltersToggle />}
      <FiltersDropdown title={t("common.display")} placement="bottom-end">
        <DisplayFiltersSelection
          layoutDisplayFiltersOptions={layoutDisplayFiltersOptions}
          displayFilters={issueFilters?.displayFilters ?? {}}
          handleDisplayFiltersUpdate={handleDisplayFilters}
          displayProperties={issueFilters?.displayProperties ?? {}}
          handleDisplayPropertiesUpdate={handleDisplayProperties}
        />
      </FiltersDropdown>
    </div>
  );
}

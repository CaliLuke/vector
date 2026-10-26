/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";

// plane imports
import { EIssueFilterType, ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import type { IIssueDisplayFilterOptions, IIssueDisplayProperties } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
import { EHeaderVariant, Header } from "@plane/ui";
// components
import { ArchiveTabsList } from "@/components/archives";
import { DisplayFiltersSelection, FiltersDropdown } from "@/components/issues/issue-layouts/filters";
import { WorkItemFiltersToggle } from "@/components/work-item-filters/filters-toggle";
// hooks
import { useLocalIssuePreferences } from "@/providers/local-issue-preferences";
import { useRouteProject } from "@/hooks/use-projects";

export function ArchivedIssuesHeader() {
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // store hooks
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;
  const preferences = useLocalIssuePreferences();
  if (!preferences || preferences.domain !== EIssuesStoreType.ARCHIVED)
    throw new Error("Archived issue preferences are not mounted for this route.");
  const { issueFilters, updateFilters } = preferences;
  // i18n
  const { t } = useTranslation();
  // for archived issues list layout is the only option
  const activeLayout = "list";

  const handleDisplayFiltersUpdate = (updatedDisplayFilter: Partial<IIssueDisplayFilterOptions>) => {
    if (!workspaceSlug || !projectId) return;

    void updateFilters(EIssueFilterType.DISPLAY_FILTERS, {
      ...issueFilters.displayFilters,
      ...updatedDisplayFilter,
    });
  };

  const handleDisplayPropertiesUpdate = (property: Partial<IIssueDisplayProperties>) => {
    if (!workspaceSlug || !projectId) return;

    void updateFilters(EIssueFilterType.DISPLAY_PROPERTIES, property);
  };

  if (!workspaceSlug || !projectId) return null;
  return (
    <Header variant={EHeaderVariant.SECONDARY}>
      <Header.LeftItem>
        <ArchiveTabsList />
      </Header.LeftItem>
      <Header.RightItem className="items-center">
        <WorkItemFiltersToggle />
        <FiltersDropdown title={t("common.display")} placement="bottom-end">
          <DisplayFiltersSelection
            displayFilters={issueFilters?.displayFilters || {}}
            displayProperties={issueFilters?.displayProperties || {}}
            handleDisplayFiltersUpdate={handleDisplayFiltersUpdate}
            handleDisplayPropertiesUpdate={handleDisplayPropertiesUpdate}
            layoutDisplayFiltersOptions={
              activeLayout ? ISSUE_DISPLAY_FILTERS_BY_PAGE.archived_issues.layoutOptions[activeLayout] : undefined
            }
            cycleViewDisabled={!currentProjectDetails?.cycle_view}
            moduleViewDisabled={!currentProjectDetails?.module_view}
          />
        </FiltersDropdown>
      </Header.RightItem>
    </Header>
  );
}

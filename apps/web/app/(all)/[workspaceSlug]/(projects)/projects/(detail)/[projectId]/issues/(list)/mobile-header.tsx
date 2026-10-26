/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { useState } from "react";

// plane imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { ChevronDownIcon } from "@plane/propel/icons";
import { EIssueLayoutTypes } from "@plane/types";
// components
import { WorkItemsModal } from "@/components/analytics/work-items/modal";
import {
  DisplayFiltersSelection,
  FiltersDropdown,
  MobileLayoutSelection,
} from "@/components/issues/issue-layouts/filters";
// hooks
import { useProjectIssueDisplayPreferences } from "@/hooks/use-project-issue-display-preferences";
import { useRouteProject } from "@/hooks/use-projects";

export const ProjectIssuesMobileHeader = function ProjectIssuesMobileHeader() {
  // i18n
  const { t } = useTranslation();
  const [analyticsModal, setAnalyticsModal] = useState(false);
  const { workspaceSlug, projectId } = useParams({
    from: "/_authenticated/$workspaceSlug/_projects/projects/$projectId/issues",
  });
  const routeProjectQuery = useRouteProject();
  const currentProjectDetails = routeProjectQuery.project;

  // store hooks
  const { issueFilters, activeLayout, handleLayoutChange, handleDisplayFilters, handleDisplayProperties } =
    useProjectIssueDisplayPreferences({ workspaceSlug, projectId });

  return (
    <>
      <WorkItemsModal
        workspaceSlug={workspaceSlug}
        isOpen={analyticsModal}
        onClose={() => setAnalyticsModal(false)}
        projectDetails={currentProjectDetails ?? undefined}
      />
      <div className="z-[13] flex justify-evenly border-b border-subtle bg-surface-1 py-2 md:hidden">
        <MobileLayoutSelection
          layouts={[EIssueLayoutTypes.LIST, EIssueLayoutTypes.KANBAN, EIssueLayoutTypes.CALENDAR]}
          onChange={handleLayoutChange}
        />
        <div className="flex flex-grow items-center justify-center border-l border-subtle text-13 text-secondary">
          <FiltersDropdown
            title={t("common.display")}
            placement="bottom-end"
            menuButton={
              <span className="flex items-center text-13 text-secondary">
                {t("common.display")}
                <ChevronDownIcon className="ml-2 h-4 w-4 text-secondary" />
              </span>
            }
          >
            <DisplayFiltersSelection
              layoutDisplayFiltersOptions={
                activeLayout ? ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.layoutOptions[activeLayout] : undefined
              }
              displayFilters={issueFilters?.displayFilters ?? {}}
              handleDisplayFiltersUpdate={handleDisplayFilters}
              displayProperties={issueFilters?.displayProperties ?? {}}
              handleDisplayPropertiesUpdate={handleDisplayProperties}
              cycleViewDisabled={!currentProjectDetails?.cycle_view}
              moduleViewDisabled={!currentProjectDetails?.module_view}
            />
          </FiltersDropdown>
        </div>

        <button
          onClick={() => setAnalyticsModal(true)}
          className="flex flex-grow justify-center border-l border-subtle text-13 text-secondary"
        >
          {t("common.analytics")}
        </button>
      </div>
    </>
  );
};

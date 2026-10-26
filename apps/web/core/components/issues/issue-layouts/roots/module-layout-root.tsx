/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

import { useParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
// plane imports
import { ISSUE_DISPLAY_FILTERS_BY_PAGE, PROJECT_VIEW_TRACKER_ELEMENTS } from "@plane/constants";
import { EIssuesStoreType, EIssueLayoutTypes } from "@plane/types";
import { Row, ERowVariant } from "@plane/ui";
// hooks
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { ModuleCalendarLayout } from "../calendar/roots/module-root";
import { BaseGanttRoot } from "../gantt";
import { ModuleKanBanLayout } from "../kanban/roots/module-root";
import { ModuleListLayout } from "../list/roots/module-root";
import { ModuleSpreadsheetLayout } from "../spreadsheet/roots/module-root";

function ModuleIssueLayout(props: { activeLayout: EIssueLayoutTypes | undefined; moduleId: string }) {
  switch (props.activeLayout) {
    case EIssueLayoutTypes.LIST:
      return <ModuleListLayout />;
    case EIssueLayoutTypes.KANBAN:
      return <ModuleKanBanLayout />;
    case EIssueLayoutTypes.CALENDAR:
      return <ModuleCalendarLayout />;
    case EIssueLayoutTypes.GANTT:
      return <BaseGanttRoot viewId={props.moduleId} />;
    case EIssueLayoutTypes.SPREADSHEET:
      return <ModuleSpreadsheetLayout />;
    default:
      return null;
  }
}

export function ModuleLayoutRoot() {
  // router
  const { workspaceSlug, projectId, moduleId } = useParams({ strict: false });
  // hooks
  const preferences = useParentIssuePreferences();
  // derived values
  const workItemFilters = preferences.filters;
  const activeLayout = workItemFilters?.displayFilters?.layout || undefined;

  if (preferences.isError || preferences.isAccessDenied)
    return <CatalogueReadFailure label="module preferences" retry={preferences.retry} />;
  if (preferences.isPending) return <output>Loading module preferences…</output>;
  if (!workspaceSlug || !projectId || !moduleId || !workItemFilters) return <></>;
  return (
    <>
      {preferences.isSaveError && (
        <CatalogueReadFailure label="saved module preferences" retry={preferences.retrySave} />
      )}
      <ProjectLevelWorkItemFiltersHOC
        enableSaveView
        entityType={EIssuesStoreType.MODULE}
        entityId={moduleId}
        filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.filters}
        initialWorkItemFilters={workItemFilters}
        updateFilters={preferences.setRich}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      >
        {({ editor }) => (
          <div className="relative flex h-full w-full flex-col overflow-hidden">
            {editor && (
              <WorkItemFiltersRow
                editor={editor}
                trackerElements={{
                  saveView: PROJECT_VIEW_TRACKER_ELEMENTS.MODULE_HEADER_SAVE_AS_VIEW_BUTTON,
                }}
              />
            )}
            <Row variant={ERowVariant.HUGGING} className="h-full w-full overflow-auto">
              <ModuleIssueLayout activeLayout={activeLayout} moduleId={moduleId} />
            </Row>
            {/* peek overview */}
            <IssuePeekOverview />
          </div>
        )}
      </ProjectLevelWorkItemFiltersHOC>
    </>
  );
}

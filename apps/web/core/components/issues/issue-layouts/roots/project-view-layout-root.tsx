/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React from "react";

// plane constants
import { ISSUE_DISPLAY_FILTERS_BY_PAGE, PROJECT_VIEW_TRACKER_ELEMENTS } from "@plane/constants";
import { EIssuesStoreType, EIssueLayoutTypes } from "@plane/types";
// hooks
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
import { useProjectView } from "@/hooks/use-project-views";
import {
  useProjectViewIssuePreferences,
  ProjectViewIssuePreferencesProvider,
} from "@/providers/project-view-issue-preferences";
import { useCurrentUser } from "@/hooks/use-current-user";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { ProjectViewCalendarLayout } from "../calendar/roots/project-view-root";
import { BaseGanttRoot } from "../gantt";
import { ProjectViewKanBanLayout } from "../kanban/roots/project-view-root";
import { ProjectViewListLayout } from "../list/roots/project-view-root";
import { ProjectViewSpreadsheetLayout } from "../spreadsheet/roots/project-view-root";

function ProjectViewIssueLayout(props: { activeLayout: EIssueLayoutTypes | undefined; viewId: string }) {
  switch (props.activeLayout) {
    case EIssueLayoutTypes.LIST:
      return <ProjectViewListLayout />;
    case EIssueLayoutTypes.KANBAN:
      return <ProjectViewKanBanLayout />;
    case EIssueLayoutTypes.CALENDAR:
      return <ProjectViewCalendarLayout />;
    case EIssueLayoutTypes.GANTT:
      return <BaseGanttRoot viewId={props.viewId} />;
    case EIssueLayoutTypes.SPREADSHEET:
      return <ProjectViewSpreadsheetLayout />;
    default:
      return null;
  }
}

function ProjectViewIssueContent({
  viewId,
  projectId,
  workspaceSlug,
  view,
}: {
  viewId: string;
  projectId: string;
  workspaceSlug: string;
  view: NonNullable<ReturnType<typeof useProjectView>["view"]>;
}) {
  const { issueFilters, updateRichFilters } = useProjectViewIssuePreferences();
  const activeLayout = issueFilters.displayFilters?.layout;
  const initialWorkItemFilters = {
    richFilters: issueFilters.richFilters ?? view.rich_filters,
    displayFilters: issueFilters.displayFilters,
    displayProperties: issueFilters.displayProperties,
  };
  return (
    <ProjectLevelWorkItemFiltersHOC
      enableSaveView
      saveViewOptions={{ label: "Save as" }}
      enableUpdateView
      entityId={viewId}
      entityType={EIssuesStoreType.PROJECT_VIEW}
      filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.filters}
      initialWorkItemFilters={initialWorkItemFilters}
      updateFilters={updateRichFilters}
      projectId={projectId}
      workspaceSlug={workspaceSlug}
    >
      {({ editor: projectViewWorkItemsFilter }) => (
        <div className="relative flex h-full w-full flex-col overflow-hidden">
          {projectViewWorkItemsFilter && (
            <WorkItemFiltersRow
              editor={projectViewWorkItemsFilter}
              trackerElements={{ saveView: PROJECT_VIEW_TRACKER_ELEMENTS.HEADER_SAVE_VIEW_BUTTON }}
            />
          )}
          <div className="relative h-full w-full overflow-auto">
            <ProjectViewIssueLayout activeLayout={activeLayout} viewId={viewId} />
          </div>
          <IssuePeekOverview />
        </div>
      )}
    </ProjectLevelWorkItemFiltersHOC>
  );
}

export function ProjectViewLayoutRoot() {
  // router
  const { workspaceSlug, projectId, viewId } = useParams({ strict: false });
  // hooks
  const { data: currentUser } = useCurrentUser();
  const viewQuery = useProjectView(workspaceSlug, projectId, viewId);
  // derived values
  const projectView = viewQuery.view;
  if (!workspaceSlug || !projectId || !viewId || !projectView || !currentUser?.id) return <></>;
  return (
    <ProjectViewIssuePreferencesProvider
      view={projectView}
      workspaceSlug={workspaceSlug}
      projectId={projectId}
      accountId={currentUser.id}
      isScopeCurrent={viewQuery.isScopeCurrent}
      isReadScopeCurrent={viewQuery.isReadScopeCurrent}
    >
      <ProjectViewIssueContent view={projectView} viewId={viewId} workspaceSlug={workspaceSlug} projectId={projectId} />
    </ProjectViewIssuePreferencesProvider>
  );
}

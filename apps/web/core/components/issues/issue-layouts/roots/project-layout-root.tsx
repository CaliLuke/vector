/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
// plane constants
import { ISSUE_DISPLAY_FILTERS_BY_PAGE, PROJECT_VIEW_TRACKER_ELEMENTS } from "@plane/constants";
import type { TWorkItemFilterExpression } from "@plane/types";
import { EIssueLayoutTypes, EIssuesStoreType } from "@plane/types";
import { Spinner } from "@plane/ui";
// components
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
// hooks
import { useProjectIssuePreferencesContext } from "@/hooks/use-project-issue-preferences-context";
import type { IProjectIssuePreferencesContext } from "@/hooks/use-project-issue-preferences-context";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { CalendarLayout } from "../calendar/roots/project-root";
import { BaseGanttRoot } from "../gantt";
import { KanBanLayout } from "../kanban/roots/project-root";
import { ListLayout } from "../list/roots/project-root";
import { ProjectSpreadsheetLayout } from "../spreadsheet/roots/project-root";

function ProjectIssueLayout(props: { activeLayout: EIssueLayoutTypes | undefined }) {
  switch (props.activeLayout) {
    case EIssueLayoutTypes.LIST:
      return <ListLayout />;
    case EIssueLayoutTypes.KANBAN:
      return <KanBanLayout />;
    case EIssueLayoutTypes.CALENDAR:
      return <CalendarLayout />;
    case EIssueLayoutTypes.GANTT:
      return <BaseGanttRoot />;
    case EIssueLayoutTypes.SPREADSHEET:
      return <ProjectSpreadsheetLayout />;
    default:
      return null;
  }
}

function updateProjectRichFilters(
  projectPreferences: IProjectIssuePreferencesContext | undefined,
  expression: TWorkItemFilterExpression
) {
  if (!projectPreferences) return Promise.reject(new Error("Project preferences are unavailable."));
  return projectPreferences.save({ rich_filters: expression });
}

export function ProjectLayoutRoot() {
  // router
  const { workspaceSlug, projectId } = useParams({ strict: false });
  // hooks
  const projectPreferences = useProjectIssuePreferencesContext();
  // derived values
  const nativeFilters = projectPreferences?.issueFilters;
  const workItemFilters = nativeFilters;
  const activeLayout = nativeFilters?.displayFilters.layout;
  const updateRichFilters = async (expression: TWorkItemFilterExpression) => {
    await updateProjectRichFilters(projectPreferences, expression);
  };

  if (!workspaceSlug || !projectId) return null;
  if (!projectPreferences?.properties && projectPreferences?.isError) {
    return (
      <div role="alert" className="flex items-center gap-3">
        <p>Unable to load project preferences.</p>
        <button disabled={projectPreferences.isPending} onClick={() => void projectPreferences.refetch()}>
          Retry
        </button>
      </div>
    );
  }
  if (!workItemFilters || !projectPreferences?.requestInput) return <Spinner />;
  return (
    <ProjectLevelWorkItemFiltersHOC
      enableSaveView
      entityType={EIssuesStoreType.PROJECT}
      entityId={projectId}
      filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.filters}
      initialWorkItemFilters={workItemFilters}
      updateFilters={updateRichFilters}
      projectId={projectId}
      workspaceSlug={workspaceSlug}
    >
      {({ editor: projectWorkItemsFilter }) => (
        <div className="relative flex h-full w-full flex-col overflow-hidden">
          {projectWorkItemsFilter && (
            <WorkItemFiltersRow
              editor={projectWorkItemsFilter}
              trackerElements={{
                saveView: PROJECT_VIEW_TRACKER_ELEMENTS.PROJECT_HEADER_SAVE_AS_VIEW_BUTTON,
              }}
            />
          )}
          <div className="relative h-full w-full overflow-auto bg-surface-1">
            <ProjectIssueLayout activeLayout={activeLayout} />
          </div>
          {/* peek overview */}
          <IssuePeekOverview />
        </div>
      )}
    </ProjectLevelWorkItemFiltersHOC>
  );
}

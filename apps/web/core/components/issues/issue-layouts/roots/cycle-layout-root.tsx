/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import React, { useState } from "react";
import isEmpty from "lodash-es/isEmpty.js";

import { useParentIssuePreferences } from "@/providers/parent-issue-preferences";
import { CatalogueReadFailure } from "@/components/common/catalogue-read-failure";
// plane constants
import { ISSUE_DISPLAY_FILTERS_BY_PAGE, PROJECT_VIEW_TRACKER_ELEMENTS } from "@plane/constants";
import { EIssuesStoreType, EIssueLayoutTypes } from "@plane/types";
// components
import { TransferIssues } from "@/components/cycles/transfer-issues";
import { TransferIssuesModal } from "@/components/cycles/transfer-issues-modal";
// hooks
import { ProjectLevelWorkItemFiltersHOC } from "@/components/work-item-filters/filters-hoc/project-level";
import { WorkItemFiltersRow } from "@/components/work-item-filters/filters-row";
import { useCycleDetails } from "@/hooks/use-cycles";
// local imports
import { IssuePeekOverview } from "../../peek-overview";
import { CycleCalendarLayout } from "../calendar/roots/cycle-root";
import { BaseGanttRoot } from "../gantt";
import { CycleKanBanLayout } from "../kanban/roots/cycle-root";
import { CycleListLayout } from "../list/roots/cycle-root";
import { CycleSpreadsheetLayout } from "../spreadsheet/roots/cycle-root";

function CycleIssueLayout(props: {
  activeLayout: EIssueLayoutTypes | undefined;
  cycleId: string;
  isCompletedCycle: boolean;
}) {
  switch (props.activeLayout) {
    case EIssueLayoutTypes.LIST:
      return <CycleListLayout />;
    case EIssueLayoutTypes.KANBAN:
      return <CycleKanBanLayout />;
    case EIssueLayoutTypes.CALENDAR:
      return <CycleCalendarLayout />;
    case EIssueLayoutTypes.GANTT:
      return <BaseGanttRoot viewId={props.cycleId} isCompletedCycle={props.isCompletedCycle} />;
    case EIssueLayoutTypes.SPREADSHEET:
      return <CycleSpreadsheetLayout />;
    default:
      return null;
  }
}

export function CycleLayoutRoot() {
  const { workspaceSlug, projectId, cycleId } = useParams({ strict: false });
  // store hooks
  const preferences = useParentIssuePreferences();
  const { cycle: cycleDetails } = useCycleDetails(workspaceSlug, projectId, cycleId);
  // state
  const [transferIssuesModal, setTransferIssuesModal] = useState(false);
  // derived values
  const workItemFilters = preferences.filters;
  const activeLayout = workItemFilters?.displayFilters?.layout;

  const cycleStatus = cycleDetails?.status?.toLocaleLowerCase();
  const isCompletedCycle = cycleStatus === "completed";
  const isProgressSnapshotEmpty = isEmpty(cycleDetails?.progress_snapshot);
  const transferableIssuesCount = cycleDetails
    ? cycleDetails.backlog_issues + cycleDetails.unstarted_issues + cycleDetails.started_issues
    : 0;
  const canTransferIssues = isProgressSnapshotEmpty && transferableIssuesCount > 0;

  if (preferences.isError || preferences.isAccessDenied)
    return <CatalogueReadFailure label="cycle preferences" retry={preferences.retry} />;
  if (preferences.isPending) return <output>Loading cycle preferences…</output>;
  if (!workspaceSlug || !projectId || !cycleId || !workItemFilters) return <></>;
  return (
    <>
      {preferences.isSaveError && (
        <CatalogueReadFailure label="saved cycle preferences" retry={preferences.retrySave} />
      )}
      <ProjectLevelWorkItemFiltersHOC
        enableSaveView
        entityType={EIssuesStoreType.CYCLE}
        entityId={cycleId}
        filtersToShowByLayout={ISSUE_DISPLAY_FILTERS_BY_PAGE.issues.filters}
        initialWorkItemFilters={workItemFilters}
        updateFilters={preferences.setRich}
        projectId={projectId}
        workspaceSlug={workspaceSlug}
      >
        {({ editor }) => (
          <>
            <TransferIssuesModal
              handleClose={() => setTransferIssuesModal(false)}
              cycleId={cycleId}
              isOpen={transferIssuesModal}
              projectId={projectId}
              workspaceSlug={workspaceSlug}
            />
            <div className="relative flex h-full w-full flex-col overflow-hidden">
              {cycleStatus === "completed" && (
                <TransferIssues
                  handleClick={() => setTransferIssuesModal(true)}
                  canTransferIssues={canTransferIssues}
                  disabled={!isProgressSnapshotEmpty}
                />
              )}
              {editor && (
                <WorkItemFiltersRow
                  editor={editor}
                  trackerElements={{
                    saveView: PROJECT_VIEW_TRACKER_ELEMENTS.CYCLE_HEADER_SAVE_AS_VIEW_BUTTON,
                  }}
                />
              )}
              <div className="h-full w-full overflow-auto">
                <CycleIssueLayout activeLayout={activeLayout} cycleId={cycleId} isCompletedCycle={isCompletedCycle} />
              </div>
              {/* peek overview */}
              <IssuePeekOverview />
            </div>
          </>
        )}
      </ProjectLevelWorkItemFiltersHOC>
    </>
  );
}

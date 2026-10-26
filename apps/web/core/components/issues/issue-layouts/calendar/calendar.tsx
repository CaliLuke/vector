/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { autoScrollForElements } from "@atlaskit/pragmatic-drag-and-drop-auto-scroll/element";
// plane constants
// types
import type { IIssueDisplayFilterOptions, TIssue, TIssueMap, TPaginationData, ICalendarPayload } from "@plane/types";
import { EIssueLayoutTypes } from "@plane/types";
// ui
import { Spinner } from "@plane/ui";
import { renderFormattedPayloadDate } from "@plane/utils/datetime";
import { cn } from "@plane/utils/common";
// constants
import { MONTHS_LIST } from "@plane/constants";
// helpers
// hooks
import useSize from "@/hooks/use-window-size";
import type { TCalendarDates } from "@/providers/issue-layout-ui-state";
import type { getCalendarState } from "./calendar-state";
// local imports
import { IssueLayoutHOC } from "../issue-layout-HOC";
import type { TRenderQuickActions } from "../list/list-view-types";
import { CalendarHeader } from "./header";
import { CalendarIssueBlocks } from "./issue-blocks";
import { CalendarWeekDays } from "./week-days";
import { CalendarWeekHeader } from "./week-header";
import type { TCalendarDisplayFilterUpdate } from "@/hooks/use-calendar-display-filter-update";
import { getGroupedIssueIds } from "../issue-display-ordering";
import type { IssueListGroupedIds } from "@/queries/issue-lists";

type Props = {
  displayFilters: IIssueDisplayFilterOptions;
  issues: TIssueMap | undefined;
  groupedIssueIds: IssueListGroupedIds;
  layout: "month" | "week" | undefined;
  showWeekends: boolean;
  calendarState: ReturnType<typeof getCalendarState>;
  activeMonthDate: Date;
  activeWeekDate: Date;
  updateCalendarDates: (filters: Partial<TCalendarDates>) => void;
  loadMoreIssues: (dateString: string) => void;
  getPaginationData: (groupId: string | undefined) => TPaginationData | undefined;
  getGroupIssueCount: (groupId: string | undefined) => number | undefined;
  quickAddCallback?: (projectId: string | null | undefined, data: TIssue) => Promise<TIssue | undefined>;
  quickActions: TRenderQuickActions;
  handleDragAndDrop: (
    issueId: string | undefined,
    issueProjectId: string | undefined,
    sourceDate: string | undefined,
    destinationDate: string | undefined
  ) => Promise<void>;
  addIssuesToView?: (issueIds: string[]) => Promise<unknown>;
  readOnly?: boolean;
  updateFilters: TCalendarDisplayFilterUpdate;
  canEditProperties: (projectId: string | undefined) => boolean;
  isEpic?: boolean;
  readState: {
    isPending: boolean;
    isError: boolean;
    issueCount: number;
    retry: () => Promise<void>;
  };
};

export function CalendarChart(props: Props) {
  const {
    displayFilters,
    issues,
    groupedIssueIds,
    layout,
    showWeekends,
    calendarState,
    activeMonthDate,
    activeWeekDate,
    updateCalendarDates,
    loadMoreIssues,
    handleDragAndDrop,
    quickActions,
    quickAddCallback,
    addIssuesToView,
    getPaginationData,
    getGroupIssueCount,
    updateFilters,
    canEditProperties,
    readOnly = false,
    isEpic = false,
    readState,
  } = props;
  // states
  const [selectedDate, setSelectedDate] = useState<Date>(() => new Date());
  //refs
  const scrollableContainerRef = useRef<HTMLDivElement | null>(null);
  const [windowWidth] = useSize();

  const enableIssueCreation = true;
  const enableQuickAdd = true;

  const calendarPayload: ICalendarPayload = calendarState.calendarPayload;

  const allWeeksOfActiveMonth = calendarState.allWeeksOfActiveMonth;

  const formattedDatePayload = renderFormattedPayloadDate(selectedDate) ?? undefined;

  // Enable Auto Scroll for calendar
  useEffect(() => {
    const element = scrollableContainerRef.current;

    if (!element) return undefined;

    return combine(
      autoScrollForElements({
        element,
      })
    );
  }, []);

  if (!calendarPayload || !formattedDatePayload)
    return (
      <div className="grid h-full w-full place-items-center">
        <Spinner />
      </div>
    );

  const weekProps = {
    selectedDate,
    setSelectedDate,
    handleDragAndDrop,
    displayFilters,
    issues,
    groupedIssueIds,
    loadMoreIssues,
    getPaginationData,
    getGroupIssueCount,
    quickActions,
    quickAddCallback,
    addIssuesToView,
    readOnly,
    canEditProperties,
    isEpic,
    enableQuickIssueCreate: enableQuickAdd,
    disableIssueCreation: !enableIssueCreation,
  };
  const issueIdList = getGroupedIssueIds(groupedIssueIds, formattedDatePayload) ?? [];

  return (
    <>
      <div className="flex h-full w-full flex-col overflow-hidden">
        <CalendarHeader
          setSelectedDate={setSelectedDate}
          displayFilters={displayFilters}
          updateFilters={updateFilters}
          activeMonthDate={activeMonthDate}
          activeWeekDate={activeWeekDate}
          activeWeek={calendarState.activeWeek}
          updateCalendarDates={updateCalendarDates}
        />

        <IssueLayoutHOC layout={EIssueLayoutTypes.CALENDAR} readState={readState}>
          <div
            className={cn("flex w-full flex-col overflow-y-auto md:h-full", {
              "vertical-scrollbar scrollbar-lg": windowWidth > 768,
            })}
            ref={scrollableContainerRef}
          >
            <CalendarWeekHeader isLoading={!issues} showWeekends={showWeekends} />
            <div className="h-full w-full">
              {layout === "month" && (
                <div className="grid h-full w-full grid-cols-1 divide-y-[0.5px] divide-subtle-1">
                  {allWeeksOfActiveMonth &&
                    Object.entries(allWeeksOfActiveMonth).map(([weekKey, week]) => (
                      <CalendarWeekDays {...weekProps} key={weekKey} week={week} />
                    ))}
                </div>
              )}
              {layout === "week" && <CalendarWeekDays {...weekProps} week={calendarState.activeWeek} />}
            </div>

            {/* mobile view */}
            <div className="md:hidden">
              <p className="p-4 text-18 font-semibold">
                {`${selectedDate.getDate()} ${
                  MONTHS_LIST[selectedDate.getMonth() + 1].title
                }, ${selectedDate.getFullYear()}`}
              </p>
              <CalendarIssueBlocks
                date={selectedDate}
                issueIdList={issueIdList}
                loadMoreIssues={loadMoreIssues}
                getPaginationData={getPaginationData}
                getGroupIssueCount={getGroupIssueCount}
                quickActions={quickActions}
                enableQuickIssueCreate={enableQuickAdd}
                disableIssueCreation={!enableIssueCreation}
                quickAddCallback={quickAddCallback}
                addIssuesToView={addIssuesToView}
                readOnly={readOnly}
                canEditProperties={canEditProperties}
                isDragDisabled
                isMobileView
                isEpic={isEpic}
              />
            </div>
          </div>
        </IssueLayoutHOC>

        {/* mobile view */}
        <div className="md:hidden">
          <p className="p-4 text-18 font-semibold">
            {`${selectedDate.getDate()} ${
              MONTHS_LIST[selectedDate.getMonth() + 1].title
            }, ${selectedDate.getFullYear()}`}
          </p>
          <CalendarIssueBlocks
            date={selectedDate}
            issueIdList={issueIdList}
            quickActions={quickActions}
            loadMoreIssues={loadMoreIssues}
            getPaginationData={getPaginationData}
            getGroupIssueCount={getGroupIssueCount}
            enableQuickIssueCreate={enableQuickAdd}
            disableIssueCreation={!enableIssueCreation}
            quickAddCallback={quickAddCallback}
            addIssuesToView={addIssuesToView}
            readOnly={readOnly}
            canEditProperties={canEditProperties}
            isDragDisabled
            isMobileView
            isEpic={isEpic}
          />
        </div>
      </div>
    </>
  );
}

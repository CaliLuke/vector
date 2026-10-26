/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { differenceInCalendarDays } from "date-fns/differenceInCalendarDays";

import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { ICalendarDate, TIssue } from "@plane/types";
// types
// ui
// components
import { cn } from "@plane/utils/common";
import { renderFormattedPayloadDate } from "@plane/utils/datetime";
import { highlightIssueOnDrop } from "@/components/issues/issue-layouts/utils";
import { getGroupedIssueIds } from "../issue-display-ordering";
// helpers
import { MONTHS_LIST } from "@plane/constants";
// helpers
// types
import { CalendarIssueBlocks } from "./issue-blocks";
import type { CalendarDayContentProps } from "./types";

type Props = CalendarDayContentProps & {
  date: ICalendarDate;
};

function isCalendarDateBeforeStart(startDate: string | null | undefined, targetDate: string | null | undefined) {
  if (!startDate || !targetDate) return false;
  return differenceInCalendarDays(new Date(targetDate), new Date(startDate)) < 0;
}

function handleCalendarIssueDrop(
  issueId: string,
  sourceDate: string,
  destinationDate: string,
  issue: TIssue | undefined,
  sourceElementId: string | undefined,
  handleDragAndDrop: CalendarDayContentProps["handleDragAndDrop"]
) {
  if (isCalendarDateBeforeStart(issue?.start_date, destinationDate)) {
    setToast({
      type: TOAST_TYPE.ERROR,
      title: "Error!",
      message: "Due date cannot be before the start date of the work item.",
    });
    return;
  }

  void handleDragAndDrop(issueId, issue?.project_id ?? undefined, sourceDate, destinationDate);
  highlightIssueOnDrop(sourceElementId, false);
}

function CalendarDayHeader({
  date,
  isToday,
  isWeekend,
  isMonthLayout,
}: {
  date: ICalendarDate;
  isToday: boolean;
  isWeekend: boolean;
  isMonthLayout: boolean;
}) {
  return (
    <div
      className={`hidden flex-shrink-0 justify-end px-2 py-1.5 text-right text-11 md:flex ${
        isMonthLayout ? (date.is_current_month ? "font-medium" : "text-tertiary") : "font-medium"
      } ${isWeekend ? "bg-layer-1" : "bg-layer-transparent"} `}
    >
      {date.date.getDate() === 1 && MONTHS_LIST[date.date.getMonth() + 1].shortTitle + " "}
      {isToday ? (
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-accent-primary text-on-color">
          {date.date.getDate()}
        </span>
      ) : (
        <>{date.date.getDate()}</>
      )}
    </div>
  );
}

export const CalendarDayTile = function CalendarDayTile(props: Props) {
  const {
    displayFilters,
    date,
    issues,
    groupedIssueIds,
    loadMoreIssues,
    getPaginationData,
    getGroupIssueCount,
    quickActions,
    enableQuickIssueCreate,
    disableIssueCreation,
    quickAddCallback,
    addIssuesToView,
    readOnly = false,
    selectedDate,
    handleDragAndDrop,
    setSelectedDate,
    canEditProperties,
    isEpic = false,
  } = props;

  const [isDraggingOver, setIsDraggingOver] = useState(false);

  const calendarLayout = displayFilters.calendar?.layout ?? "month";

  const formattedDatePayload = renderFormattedPayloadDate(date.date);

  const dayTileRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const element = dayTileRef.current;

    if (!element) return;

    return combine(
      dropTargetForElements({
        element,
        getData: () => ({ date: formattedDatePayload }),
        onDragEnter: () => {
          setIsDraggingOver(true);
        },
        onDragLeave: () => {
          setIsDraggingOver(false);
        },
        onDrop: ({ source, self }) => {
          setIsDraggingOver(false);
          const sourceData = source?.data as { id: string; date: string } | undefined;
          const destinationData = self?.data as { date: string } | undefined;
          if (!sourceData || !destinationData) return;

          handleCalendarIssueDrop(
            sourceData.id,
            sourceData.date,
            destinationData.date,
            issues?.[sourceData.id],
            source?.element?.id,
            handleDragAndDrop
          );
        },
      })
    );
  }, [formattedDatePayload, handleDragAndDrop, issues]);

  if (!formattedDatePayload) return null;
  const issueIds = getGroupedIssueIds(groupedIssueIds, formattedDatePayload) ?? [];

  const isToday = date.is_today;
  const isSelectedDate = date.date.toDateString() == selectedDate.toDateString();

  const isWeekend = [0, 6].includes(date.date.getDay());
  const isMonthLayout = calendarLayout === "month";

  const normalBackground = isWeekend ? "bg-layer-1" : "bg-layer-transparent";
  const draggingOverBackground = isWeekend ? "bg-layer-1" : "bg-layer-transparent-hover";

  return (
    <>
      <div ref={dayTileRef} className="group relative flex h-full w-full flex-col">
        {/* header */}
        <CalendarDayHeader date={date} isToday={isToday} isWeekend={isWeekend} isMonthLayout={isMonthLayout} />

        {/* content */}
        <div className="hidden h-full w-full md:block">
          <div
            className={cn(
              `h-full w-full select-none ${isDraggingOver ? `${draggingOverBackground} opacity-70` : normalBackground}`,
              {
                "min-h-[5rem]": isMonthLayout,
              }
            )}
          >
            <CalendarIssueBlocks
              date={date.date}
              issueIdList={issueIds}
              quickActions={quickActions}
              loadMoreIssues={loadMoreIssues}
              getPaginationData={getPaginationData}
              getGroupIssueCount={getGroupIssueCount}
              isDragDisabled={readOnly}
              addIssuesToView={addIssuesToView}
              disableIssueCreation={disableIssueCreation}
              enableQuickIssueCreate={enableQuickIssueCreate}
              quickAddCallback={quickAddCallback}
              readOnly={readOnly}
              canEditProperties={canEditProperties}
              isEpic={isEpic}
            />
          </div>
        </div>

        {/* Mobile view content */}
        <button
          type="button"
          onClick={() => setSelectedDate(date.date)}
          className={cn(
            "mx-auto flex h-full w-full cursor-pointer flex-col items-center justify-start py-2.5 text-13 font-medium opacity-80 md:hidden",
            {
              "bg-layer-2": !isWeekend,
            }
          )}
        >
          <div
            className={cn("flex size-6 items-center justify-center rounded-full", {
              "bg-accent-primary text-on-color": isSelectedDate,
              "bg-accent-primary/10 text-accent-primary": isToday && !isSelectedDate,
            })}
          >
            {date.date.getDate()}
          </div>
        </button>
      </div>
    </>
  );
};

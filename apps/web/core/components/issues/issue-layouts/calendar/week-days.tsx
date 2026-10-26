/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// plane imports
import type { ICalendarDate, ICalendarWeek } from "@plane/types";
import { cn } from "@plane/utils/common";
import { getOrderedDays } from "@plane/utils/calendar";
import { renderFormattedPayloadDate } from "@plane/utils/datetime";
// hooks
import { useCurrentUserProfile } from "@/hooks/use-current-user-profile";
// types
import { CalendarDayTile } from "./day-tile";
import type { CalendarDayContentProps } from "./types";

type Props = CalendarDayContentProps & {
  week: ICalendarWeek | undefined;
};

export const CalendarWeekDays = function CalendarWeekDays(props: Props) {
  const {
    displayFilters,
    issues,
    groupedIssueIds,
    handleDragAndDrop,
    week,
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
    setSelectedDate,
    canEditProperties,
    isEpic = false,
  } = props;
  // hooks
  const { data } = useCurrentUserProfile();
  const startOfWeek = data?.start_of_the_week;

  const calendarLayout = displayFilters.calendar?.layout ?? "month";
  const showWeekends = displayFilters.calendar?.show_weekends ?? false;

  if (!week) return null;

  const shouldShowDay = (dayDate: Date) => {
    if (showWeekends) return true;
    const day = dayDate.getDay();
    return !(day === 0 || day === 6);
  };

  const sortedWeekDays = getOrderedDays(Object.values(week), (item) => item.date.getDay(), startOfWeek);

  return (
    <div
      className={cn("grid divide-subtle-1 md:divide-x-[0.5px]", {
        "grid-cols-7": showWeekends,
        "grid-cols-5": !showWeekends,
        "h-full": calendarLayout !== "month",
      })}
    >
      {sortedWeekDays.map((date: ICalendarDate) => {
        if (!shouldShowDay(date.date)) return null;

        return (
          <CalendarDayTile
            selectedDate={selectedDate}
            setSelectedDate={setSelectedDate}
            displayFilters={displayFilters}
            key={renderFormattedPayloadDate(date.date)}
            date={date}
            issues={issues}
            groupedIssueIds={groupedIssueIds}
            loadMoreIssues={loadMoreIssues}
            getPaginationData={getPaginationData}
            getGroupIssueCount={getGroupIssueCount}
            quickActions={quickActions}
            enableQuickIssueCreate={enableQuickIssueCreate}
            disableIssueCreation={disableIssueCreation}
            quickAddCallback={quickAddCallback}
            addIssuesToView={addIssuesToView}
            readOnly={readOnly}
            handleDragAndDrop={handleDragAndDrop}
            canEditProperties={canEditProperties}
            isEpic={isEpic}
          />
        );
      })}
    </div>
  );
};

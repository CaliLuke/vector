/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ICalendarPayload, ICalendarWeek } from "@plane/types";
import { generateCalendarData } from "@plane/utils/calendar";
import sortBy from "lodash-es/sortBy.js";

export function getCalendarState(activeMonthDate: Date, activeWeekDate: Date, startOfWeek: number) {
  let calendarPayload: ICalendarPayload = generateCalendarData(null, activeMonthDate, startOfWeek);
  if (
    activeWeekDate.getFullYear() !== activeMonthDate.getFullYear() ||
    activeWeekDate.getMonth() !== activeMonthDate.getMonth()
  ) {
    calendarPayload = generateCalendarData(calendarPayload, activeWeekDate, startOfWeek);
  }

  const monthWeeks = calendarPayload[`y-${activeMonthDate.getFullYear()}`]?.[`m-${activeMonthDate.getMonth()}`];
  const allWeeksOfActiveMonth = monthWeeks
    ? Object.fromEntries(sortBy(Object.entries(monthWeeks), ([week]) => Number(week.slice(2))))
    : undefined;

  const activeWeekMonth = calendarPayload[`y-${activeWeekDate.getFullYear()}`]?.[`m-${activeWeekDate.getMonth()}`];
  const firstDayOfMonth =
    (new Date(activeWeekDate.getFullYear(), activeWeekDate.getMonth(), 1).getDay() - startOfWeek + 7) % 7;
  const weekIndex = Math.floor((activeWeekDate.getDate() - 1 + firstDayOfMonth) / 7);
  const activeWeek: ICalendarWeek | undefined = activeWeekMonth?.[`w-${weekIndex}`];

  const activeWeekDates = activeWeek ? Object.keys(activeWeek) : [];
  const dateRange =
    activeWeekDates.length > 0
      ? { startDate: activeWeekDates[0], endDate: activeWeekDates[activeWeekDates.length - 1] }
      : undefined;

  const monthWeekKeys = allWeeksOfActiveMonth ? Object.keys(allWeeksOfActiveMonth) : [];
  const firstMonthWeek = allWeeksOfActiveMonth?.[monthWeekKeys[0]];
  const lastMonthWeek = allWeeksOfActiveMonth?.[monthWeekKeys[monthWeekKeys.length - 1]];
  const lastMonthWeekDates = lastMonthWeek ? Object.keys(lastMonthWeek) : [];
  const monthDateRange =
    firstMonthWeek && lastMonthWeek
      ? {
          startDate: Object.keys(firstMonthWeek)[0],
          endDate: lastMonthWeekDates[lastMonthWeekDates.length - 1],
        }
      : undefined;

  return {
    calendarPayload,
    allWeeksOfActiveMonth,
    activeWeek,
    monthDateRange,
    weekDateRange: dateRange,
  };
}

export function shiftCalendarDate(date: Date, amount: number, layout: "month" | "week") {
  if (layout === "month") {
    return new Date(date.getFullYear(), date.getMonth() + amount, 1);
  }
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + amount * 7);
}

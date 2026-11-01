/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

//
import { generateChartSegment } from "./chart-segment";
import type { ChartDataType } from "@plane/types";
import { EStartOfTheWeek } from "@plane/types";
import { months, generateWeeks } from "../data";
import { getWeekNumberByDate } from "./helpers";
interface IDayBlock {
  date: Date;
  day: number;
  dayData: {
    key: number;
    shortTitle: string;
    title: string;
    abbreviation: string;
  };
  title: string;
  today: boolean;
}

export interface IWeekBlock {
  children?: IDayBlock[];
  weekNumber: number;
  weekData: {
    shortTitle: string;
    title: string;
  };
  title: string;
  startDate: Date;
  endDate: Date;
  startMonth: number;
  startYear: number;
  endMonth: number;
  endYear: number;
  today: boolean;
}

/**
 * Generate Week Chart data
 * @param weekPayload
 * @param side
 * @returns
 */
const generateWeekChart = (
  chart: ChartDataType,
  side: null | "left" | "right",
  targetDate?: Date,
  startOfWeek: EStartOfTheWeek = EStartOfTheWeek.SUNDAY
) =>
  generateChartSegment(chart, side, targetDate, (start, end) => {
    const payload = getWeeksBetweenTwoDates(start, end, true, startOfWeek);
    return { payload, startDate: payload[0].startDate, endDate: payload[payload.length - 1].endDate };
  });

/**
 * Generate weeks array between two dates
 * @param startDate
 * @param endDate
 * @param shouldPopulateDaysForWeek
 * @returns
 */
export const getWeeksBetweenTwoDates = (
  startDate: Date,
  endDate: Date,
  shouldPopulateDaysForWeek: boolean = true,
  startOfWeek: EStartOfTheWeek = EStartOfTheWeek.SUNDAY
): IWeekBlock[] => {
  const weeks: IWeekBlock[] = [];

  let currentDate = new Date(startDate.getTime());
  const today = new Date();

  // Adjust the current date to the start of the week
  const day = currentDate.getDay();
  const diff = (day + 7 - startOfWeek) % 7; // Calculate days to subtract to get to startOfWeek
  currentDate.setDate(currentDate.getDate() - diff);

  while (currentDate <= endDate) {
    const weekStartDate = new Date(currentDate.getTime());
    const weekEndDate = new Date(currentDate.getTime() + 6 * 24 * 60 * 60 * 1000);

    const monthAtStartOfTheWeek = weekStartDate.getMonth();
    const yearAtStartOfTheWeek = weekStartDate.getFullYear();
    const monthAtEndOfTheWeek = weekEndDate.getMonth();
    const yearAtEndOfTheWeek = weekEndDate.getFullYear();

    const weekNumber = getWeekNumberByDate(currentDate);

    weeks.push({
      children: shouldPopulateDaysForWeek ? populateDaysForWeek(weekStartDate, startOfWeek) : undefined,
      weekNumber,
      weekData: {
        shortTitle: `w${weekNumber}`,
        title: `Week ${weekNumber}`,
      },
      title:
        monthAtStartOfTheWeek === monthAtEndOfTheWeek
          ? `${months[monthAtStartOfTheWeek].abbreviation} ${yearAtStartOfTheWeek}`
          : `${months[monthAtStartOfTheWeek].abbreviation} ${yearAtStartOfTheWeek} - ${months[monthAtEndOfTheWeek].abbreviation} ${yearAtEndOfTheWeek}`,
      startMonth: monthAtStartOfTheWeek,
      startYear: yearAtStartOfTheWeek,
      endMonth: monthAtEndOfTheWeek,
      endYear: yearAtEndOfTheWeek,
      startDate: weekStartDate,
      endDate: weekEndDate,
      today: today >= weekStartDate && today <= weekEndDate,
    });

    currentDate = new Date(currentDate.setDate(currentDate.getDate() + 7));
  }

  return weeks;
};

/**
 * return back array of 7 days from the date provided
 * @param startDate
 * @returns
 */
const populateDaysForWeek = (startDate: Date, startOfWeek: EStartOfTheWeek = EStartOfTheWeek.SUNDAY): IDayBlock[] => {
  const currentDate = new Date(startDate);
  const days: IDayBlock[] = [];
  const today = new Date();
  const weekDays = generateWeeks(startOfWeek);

  for (let i = 0; i < 7; i++) {
    days.push({
      date: new Date(currentDate),
      day: currentDate.getDay(),
      dayData: weekDays[i],
      title: `${weekDays[i].abbreviation} ${currentDate.getDate()}`,
      today: today.setHours(0, 0, 0, 0) == currentDate.setHours(0, 0, 0, 0),
    });
    currentDate.setDate(currentDate.getDate() + 1);
  }
  return days;
};

/**
 * Merge two Week data payloads
 * @param a
 * @param b
 * @returns
 */
const mergeWeekRenderPayloads = (a: IWeekBlock[], b: IWeekBlock[]) => [...a, ...b];

export const weekView = {
  generateChart: generateWeekChart,
  mergeRenderPayloads: mergeWeekRenderPayloads,
};

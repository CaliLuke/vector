/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { generateChartSegment } from "./chart-segment";
import cloneDeep from "lodash-es/cloneDeep.js";
import uniqBy from "lodash-es/uniqBy.js";
// plane imports
import type { ChartDataType } from "@plane/types";
// local imports
import { months } from "../data";
import { getNumberOfDaysInMonth } from "./helpers";
import type { IWeekBlock } from "./week-view";
import { getWeeksBetweenTwoDates } from "./week-view";

export interface IMonthBlock {
  today: boolean;
  month: number;
  days: number;
  monthData: {
    key: number;
    shortTitle: string;
    title: string;
  };
  title: string;
  year: number;
}

export interface IMonthView {
  months: IMonthBlock[];
  weeks: IWeekBlock[];
}

/**
 * Generate Month Chart data
 * @param monthPayload
 * @param side
 * @returns
 */
const generateMonthChart = (chart: ChartDataType, side: null | "left" | "right", targetDate?: Date) =>
  generateChartSegment(cloneDeep(chart), side, targetDate, (start, end) => {
    const payload = getMonthsViewBetweenTwoDates(start, end);
    return {
      payload,
      startDate: payload.weeks[0]?.startDate,
      endDate: payload.weeks[payload.weeks.length - 1]?.endDate,
    };
  });

/**
 * Get Month View data between two dates, i.e., Months and Weeks between two dates
 * @param startDate
 * @param endDate
 * @returns
 */
const getMonthsViewBetweenTwoDates = (startDate: Date, endDate: Date): IMonthView => ({
  months: getMonthsBetweenTwoDates(startDate, endDate),
  weeks: getWeeksBetweenTwoDates(startDate, endDate, false),
});

/**
 * generate array of months between two dates
 * @param startDate
 * @param endDate
 * @returns
 */
export const getMonthsBetweenTwoDates = (startDate: Date, endDate: Date): IMonthBlock[] => {
  const monthBlocks = [];

  const startYear = startDate.getFullYear();
  const startMonth = startDate.getMonth();

  const today = new Date();
  const todayMonth = today.getMonth();
  const todayYear = today.getFullYear();

  let currentDate = new Date(startYear, startMonth);

  while (currentDate <= endDate) {
    const currentYear = currentDate.getFullYear();
    const currentMonth = currentDate.getMonth();

    monthBlocks.push({
      year: currentYear,
      month: currentMonth,
      monthData: months[currentMonth],
      title: `${months[currentMonth].title} ${currentYear}`,
      days: getNumberOfDaysInMonth(currentMonth, currentYear),
      today: todayMonth === currentMonth && todayYear === currentYear,
    });

    currentDate = new Date(currentDate.getFullYear(), currentDate.getMonth() + 1);
  }

  return monthBlocks;
};

/**
 * Merge two MonthView data payloads
 * @param a
 * @param b
 * @returns
 */
const mergeMonthRenderPayloads = (a: IMonthView, b: IMonthView): IMonthView => ({
  months: uniqBy([...a.months, ...b.months], (monthBlock) => `${monthBlock.month}_${monthBlock.year}`),
  weeks: uniqBy(
    [...a.weeks, ...b.weeks],
    (weekBlock) => `${weekBlock.startDate.getTime()}_${weekBlock.endDate.getTime()}`
  ),
});

export const monthView = {
  generateChart: generateMonthChart,
  mergeRenderPayloads: mergeMonthRenderPayloads,
};

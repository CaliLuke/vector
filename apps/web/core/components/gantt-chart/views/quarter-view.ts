/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

//
import { generateChartSegment } from "./chart-segment";
import type { ChartDataType } from "@plane/types";
import { quarters } from "../data";
import type { IMonthBlock } from "./month-view";
import { getMonthsBetweenTwoDates } from "./month-view";

export interface IQuarterMonthBlock {
  children: IMonthBlock[];
  quarterNumber: number;
  shortTitle: string;
  title: string;
  year: number;
  today: boolean;
}

/**
 * Generate Quarter Chart data, which in turn are months in an array
 * @param quarterPayload
 * @param side
 * @returns
 */
const generateQuarterChart = (chart: ChartDataType, side: null | "left" | "right", targetDate?: Date) =>
  generateChartSegment(
    chart,
    side,
    targetDate,
    (start, end) => {
      const payload = getMonthsBetweenTwoDates(start, end);
      const first = payload[0];
      const last = payload[payload.length - 1];
      return {
        payload,
        startDate: new Date(first.year, first.month, 1),
        endDate: new Date(last.year, last.month + 1, 0),
      };
    },
    true
  );

/**
 * Merge two Quarter data payloads
 * @param a
 * @param b
 * @returns
 */
const mergeQuarterRenderPayloads = (a: IMonthBlock[], b: IMonthBlock[]) => [...a, ...b];

/**
 * Group array of Months into Quarters, returns an array og Quarters and it's children Months
 * @param monthBlocks
 * @returns
 */
export const groupMonthsToQuarters = (monthBlocks: IMonthBlock[]): IQuarterMonthBlock[] => {
  const quartersMap: { [key: string]: IQuarterMonthBlock } = {};

  const today = new Date();
  const todayQuarterNumber = Math.floor(today.getMonth() / 3);
  const todayYear = today.getFullYear();

  for (const monthBlock of monthBlocks) {
    const { month, year } = monthBlock;

    const quarterNumber = Math.floor(month / 3);

    const quarterKey = `Q${quarterNumber}-${year}`;

    if (quartersMap[quarterKey]) {
      quartersMap[quarterKey].children.push(monthBlock);
    } else {
      const quarterData = quarters[quarterNumber];
      quartersMap[quarterKey] = {
        children: [monthBlock],
        quarterNumber,
        shortTitle: quarterData.shortTitle,
        title: `${quarterData.title} ${year}`,
        year,
        today: todayQuarterNumber === quarterNumber && todayYear === year,
      };
    }
  }

  return Object.values(quartersMap);
};

export const quarterView = {
  generateChart: generateQuarterChart,
  mergeRenderPayloads: mergeQuarterRenderPayloads,
};

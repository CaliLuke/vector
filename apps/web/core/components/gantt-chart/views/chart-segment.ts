/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ChartDataType } from "@plane/types";
import { getNumberOfDaysBetweenTwoDates } from "./helpers";

type Side = null | "left" | "right";
type Segment<T> = { payload: T; startDate: Date; endDate: Date };

function adjacentDay(date: Date, offset: number) {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate() + offset);
}

function requestedRange(chart: ChartDataType, side: Side, targetDate: Date | undefined, wholeMonths: boolean) {
  const { currentDate, startDate, endDate } = chart.data;
  const range = chart.data.approxFilterRange || (wholeMonths ? 12 : 6);
  if (side === null)
    return {
      startDate: new Date(
        currentDate.getFullYear(),
        currentDate.getMonth() - range,
        wholeMonths ? 1 : currentDate.getDate()
      ),
      endDate: new Date(
        currentDate.getFullYear(),
        currentDate.getMonth() + range,
        wholeMonths ? 0 : currentDate.getDate()
      ),
    };
  const extension = wholeMonths ? range / 2 : range;
  if (side === "left") {
    const anchor = targetDate ?? startDate;
    return {
      startDate: new Date(anchor.getFullYear(), anchor.getMonth() - extension, 1),
      endDate: wholeMonths
        ? new Date(startDate.getFullYear(), startDate.getMonth() - 1, 1)
        : adjacentDay(startDate, -1),
    };
  }
  const anchor = targetDate ?? endDate;
  return {
    startDate: wholeMonths ? new Date(endDate.getFullYear(), endDate.getMonth() + 1, 1) : adjacentDay(endDate, 1),
    endDate: new Date(anchor.getFullYear(), anchor.getMonth() + extension, 1),
  };
}

/** Extend only the requested edge; scroll distance covers only the newly added segment. */
export function generateChartSegment<T>(
  chart: ChartDataType,
  side: Side,
  targetDate: Date | undefined,
  buildSegment: (startDate: Date, endDate: Date) => Segment<T>,
  wholeMonths = false
) {
  const requested = requestedRange(chart, side, targetDate, wholeMonths);
  const segment = buildSegment(requested.startDate, requested.endDate);
  const startDate = side === "right" ? adjacentDay(chart.data.endDate, 1) : segment.startDate;
  const endDate = side === "left" ? adjacentDay(chart.data.startDate, -1) : segment.endDate;
  return {
    state: {
      ...chart,
      data: {
        ...chart.data,
        startDate: side === "right" ? chart.data.startDate : startDate,
        endDate: side === "left" ? chart.data.endDate : endDate,
      },
    },
    payload: segment.payload,
    scrollWidth: (Math.abs(getNumberOfDaysBetweenTwoDates(startDate, endDate)) + 1) * chart.data.dayWidth,
  };
}

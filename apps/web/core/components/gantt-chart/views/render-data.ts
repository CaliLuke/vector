/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { ChartDataType, EStartOfTheWeek } from "@plane/types";
import type { TGanttRenderData } from "../contexts/chart-state";
import { monthView } from "./month-view";
import { quarterView } from "./quarter-view";
import { weekView } from "./week-view";

type GeneratedView = { state: ChartDataType; payload: TGanttRenderData; scrollWidth: number };

export function generateGanttView(
  state: ChartDataType,
  side: null | "left" | "right",
  targetDate?: Date,
  startOfWeek?: EStartOfTheWeek
): GeneratedView {
  switch (state.key) {
    case "week": {
      const result = weekView.generateChart(state, side, targetDate, startOfWeek);
      return { ...result, payload: { kind: "week", data: result.payload } };
    }
    case "month": {
      const result = monthView.generateChart(state, side, targetDate);
      return { ...result, payload: { kind: "month", data: result.payload } };
    }
    case "quarter": {
      const result = quarterView.generateChart(state, side, targetDate);
      return { ...result, payload: { kind: "quarter", data: result.payload } };
    }
    default:
      throw new Error(`Unknown timeline view: ${state.key}`);
  }
}

export function mergeGanttRenderData(left: TGanttRenderData, right: TGanttRenderData): TGanttRenderData {
  if (left.kind === "week" && right.kind === "week")
    return { kind: "week", data: weekView.mergeRenderPayloads(left.data, right.data) };
  if (left.kind === "month" && right.kind === "month")
    return { kind: "month", data: monthView.mergeRenderPayloads(left.data, right.data) };
  if (left.kind === "quarter" && right.kind === "quarter")
    return { kind: "quarter", data: quarterView.mergeRenderPayloads(left.data, right.data) };
  throw new Error("Cannot merge different timeline views");
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext } from "react";
import type { ChartDataType, IGanttBlock, TGanttViews } from "@plane/types";
import type { IMonthBlock, IMonthView, IWeekBlock } from "../views";

export type TGanttRenderData =
  | { kind: "week"; data: IWeekBlock[] }
  | { kind: "month"; data: IMonthView }
  | { kind: "quarter"; data: IMonthBlock[] };

export type TGanttChartState = {
  currentView: TGanttViews;
  viewData: ChartDataType | undefined;
  renderData: TGanttRenderData;
  blocksById: Readonly<Record<string, IGanttBlock>>;
  activeBlockId: string | null;
  isDragging: boolean;
  setActiveBlockId: (blockId: string | null) => void;
  setIsDragging: (isDragging: boolean) => void;
  setBlockPosition: (blockId: string, position: { marginLeft: number; width: number }) => void;
};

const GanttChartStateContext = createContext<TGanttChartState | undefined>(undefined);

export const GanttChartStateProvider = GanttChartStateContext.Provider;

export function useGanttChartState(): TGanttChartState {
  const state = useContext(GanttChartStateContext);
  if (!state) throw new Error("useGanttChartState must be used within GanttChartRoot");
  return state;
}

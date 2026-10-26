/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";
// plane imports
// components
import type {
  ChartDataType,
  EStartOfTheWeek,
  IBlockUpdateData,
  IBlockUpdateDependencyData,
  TGanttViews,
} from "@plane/types";
import type { IGanttBlock } from "@plane/types";
import { cn } from "@plane/utils/common";
import { GanttChartHeader, GanttChartMainContent } from "@/components/gantt-chart";
// helpers
// hooks
import { useCurrentUserProfile } from "@/hooks/use-current-user-profile";
import { useGanttViewPreference } from "@/providers/gantt-view-preferences";
//
import { generateGanttView, mergeGanttRenderData } from "../views/render-data";
import { SIDEBAR_WIDTH } from "../constants";
import { currentViewDataWithView } from "../data";
import { getNumberOfDaysBetweenTwoDates } from "../views";
import { GanttChartStateProvider, type TGanttChartState, type TGanttRenderData } from "../contexts/chart-state";
import { projectGanttBlocks, shiftBlockPositions, type TGanttBlockPosition, type TGanttItemData } from "../state";

type ChartViewRootProps<TItem extends TGanttItemData> = {
  border: boolean;
  title: string;
  loaderTitle: string;
  blockIds: string[];
  blockItems: readonly TItem[];
  blockUpdateHandler: (block: TItem, payload: IBlockUpdateData) => void;
  blockToRender: (data: TItem) => React.ReactNode;
  sidebarToRender: (props: any) => React.ReactNode;
  enableBlockLeftResize: boolean | ((blockId: string) => boolean);
  enableBlockRightResize: boolean | ((blockId: string) => boolean);
  enableBlockMove: boolean | ((blockId: string) => boolean);
  enableReorder: boolean | ((blockId: string) => boolean);
  enableAddBlock: boolean | ((blockId: string) => boolean);
  enableSelection: boolean | ((blockId: string) => boolean);
  enableDependency: boolean | ((blockId: string) => boolean);
  bottomSpacing: boolean;
  showAllBlocks: boolean;
  loadMoreBlocks?: () => void;
  updateBlockDates?: (updates: IBlockUpdateDependencyData[]) => Promise<void>;
  canLoadMoreBlocks?: boolean;
  quickAdd?: React.ReactNode | undefined;
  showToday: boolean;
  isEpic?: boolean;
};

export function ChartViewRoot<TItem extends TGanttItemData>(props: ChartViewRootProps<TItem>) {
  const [fullScreenMode, setFullScreenMode] = useState(false);
  const toggleFullScreenMode = () => setFullScreenMode((current) => !current);
  const { currentView, selectView } = useGanttViewPreference();
  const { data } = useCurrentUserProfile();
  const startOfWeek = data?.start_of_the_week;
  return (
    <ChartCanvas
      key={`${currentView}:${startOfWeek}`}
      {...props}
      currentView={currentView}
      selectView={selectView}
      startOfWeek={startOfWeek}
      fullScreenMode={fullScreenMode}
      toggleFullScreenMode={toggleFullScreenMode}
    />
  );
}

function ChartCanvas<TItem extends TGanttItemData>(
  props: ChartViewRootProps<TItem> & {
    currentView: TGanttViews;
    selectView: (view: TGanttViews) => void;
    startOfWeek: EStartOfTheWeek | undefined;
    fullScreenMode: boolean;
    toggleFullScreenMode: () => void;
  }
) {
  const { currentView, selectView, startOfWeek, fullScreenMode, toggleFullScreenMode } = props;
  const [initialView] = useState(() => {
    const state = currentViewDataWithView(currentView);
    if (!state) throw new Error(`Unknown timeline view: ${currentView}`);
    return generateGanttView(state, null, undefined, startOfWeek);
  });
  const {
    border,
    title,
    blockIds,
    blockItems,
    loadMoreBlocks,
    loaderTitle,
    blockUpdateHandler,
    sidebarToRender,
    blockToRender,
    canLoadMoreBlocks,
    enableBlockLeftResize,
    enableBlockRightResize,
    enableBlockMove,
    enableReorder,
    enableAddBlock,
    enableSelection,
    enableDependency,
    bottomSpacing,
    showAllBlocks,
    quickAdd,
    showToday,
    updateBlockDates,
    isEpic = false,
  } = props;
  // states
  const [itemsContainerWidth, setItemsContainerWidth] = useState(initialView.scrollWidth);
  const [currentViewData, setCurrentViewData] = useState(initialView.state);
  const [renderView, setRenderView] = useState<TGanttRenderData>(initialView.payload);
  const [activeBlockId, setActiveBlockId] = useState<string | null>(null);
  const [isDraggingState, setIsDraggingState] = useState(false);
  const [dragSnapshot, setDragSnapshot] = useState<Record<string, IGanttBlock>>();
  const [positionOverrides, setPositionOverrides] = useState<Record<string, TGanttBlockPosition>>({});
  const ganttContainerRef = useRef<HTMLDivElement>(null);
  const scrollTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const projectedBlocks = useMemo(
    () => projectGanttBlocks(blockIds, blockItems, currentViewData, positionOverrides),
    [blockIds, blockItems, currentViewData, positionOverrides]
  );
  const isDragging = isDraggingState;
  const blocksById = useMemo(() => {
    const source = isDragging && dragSnapshot ? dragSnapshot : projectedBlocks;
    return Object.fromEntries(
      Object.entries(source).map(([id, block]) => [
        id,
        positionOverrides[id] ? { ...block, position: positionOverrides[id] } : block,
      ])
    );
  }, [dragSnapshot, isDragging, positionOverrides, projectedBlocks]);
  const setIsDragging = (dragging: boolean) => {
    if (dragging) {
      setDragSnapshot(blocksById);
      setPositionOverrides(
        Object.fromEntries(
          Object.entries(blocksById).flatMap(([id, block]) => (block.position ? [[id, block.position]] : []))
        )
      );
    } else {
      setDragSnapshot(undefined);
      setPositionOverrides({});
    }
    setIsDraggingState(dragging);
  };
  const setBlockPosition = (blockId: string, position: TGanttBlockPosition) => {
    setPositionOverrides((current) => ({ ...current, [blockId]: position }));
  };
  const chartState: TGanttChartState = {
    currentView,
    viewData: currentViewData,
    renderData: renderView,
    blocksById,
    activeBlockId,
    isDragging,
    setActiveBlockId,
    setIsDragging,
    setBlockPosition,
  };

  const updateCurrentLeftScrollPosition = (width: number) => {
    const scrollContainer = ganttContainerRef.current;
    if (scrollContainer) scrollContainer.scrollLeft += width;
  };

  const handleScrollToCurrentSelectedDate = (state: ChartDataType, date: Date) => {
    const container = ganttContainerRef.current;
    if (!container) return;
    const days = getNumberOfDaysBetweenTwoDates(state.data.startDate, date);
    container.scrollLeft =
      Math.abs(days) * state.data.dayWidth - (container.clientWidth / 2 - state.data.dayWidth) + SIDEBAR_WIDTH / 2;
  };

  useEffect(() => {
    const container = ganttContainerRef.current;
    if (container) {
      const { state } = initialView;
      const days = getNumberOfDaysBetweenTwoDates(state.data.startDate, state.data.currentDate);
      container.scrollLeft =
        Math.abs(days) * state.data.dayWidth - (container.clientWidth / 2 - state.data.dayWidth) + SIDEBAR_WIDTH / 2;
    }
    return () => {
      if (scrollTimer.current !== undefined) clearTimeout(scrollTimer.current);
    };
  }, [initialView]);

  const updateCurrentViewRenderPayload = (side: null | "left" | "right", view: TGanttViews, targetDate?: Date) => {
    const state = view === currentViewData.key ? currentViewData : currentViewDataWithView(view);
    if (!state) return;
    const next = generateGanttView(state, side, targetDate, startOfWeek);
    setCurrentViewData(next.state);
    if (side === "left") {
      setRenderView((previous) => mergeGanttRenderData(next.payload, previous));
      if (!targetDate) updateCurrentLeftScrollPosition(next.scrollWidth);
      if (isDragging) setPositionOverrides((previous) => shiftBlockPositions(previous, next.scrollWidth));
      setItemsContainerWidth((width) => width + next.scrollWidth);
    } else if (side === "right") {
      setRenderView((previous) => mergeGanttRenderData(previous, next.payload));
      setItemsContainerWidth((width) => width + next.scrollWidth);
    } else {
      setRenderView(next.payload);
      setItemsContainerWidth(next.scrollWidth);
      if (scrollTimer.current !== undefined) clearTimeout(scrollTimer.current);
      scrollTimer.current = setTimeout(() => {
        scrollTimer.current = undefined;
        handleScrollToCurrentSelectedDate(next.state, next.state.data.currentDate);
      }, 50);
    }
    return next.state;
  };

  const handleToday = () => updateCurrentViewRenderPayload(null, currentView);

  const portalContainer = document.getElementById("full-screen-portal");

  const content = (
    <div
      className={cn("shadow relative flex h-full flex-col rounded-xs bg-surface-1 select-none", {
        "inset-0 z-[25] bg-surface-1": fullScreenMode,
        "border-[0.5px] border-subtle": border,
      })}
    >
      <GanttChartHeader
        blockIds={blockIds}
        currentView={currentView}
        fullScreenMode={fullScreenMode}
        toggleFullScreenMode={toggleFullScreenMode}
        handleChartView={selectView}
        handleToday={handleToday}
        loaderTitle={loaderTitle}
        showToday={showToday}
      />
      <GanttChartMainContent
        ganttContainerRef={ganttContainerRef}
        blockIds={blockIds}
        loadMoreBlocks={loadMoreBlocks}
        canLoadMoreBlocks={canLoadMoreBlocks}
        blockToRender={blockToRender}
        blockUpdateHandler={blockUpdateHandler}
        bottomSpacing={bottomSpacing}
        enableBlockLeftResize={enableBlockLeftResize}
        enableBlockMove={enableBlockMove}
        enableBlockRightResize={enableBlockRightResize}
        enableReorder={enableReorder}
        enableSelection={enableSelection}
        enableAddBlock={enableAddBlock}
        enableDependency={enableDependency}
        itemsContainerWidth={itemsContainerWidth}
        showAllBlocks={showAllBlocks}
        sidebarToRender={sidebarToRender}
        title={title}
        updateCurrentViewRenderPayload={updateCurrentViewRenderPayload}
        currentView={currentView}
        currentViewData={currentViewData}
        quickAdd={quickAdd}
        updateBlockDates={updateBlockDates}
        isEpic={isEpic}
      />
    </div>
  );

  const chart = <GanttChartStateProvider value={chartState}>{content}</GanttChartStateProvider>;
  return fullScreenMode && portalContainer ? createPortal(chart, portalContainer) : chart;
}

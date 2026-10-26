/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
import type { IBaseLayoutsBaseItem, IBlockUpdateData } from "@plane/types";
import { Row } from "@plane/ui";
import { cn } from "@plane/utils/common";
import RenderIfVisible from "@/components/core/render-if-visible-HOC";
import { BLOCK_HEIGHT } from "@/components/gantt-chart/constants";
import { GanttDnDHOC } from "@/components/gantt-chart/sidebar/gantt-dnd-HOC";
import { GanttLayoutListItemLoader } from "@/components/ui/loader/layouts/gantt-layout-loader";
import { useGanttChartState } from "@/components/gantt-chart/contexts/chart-state";
import { daysFromGanttWidth } from "@/components/gantt-chart/state";
import {
  createGanttSidebarOrderChangeHandler,
  GanttSidebarDuration,
  GanttSidebarRows,
} from "@/components/gantt-chart/sidebar/common";

type Props<T extends IBaseLayoutsBaseItem> = {
  blockUpdateHandler: (block: T, payload: IBlockUpdateData) => void;
  canLoadMoreBlocks?: boolean;
  loadMoreItems?: (groupId: string) => void;
  ganttContainerRef: RefObject<HTMLDivElement | null>;
  blockIds: string[];
  enableReorder: boolean;
  showAllBlocks?: boolean;
  items: Record<string, T>;
  renderItem: (item: T) => React.ReactNode;
};

export function BaseGanttSidebar<T extends IBaseLayoutsBaseItem>(props: Props<T>) {
  const {
    blockUpdateHandler,
    blockIds,
    enableReorder,
    loadMoreItems,
    canLoadMoreBlocks,
    ganttContainerRef,
    showAllBlocks = false,
    items,
    renderItem,
  } = props;

  const { blocksById, activeBlockId, setActiveBlockId, viewData } = useGanttChartState();

  const isPaginating = false; // TODO: Add proper pagination state

  const handleOnDrop = createGanttSidebarOrderChangeHandler(blockIds, blocksById, blockUpdateHandler);

  return (
    <div>
      <GanttSidebarRows
        blockIds={blockIds}
        canLoadMore={Boolean(canLoadMoreBlocks)}
        ganttContainerRef={ganttContainerRef}
        loadMore={loadMoreItems ? () => loadMoreItems("") : undefined}
        isPaginating={isPaginating}
        renderBlock={(blockId, index) => {
          const block = blocksById[blockId];
          const item = items[blockId];
          const isBlockVisibleOnSidebar = block?.start_date && block?.target_date;

          // hide the block if it doesn't have start and target dates and showAllBlocks is false
          if (!block || (!showAllBlocks && !isBlockVisibleOnSidebar)) return null;
          if (!item) return null;

          return (
            <RenderIfVisible
              key={blockId}
              root={ganttContainerRef}
              horizontalOffset={100}
              verticalOffset={200}
              shouldRecordHeights={false}
              placeholderChildren={<GanttLayoutListItemLoader />}
            >
              <GanttDnDHOC
                id={blockId}
                isLastChild={index === blockIds.length - 1}
                isDragEnabled={enableReorder}
                onDrop={handleOnDrop}
              >
                {(isDragging: boolean) => {
                  const currentBlock = blocksById[blockId];
                  const isBlockComplete = !!currentBlock?.start_date && !!currentBlock?.target_date;
                  const duration = isBlockComplete
                    ? daysFromGanttWidth(viewData, currentBlock?.position?.width)
                    : undefined;
                  const isBlockHoveredOn = activeBlockId === blockId;

                  return (
                    <div
                      className={cn("group/list-block", {
                        "rounded-sm bg-layer-1": isDragging,
                      })}
                      onMouseEnter={() => setActiveBlockId(blockId)}
                      onMouseLeave={() => setActiveBlockId(null)}
                    >
                      <Row
                        className={cn(
                          "group flex w-full items-center gap-2 bg-layer-transparent pr-4 hover:bg-layer-transparent-hover",
                          {
                            "bg-layer-transparent-hover": isBlockHoveredOn,
                          }
                        )}
                        style={{
                          height: `${BLOCK_HEIGHT}px`,
                        }}
                      >
                        <div className="flex h-full flex-grow items-center justify-between gap-2 truncate">
                          <div className="flex-grow truncate">{renderItem(item)}</div>
                          {duration && <GanttSidebarDuration days={duration} />}
                        </div>
                      </Row>
                    </div>
                  );
                }}
              </GanttDnDHOC>
            </RenderIfVisible>
          );
        }}
      />
    </div>
  );
}

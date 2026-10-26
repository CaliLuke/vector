/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
import { useRef } from "react";
// components
import type { IBlockUpdateDependencyData } from "@plane/types";
import { cn } from "@plane/utils/common";
import RenderIfVisible from "@/components/core/render-if-visible-HOC";
// helpers
// hooks
import { useGanttChartState } from "../contexts/chart-state";
// constants
import { BLOCK_HEIGHT } from "../constants";
// components
import { ChartDraggable } from "../helpers";
import { useGanttResizable } from "../helpers/blockResizables/use-gantt-resizable";

type Props = {
  blockId: string;
  showAllBlocks: boolean;
  blockToRender: (data: any) => React.ReactNode;
  enableBlockLeftResize: boolean;
  enableBlockRightResize: boolean;
  enableBlockMove: boolean;
  enableDependency: boolean;
  ganttContainerRef: RefObject<HTMLDivElement | null>;
  updateBlockDates?: (updates: IBlockUpdateDependencyData[]) => Promise<void>;
};

export function GanttChartBlock(props: Props) {
  const {
    blockId,
    showAllBlocks,
    blockToRender,
    enableBlockLeftResize,
    enableBlockRightResize,
    enableBlockMove,
    ganttContainerRef,
    enableDependency,
    updateBlockDates,
  } = props;
  // store hooks
  const { setActiveBlockId, blocksById, currentView } = useGanttChartState();
  // refs
  const resizableRef = useRef<HTMLDivElement>(null);

  const block = blocksById[blockId];

  const { isMoving, handleBlockDrag } = useGanttResizable(block, resizableRef, ganttContainerRef, updateBlockDates);

  const isBlockVisibleOnChart = block?.start_date || block?.target_date;
  const isBlockComplete = block?.start_date && block?.target_date;

  // hide the block if it doesn't have start and target dates and showAllBlocks is false
  if (!block || (!showAllBlocks && !isBlockVisibleOnChart)) return null;

  if (!block.data) return null;

  return (
    <div
      className={cn("relative z-[5]", {
        "transition-all": !!isMoving && currentView === "week",
        "pointer-events-none": !isBlockVisibleOnChart,
      })}
      id={`gantt-block-${block.id}`}
      ref={resizableRef}
      style={{
        height: `${BLOCK_HEIGHT}px`,
        marginLeft: `${block.position?.marginLeft}px`,
        width: `${block.position?.width}px`,
      }}
    >
      {isBlockVisibleOnChart && (
        <RenderIfVisible
          root={ganttContainerRef}
          horizontalOffset={100}
          verticalOffset={200}
          classNames="flex h-full w-full items-center"
          placeholderChildren={<div className="h-8 w-full rounded-sm bg-layer-1" />}
          shouldRecordHeights={false}
        >
          <div
            className={cn("relative h-full w-full")}
            onMouseEnter={() => setActiveBlockId(blockId)}
            onMouseLeave={() => setActiveBlockId(null)}
          >
            <ChartDraggable
              block={block}
              blockToRender={blockToRender}
              handleBlockDrag={handleBlockDrag}
              enableBlockLeftResize={enableBlockLeftResize}
              enableBlockRightResize={enableBlockRightResize}
              enableBlockMove={enableBlockMove && !!isBlockComplete}
              enableDependency={enableDependency}
              isMoving={isMoving}
              ganttContainerRef={ganttContainerRef}
            />
          </div>
        </RenderIfVisible>
      )}
    </div>
  );
}

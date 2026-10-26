/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
// ui
import type { IBlockUpdateData } from "@plane/types";
// components
import RenderIfVisible from "@/components/core/render-if-visible-HOC";
import { GanttLayoutListItemLoader } from "@/components/ui/loader/layouts/gantt-layout-loader";
//hooks
import type { TSelectionHelper } from "@/hooks/use-multiple-select";
// local imports
import { useGanttChartState } from "../../contexts/chart-state";
import { GanttDnDHOC } from "../gantt-dnd-HOC";
import { IssuesSidebarBlock } from "./block";
import { createGanttSidebarOrderChangeHandler, GanttSidebarRows } from "../common";

type Props = {
  blockUpdateHandler: (block: any, payload: IBlockUpdateData) => void;
  canLoadMoreBlocks?: boolean;
  loadMoreBlocks?: () => void;
  ganttContainerRef: RefObject<HTMLDivElement | null>;
  blockIds: string[];
  enableReorder: boolean;
  enableSelection: boolean;
  showAllBlocks?: boolean;
  selectionHelpers?: TSelectionHelper;
  isEpic?: boolean;
  isPaginating?: boolean;
};

export function IssueGanttSidebar(props: Props) {
  const {
    blockUpdateHandler,
    blockIds,
    enableReorder,
    enableSelection,
    loadMoreBlocks,
    canLoadMoreBlocks,
    ganttContainerRef,
    showAllBlocks = false,
    selectionHelpers,
    isEpic = false,
    isPaginating = false,
  } = props;

  const { blocksById } = useGanttChartState();

  const handleOnDrop = createGanttSidebarOrderChangeHandler(blockIds, blocksById, blockUpdateHandler);

  return (
    <div>
      <GanttSidebarRows
        blockIds={blockIds}
        canLoadMore={Boolean(canLoadMoreBlocks)}
        ganttContainerRef={ganttContainerRef}
        loadMore={loadMoreBlocks}
        isPaginating={isPaginating}
        renderBlock={(blockId, index) => {
          const block = blocksById[blockId];
          const isBlockVisibleOnSidebar = block?.start_date && block?.target_date;

          // hide the block if it doesn't have start and target dates and showAllBlocks is false
          if (!block || (!showAllBlocks && !isBlockVisibleOnSidebar)) return null;

          return (
            <RenderIfVisible
              key={block.id}
              root={ganttContainerRef}
              horizontalOffset={100}
              verticalOffset={200}
              shouldRecordHeights={false}
              placeholderChildren={<GanttLayoutListItemLoader />}
            >
              <GanttDnDHOC
                id={block.id}
                isLastChild={index === blockIds.length - 1}
                isDragEnabled={enableReorder}
                onDrop={handleOnDrop}
              >
                {(isDragging: boolean) => (
                  <IssuesSidebarBlock
                    block={block}
                    enableSelection={enableSelection}
                    isDragging={isDragging}
                    selectionHelpers={selectionHelpers}
                    isEpic={isEpic}
                  />
                )}
              </GanttDnDHOC>
            </RenderIfVisible>
          );
        }}
      />
    </div>
  );
}

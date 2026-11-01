import type { TGanttSidebarProps } from "../types";
import type { TGanttBlock } from "../../state";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// ui
import type { TIssue } from "@plane/types";
// components
import RenderIfVisible from "@/components/core/render-if-visible-HOC";
import { GanttLayoutListItemLoader } from "@/components/ui/loader/layouts/gantt-layout-loader";
//hooks
// local imports
import { GanttDnDHOC } from "../gantt-dnd-HOC";
import { IssuesSidebarBlock } from "./block";
import { createGanttSidebarOrderChangeHandler, GanttSidebarRows } from "../common";

type Props = TGanttSidebarProps & {
  blocksById: Readonly<Record<string, TGanttBlock<TIssue>>>;
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

  const { blocksById } = props;

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
                isDragEnabled={typeof enableReorder === "function" ? enableReorder(blockId) : enableReorder}
                onDrop={handleOnDrop}
              >
                {(isDragging: boolean) => (
                  <IssuesSidebarBlock
                    block={block}
                    enableSelection={typeof enableSelection === "function" ? enableSelection(blockId) : enableSelection}
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

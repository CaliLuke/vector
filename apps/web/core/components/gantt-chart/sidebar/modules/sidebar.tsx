/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// ui
import type { IBlockUpdateData } from "@plane/types";
// components
// hooks
import { useGanttChartState } from "../../contexts/chart-state";
import { GanttDnDHOC } from "../gantt-dnd-HOC";
import { handleOrderChange } from "../utils";
import { ModulesSidebarBlock } from "./block";
import { GanttSidebarLoading } from "../common";
// types

type Props = {
  title: string;
  blockUpdateHandler: (block: any, payload: IBlockUpdateData) => void;
  blockIds: string[];
  enableReorder: boolean;
};

export function ModuleGanttSidebar(props: Props) {
  const { blockUpdateHandler, blockIds, enableReorder } = props;

  const { blocksById } = useGanttChartState();

  const handleOnDrop = (
    draggingBlockId: string | undefined,
    droppedBlockId: string | undefined,
    dropAtEndOfList: boolean
  ) => {
    handleOrderChange(draggingBlockId, droppedBlockId, dropAtEndOfList, blockIds, blocksById, blockUpdateHandler);
  };

  return (
    <div className="h-full">
      {blockIds ? (
        blockIds.map((blockId, index) => (
          <GanttDnDHOC
            key={blockId}
            id={blockId}
            isLastChild={index === blockIds.length - 1}
            isDragEnabled={enableReorder}
            onDrop={handleOnDrop}
          >
            {(isDragging: boolean) => <ModulesSidebarBlock blockId={blockId} isDragging={isDragging} />}
          </GanttDnDHOC>
        ))
      ) : (
        <GanttSidebarLoading />
      )}
    </div>
  );
}

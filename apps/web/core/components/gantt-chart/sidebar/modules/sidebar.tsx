import type { TGanttSidebarProps } from "../types";
import type { TGanttBlock } from "../../state";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// ui
import type { IModule } from "@plane/types";
// components
// hooks
import { GanttDnDHOC } from "../gantt-dnd-HOC";
import { handleOrderChange } from "../utils";
import { ModulesSidebarBlock } from "./block";
import { GanttSidebarLoading } from "../common";
// types

type Props = TGanttSidebarProps & {
  blocksById: Readonly<Record<string, TGanttBlock<IModule>>>;
};

export function ModuleGanttSidebar(props: Props) {
  const { blockUpdateHandler, blockIds, enableReorder } = props;

  const { blocksById } = props;

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
            isDragEnabled={typeof enableReorder === "function" ? enableReorder(blockId) : enableReorder}
            onDrop={handleOnDrop}
          >
            {(isDragging: boolean) => <ModulesSidebarBlock block={blocksById[blockId]} isDragging={isDragging} />}
          </GanttDnDHOC>
        ))
      ) : (
        <GanttSidebarLoading />
      )}
    </div>
  );
}

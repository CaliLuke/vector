/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IBlockUpdateData, IGanttBlock } from "@plane/types";

function sortOrderAt(index: number, blockIds: string[], blocks: Readonly<Record<string, IGanttBlock>>) {
  const order = (position: number) => blocks[blockIds[position]]?.sort_order ?? 0;
  if (index === 0) return order(0) - 1000;
  if (index === blockIds.length) return order(blockIds.length - 1) + 1000;
  return (order(index) + order(index - 1)) / 2;
}

export const handleOrderChange = (
  draggingBlockId: string | undefined,
  droppedBlockId: string | undefined,
  dropAtEndOfList: boolean,
  blockIds: string[] | null,
  blocksById: Readonly<Record<string, IGanttBlock>>,
  blockUpdateHandler: (blockId: string, payload: IBlockUpdateData) => void
) => {
  if (!blockIds || !draggingBlockId || !droppedBlockId) return;

  const sourceBlockIndex = blockIds.findIndex((id) => id === draggingBlockId);
  const destinationBlockIndex = dropAtEndOfList ? blockIds.length : blockIds.findIndex((id) => id === droppedBlockId);

  // return if dropped outside the list
  if (sourceBlockIndex === -1 || destinationBlockIndex === -1 || sourceBlockIndex === destinationBlockIndex) return;

  const sourceBlock = blocksById[draggingBlockId];
  if (!sourceBlock) return;

  // call the block update handler with the updated sort order, new and old index
  blockUpdateHandler(sourceBlock.id, {
    sort_order: {
      destinationIndex: destinationBlockIndex,
      newSortOrder: sortOrderAt(destinationBlockIndex, blockIds, blocksById),
      sourceIndex: sourceBlockIndex,
    },
  });
};

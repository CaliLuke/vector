/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import type { IBlockUpdateData, IBlockUpdateDependencyData } from "@plane/types";
// hooks
import { ChartViewRoot } from "./chart/root";
import type { TGanttItemData } from "./state";

type GanttChartRootProps<TItem extends TGanttItemData> = {
  border?: boolean;
  title: string;
  loaderTitle: string;
  blockIds: string[];
  blockItems: readonly TItem[];
  blockUpdateHandler: (block: TItem, payload: IBlockUpdateData) => void;
  blockToRender: (data: TItem) => React.ReactNode;
  sidebarToRender: (props: any) => React.ReactNode;
  quickAdd?: React.ReactNode | undefined;
  canLoadMoreBlocks?: boolean;
  loadMoreBlocks?: () => void;
  updateBlockDates?: (updates: IBlockUpdateDependencyData[]) => Promise<void>;
  enableBlockLeftResize?: boolean | ((blockId: string) => boolean);
  enableBlockRightResize?: boolean | ((blockId: string) => boolean);
  enableBlockMove?: boolean | ((blockId: string) => boolean);
  enableReorder?: boolean | ((blockId: string) => boolean);
  enableAddBlock?: boolean | ((blockId: string) => boolean);
  enableSelection?: boolean | ((blockId: string) => boolean);
  enableDependency?: boolean | ((blockId: string) => boolean);
  bottomSpacing?: boolean;
  showAllBlocks?: boolean;
  showToday?: boolean;
  isEpic?: boolean;
};

export function GanttChartRoot<TItem extends TGanttItemData>(props: GanttChartRootProps<TItem>) {
  const {
    border = true,
    title,
    blockIds,
    blockItems,
    loaderTitle = "blocks",
    blockUpdateHandler,
    sidebarToRender,
    blockToRender,
    loadMoreBlocks,
    canLoadMoreBlocks,
    enableBlockLeftResize = false,
    enableBlockRightResize = false,
    enableBlockMove = false,
    enableReorder = false,
    enableAddBlock = false,
    enableSelection = false,
    enableDependency = false,
    bottomSpacing = false,
    showAllBlocks = false,
    showToday = true,
    quickAdd,
    updateBlockDates,
    isEpic = false,
  } = props;

  return (
    <ChartViewRoot
      border={border}
      title={title}
      blockIds={blockIds}
      blockItems={blockItems}
      loadMoreBlocks={loadMoreBlocks}
      canLoadMoreBlocks={canLoadMoreBlocks}
      loaderTitle={loaderTitle}
      blockUpdateHandler={blockUpdateHandler}
      sidebarToRender={sidebarToRender}
      blockToRender={blockToRender}
      enableBlockLeftResize={enableBlockLeftResize}
      enableBlockRightResize={enableBlockRightResize}
      enableBlockMove={enableBlockMove}
      enableReorder={enableReorder}
      enableAddBlock={enableAddBlock}
      enableSelection={enableSelection}
      enableDependency={enableDependency}
      bottomSpacing={bottomSpacing}
      showAllBlocks={showAllBlocks}
      quickAdd={quickAdd}
      showToday={showToday}
      updateBlockDates={updateBlockDates}
      isEpic={isEpic}
    />
  );
}

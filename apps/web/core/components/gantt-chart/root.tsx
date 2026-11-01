import type { TGanttSidebarProps } from "@/components/gantt-chart/sidebar/types";
import type { TGanttBlock } from "@/components/gantt-chart/state";
/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import type { IBlockUpdateData, IBlockUpdateDependencyData } from "@plane/types";
import { useQueryClient } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { setToast, TOAST_TYPE } from "@plane/propel/toast";
import { useCommittedScope } from "@/hooks/use-committed-scope";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
// hooks
import { ChartViewRoot } from "./chart/root";
import type { TGanttItemData } from "./state";

type GanttChartRootProps<TItem extends TGanttItemData> = {
  border?: boolean;
  title: string;
  loaderTitle: string;
  blockIds: string[];
  blockItems: readonly TItem[];
  blockUpdateHandler: (block: TItem, payload: IBlockUpdateData) => void | Promise<void>;
  blockToRender: (data: TItem) => React.ReactNode;
  sidebarToRender: (
    props: TGanttSidebarProps & { blocksById: Readonly<Record<string, TGanttBlock<TItem>>> }
  ) => React.ReactNode;
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
    loaderTitle,
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

  const { workspaceSlug, projectId } = useParams({ strict: false });
  const accountId = useCurrentUserId();
  const client = useQueryClient();
  const generation = useIdentityGeneration(client);
  const isCurrent = useCommittedScope(JSON.stringify([accountId, generation, workspaceSlug, projectId]));
  const updateBlock = async (block: TItem, payload: IBlockUpdateData) => {
    try {
      await blockUpdateHandler(block, payload);
    } catch {
      if (isCurrent())
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Update failed",
          message: "Unable to update this timeline item. Please try again.",
        });
    }
  };

  return (
    <ChartViewRoot
      border={border}
      title={title}
      blockIds={blockIds}
      blockItems={blockItems}
      loadMoreBlocks={loadMoreBlocks}
      canLoadMoreBlocks={canLoadMoreBlocks}
      loaderTitle={loaderTitle}
      blockUpdateHandler={(block, payload) => {
        void updateBlock(block, payload);
      }}
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

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode, RefObject } from "react";
import { useState } from "react";
import { Loader } from "@plane/ui";
import type { IGanttBlock } from "@plane/types";
import { useIntersectionObserver } from "@/hooks/use-intersection-observer";
import { handleOrderChange } from "./utils";

type BlockUpdateHandler = Parameters<typeof handleOrderChange>[5];

export function createGanttSidebarOrderChangeHandler(
  blockIds: string[] | null,
  blocksById: Readonly<Record<string, IGanttBlock>>,
  blockUpdateHandler: BlockUpdateHandler
) {
  return (draggingBlockId: string | undefined, droppedBlockId: string | undefined, dropAtEndOfList: boolean) => {
    handleOrderChange(draggingBlockId, droppedBlockId, dropAtEndOfList, blockIds, blocksById, blockUpdateHandler);
  };
}

export function GanttSidebarDuration({ days }: { days: number }) {
  return (
    <div className="flex-shrink-0 text-13 text-secondary">
      {days} day{days > 1 ? "s" : ""}
    </div>
  );
}

function GanttSidebarLoadMoreMarker({ setElement }: { setElement: (element: HTMLDivElement | null) => void }) {
  return (
    <div ref={setElement} className="p-2">
      <div className="flex h-10 w-full animate-pulse items-center justify-between gap-1.5 rounded-sm bg-layer-1 px-4 py-1.5 md:h-8 md:px-1" />
    </div>
  );
}

export function GanttSidebarLoading() {
  return (
    <Loader className="space-y-3 pr-2">
      <Loader.Item height="34px" />
      <Loader.Item height="34px" />
      <Loader.Item height="34px" />
      <Loader.Item height="34px" />
    </Loader>
  );
}

function GanttSidebarLoadState({
  isLoading,
  canLoadMore,
  setElement,
}: {
  isLoading: boolean;
  canLoadMore: boolean;
  setElement: (element: HTMLDivElement | null) => void;
}) {
  if (isLoading) return <GanttSidebarLoading />;
  if (canLoadMore) return <GanttSidebarLoadMoreMarker setElement={setElement} />;
  return null;
}

export function GanttSidebarRows({
  blockIds,
  canLoadMore,
  ganttContainerRef,
  loadMore,
  isPaginating = false,
  renderBlock,
}: {
  blockIds: readonly string[] | null | undefined;
  canLoadMore: boolean;
  ganttContainerRef: RefObject<HTMLDivElement | null>;
  loadMore?: () => void;
  isPaginating?: boolean;
  renderBlock: (blockId: string, index: number) => ReactNode;
}) {
  const [intersectionElement, setIntersectionElement] = useState<HTMLDivElement | null>(null);

  useIntersectionObserver(ganttContainerRef, isPaginating ? null : intersectionElement, loadMore, "100% 0% 100% 0%");

  if (!blockIds) return <GanttSidebarLoadState isLoading canLoadMore={false} setElement={setIntersectionElement} />;

  return (
    <>
      {blockIds.map(renderBlock)}
      <GanttSidebarLoadState isLoading={false} canLoadMore={canLoadMore} setElement={setIntersectionElement} />
    </>
  );
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
import React from "react";

// hooks
import type { IGanttBlock } from "@plane/types";
// helpers
import { cn } from "@plane/utils/common";
// components
import { GanttResizeHandle } from "./blockResizables/resize-handle";

type Props = {
  block: IGanttBlock;
  blockToRender: (blockId: string) => React.ReactNode;
  handleBlockDrag: (e: React.MouseEvent<HTMLElement>, dragDirection: "left" | "right" | "move") => void;
  handleResizeToDay: (direction: "left" | "right", day: number) => Promise<void>;
  isSaving: boolean;
  isMoving: "left" | "right" | "move" | undefined;
  enableBlockLeftResize: boolean;
  enableBlockRightResize: boolean;
  enableBlockMove: boolean;
  enableDependency: boolean | ((blockId: string) => boolean);
  ganttContainerRef: RefObject<HTMLDivElement | null>;
};

export const ChartDraggable = function ChartDraggable(props: Props) {
  const {
    block,
    blockToRender,
    handleBlockDrag,
    handleResizeToDay,
    isSaving,
    enableBlockLeftResize,
    enableBlockRightResize,
    enableBlockMove,
    isMoving,
  } = props;

  return (
    <div className="group relative z-[5] inline-flex h-full w-full cursor-pointer items-center font-medium transition-all">
      {enableBlockLeftResize && (
        <GanttResizeHandle
          block={block}
          direction="left"
          handleBlockDrag={handleBlockDrag}
          handleResizeToDay={handleResizeToDay}
          isMoving={isMoving}
          isSaving={isSaving}
        />
      )}
      {/* oxlint-disable-next-line jsx_a11y/no-static-element-interactions */}
      <div
        className={cn("relative z-[6] flex h-8 w-full items-center rounded-sm", {
          "pointer-events-none": isMoving,
        })}
        onMouseDown={(e) => enableBlockMove && handleBlockDrag(e, "move")}
      >
        {blockToRender(block.id)}
      </div>
      {/* right resize drag handle */}
      {enableBlockRightResize && (
        <GanttResizeHandle
          block={block}
          direction="right"
          handleBlockDrag={handleBlockDrag}
          handleResizeToDay={handleResizeToDay}
          isMoving={isMoving}
          isSaving={isSaving}
        />
      )}
    </div>
  );
};

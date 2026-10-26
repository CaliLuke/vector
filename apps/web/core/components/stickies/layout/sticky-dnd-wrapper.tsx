/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLocation } from "@tanstack/react-router";
import { useEffect, useRef } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import type { DropTargetRecord, DragLocationHistory } from "@atlaskit/pragmatic-drag-and-drop/types";
import type { ElementDragPayload } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { attachReorderInstruction } from "@/lib/tree-drop-instruction";

// plane types
// components
import { StickyNote } from "../sticky";
// helpers

type Props = {
  stickyId: string;
  workspaceSlug: string;
  itemWidth: string;
  className?: string;
  isLastChild: boolean;
  isInFirstRow: boolean;
  isInLastRow: boolean;
  handleDrop: (self: DropTargetRecord, source: ElementDragPayload, location: DragLocationHistory) => void;
  handleLayout: () => void;
};

export const StickyDNDWrapper = function StickyDNDWrapper(props: Props) {
  const { stickyId, workspaceSlug, itemWidth, className = "", isLastChild, handleDrop, handleLayout } = props;
  // refs
  const elementRef = useRef<HTMLDivElement>(null);
  // navigation
  const pathname = useLocation({ select: (location) => location.pathname });

  useEffect(() => {
    const stickyElement = elementRef.current;
    if (!stickyElement) return;

    const initialData = { id: stickyId, type: "sticky", parentId: null, isGroup: false, isChild: false };

    if (pathname.includes("stickies"))
      return combine(
        draggable({
          element: stickyElement,
          dragHandle: stickyElement,
          getInitialData: () => initialData,
        }),
        dropTargetForElements({
          element: stickyElement,
          canDrop: ({ source }) => source.data?.type === "sticky",
          getData: (target) => attachReorderInstruction(initialData, target, isLastChild),
          onDrop: ({ self, source, location }) => {
            handleDrop(self, source, location);
          },
        })
      );
  }, [handleDrop, isLastChild, pathname, stickyId]);

  return (
    <div
      ref={elementRef}
      className={`box-border flex flex-col p-[8px] ${className}`}
      style={{
        width: itemWidth,
      }}
    >
      <StickyNote key={stickyId} workspaceSlug={workspaceSlug} stickyId={stickyId} handleLayout={handleLayout} />
    </div>
  );
};

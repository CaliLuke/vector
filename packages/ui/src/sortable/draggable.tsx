/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { attachClosestEdge, extractClosestEdge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";
import React, { useEffect, useRef, useState } from "react";
import { DropIndicator } from "../drop-indicator";
import { cn } from "../utils";

type Props = {
  children: React.ReactNode;
  data: SortableDragData;
  className?: string;
};

export type SortableDragData = {
  sortableId: string;
  sortableGroupId: string;
};

export function isSortableDragData(
  data: Record<string | symbol, unknown>
): data is Record<string | symbol, unknown> & SortableDragData {
  return typeof data.sortableId === "string" && typeof data.sortableGroupId === "string";
}

function Draggable({ children, data, className }: Props) {
  const ref = useRef<HTMLDivElement>(null);
  const [dragging, setDragging] = useState<boolean>(false);
  const [isDraggedOver, setIsDraggedOver] = useState(false);

  const [closestEdge, setClosestEdge] = useState<string | null>(null);
  useEffect(() => {
    const el = ref.current;

    if (el) {
      return combine(
        draggable({
          element: el,
          onDragStart: () => setDragging(true),
          onDrop: () => setDragging(false),
          getInitialData: () => data,
        }),
        dropTargetForElements({
          element: el,
          onDragEnter: (args) => {
            setIsDraggedOver(true);
            setClosestEdge(extractClosestEdge(args.self.data));
          },
          onDragLeave: () => setIsDraggedOver(false),
          onDrop: () => {
            setIsDraggedOver(false);
          },
          canDrop: ({ source }) =>
            isSortableDragData(source.data) &&
            source.data.sortableGroupId === data.sortableGroupId &&
            source.data.sortableId !== data.sortableId,
          getData: ({ input, element }) =>
            attachClosestEdge(data, {
              input,
              element,
              allowedEdges: ["top", "bottom"],
            }),
        })
      );
    }
    return undefined;
  }, [data]);

  return (
    <div ref={ref} className={cn(dragging && "opacity-25", className)}>
      {<DropIndicator isVisible={isDraggedOver && closestEdge === "top"} />}
      {children}
      {<DropIndicator isVisible={isDraggedOver && closestEdge === "bottom"} />}
    </div>
  );
}

export { Draggable };

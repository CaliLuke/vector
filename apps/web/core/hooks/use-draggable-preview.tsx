/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useEffectEvent } from "react";
import type { ReactNode, RefObject } from "react";
import { draggable } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { pointerOutsideOfPreview } from "@atlaskit/pragmatic-drag-and-drop/element/pointer-outside-of-preview";
import { setCustomNativeDragPreview } from "@atlaskit/pragmatic-drag-and-drop/element/set-custom-native-drag-preview";
import { renderNativeDragPreview } from "@/lib/native-drag-preview";

type Options = {
  elementRef: RefObject<HTMLElement | null>;
  getInitialData: () => Record<string, unknown>;
  preview: ReactNode;
  onDraggingChange: (isDragging: boolean) => void;
};

export function useDraggablePreview({ elementRef, getInitialData, preview, onDraggingChange }: Options) {
  const readData = useEffectEvent(getInitialData);
  const readPreview = useEffectEvent(() => preview);
  const markDragging = useEffectEvent(onDraggingChange);

  useEffect(() => {
    const element = elementRef.current;
    if (!element) return undefined;

    return draggable({
      element,
      dragHandle: element,
      getInitialData: () => readData(),
      onDragStart: () => markDragging(true),
      onDrop: () => markDragging(false),
      onGenerateDragPreview: ({ nativeSetDragImage }) => {
        const content = readPreview();
        setCustomNativeDragPreview({
          getOffset: pointerOutsideOfPreview({ x: "0px", y: "0px" }),
          nativeSetDragImage,
          render: ({ container }) =>
            renderNativeDragPreview(
              container,
              <div className="rounded-sm bg-surface-1 p-1 pr-2 text-13">{content}</div>
            ),
        });
      },
    });
  }, [elementRef]);
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { treeDropHandlers } from "@/lib/tree-drop-instruction";
import { renderNativeDragPreview } from "@/lib/native-drag-preview";
import React, { useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import type { DropTargetRecord, DragLocationHistory } from "@atlaskit/pragmatic-drag-and-drop/types";
import type { ElementDragPayload } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { pointerOutsideOfPreview } from "@atlaskit/pragmatic-drag-and-drop/element/pointer-outside-of-preview";
import { setCustomNativeDragPreview } from "@atlaskit/pragmatic-drag-and-drop/element/set-custom-native-drag-preview";
import { attachReorderInstruction } from "@/lib/tree-drop-instruction";

// plane types
import { useTranslation } from "@plane/i18n";
import type { InstructionType, TWidgetEntityData } from "@plane/types";
// plane ui
import { DropIndicator, ToggleSwitch } from "@plane/ui";
// plane utils
import { cn } from "@plane/utils/common";
// hooks
import { HOME_WIDGETS_LIST } from "../../home-dashboard-widgets";
import { WidgetItemDragHandle } from "./widget-item-drag-handle";
import { getCanDrop } from "./widget.helpers";

type Props = {
  widget: TWidgetEntityData;
  disabled: boolean;
  isLastChild: boolean;
  handleDrop: (self: DropTargetRecord, source: ElementDragPayload, location: DragLocationHistory) => void;
  handleToggle: () => void;
};

export const WidgetItem = function WidgetItem(props: Props) {
  // props
  const { widget, disabled, isLastChild, handleDrop, handleToggle } = props;
  //state
  const [isDragging, setIsDragging] = useState(false);
  const [instruction, setInstruction] = useState<InstructionType | undefined>(undefined);
  //ref
  const elementRef = useRef<HTMLDivElement>(null);
  // hooks
  const { t } = useTranslation();
  // derived values
  const widgetTitle = HOME_WIDGETS_LIST[widget.key]?.title;

  // drag and drop
  useEffect(() => {
    const element = elementRef.current;

    if (!element) return;
    const initialData = { id: widget.key, parentId: null, isGroup: false, isChild: false };
    return combine(
      draggable({
        element,
        dragHandle: element,
        getInitialData: () => initialData,
        onDragStart: () => {
          setIsDragging(true);
        },
        onDrop: () => {
          setIsDragging(false);
        },
        onGenerateDragPreview: ({ nativeSetDragImage }) => {
          setCustomNativeDragPreview({
            getOffset: pointerOutsideOfPreview({ x: "0px", y: "0px" }),
            nativeSetDragImage,
            render: ({ container }) =>
              renderNativeDragPreview(
                container,
                <div className="rounded-sm bg-surface-1 p-1 pr-2 text-13">{widget.key}</div>
              ),
          });
        },
      }),
      dropTargetForElements({
        element,
        canDrop: ({ source }) => getCanDrop(source, widget),
        onDragStart: () => {
          setIsDragging(true);
        },
        getData: (target) => attachReorderInstruction(initialData, target, isLastChild),
        ...treeDropHandlers(setInstruction, handleDrop),
      })
    );
  }, [handleDrop, isLastChild, widget]);

  return (
    <div className="">
      <DropIndicator isVisible={instruction === "reorder-above"} />
      <div
        ref={elementRef}
        className={cn(
          "group/widget-item relative flex items-center justify-between rounded-sm px-2 py-2 text-13 font-medium hover:bg-layer-1",
          {
            "cursor-grabbing bg-layer-1": isDragging,
          }
        )}
      >
        <div className="flex items-center">
          <WidgetItemDragHandle sort_order={widget.sort_order} isDragging={isDragging} />
          <div>{t(widgetTitle, { count: 1 })}</div>
        </div>
        <ToggleSwitch value={widget.is_enabled} onChange={handleToggle} disabled={disabled} />
      </div>
      {isLastChild && <DropIndicator isVisible={instruction === "reorder-below"} />}
    </div>
  );
};

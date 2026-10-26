/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { treeDropPositionTarget } from "@/helpers/tree-drop-position";

import { useOutsideClickDetector } from "@plane/hooks";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { DropIndicator } from "@plane/ui";
import { HIGHLIGHT_WITH_LINE, highlightIssueOnDrop } from "@/components/issues/issue-layouts/utils";

type Props = {
  id: string;
  isLastChild: boolean;
  isDragEnabled: boolean;
  children: (isDragging: boolean) => React.ReactNode;
  onDrop: (draggingBlockId: string | undefined, droppedBlockId: string | undefined, dropAtEndOfList: boolean) => void;
};

export const GanttDnDHOC = function GanttDnDHOC(props: Props) {
  const { id, isLastChild, children, onDrop, isDragEnabled } = props;
  // states
  const [isDragging, setIsDragging] = useState(false);
  const [instruction, setInstruction] = useState<"DRAG_OVER" | "DRAG_BELOW" | undefined>(undefined);
  // refs
  const blockRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const element = blockRef.current;

    if (!element) return;

    return combine(
      draggable({
        element,
        canDrag: () => isDragEnabled,
        getInitialData: () => ({ id, dragInstanceId: "GANTT_REORDER" }),
        onDragStart: () => {
          setIsDragging(true);
        },
        onDrop: () => {
          setIsDragging(false);
        },
      }),
      dropTargetForElements({
        element,
        canDrop: ({ source }) => source?.data?.id !== id && source?.data?.dragInstanceId === "GANTT_REORDER",
        ...treeDropPositionTarget(id, isLastChild, setInstruction, (sourceId, destinationId, below, sourceElement) => {
          onDrop(sourceId, destinationId, below);
          highlightIssueOnDrop(sourceElement?.id, false, true);
        }),
      })
    );
  }, [id, isDragEnabled, isLastChild, onDrop]);

  useOutsideClickDetector(blockRef, () => blockRef?.current?.classList?.remove(HIGHLIGHT_WITH_LINE));

  return (
    <div
      id={`draggable-${id}`}
      className={"relative"}
      ref={blockRef}
      onDragStart={() => {
        if (!isDragEnabled) {
          setToast({
            title: "Warning!",
            type: TOAST_TYPE.WARNING,
            message: "Drag and drop is only enabled when sorted by manual",
          });
        }
      }}
    >
      <DropIndicator classNames="absolute top-0" isVisible={instruction === "DRAG_OVER"} />
      {children(isDragging)}
      {isLastChild && <DropIndicator isVisible={instruction === "DRAG_BELOW"} />}
    </div>
  );
};

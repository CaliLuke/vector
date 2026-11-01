/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Fragment, useCallback, useEffect, useRef, useState } from "react";
import { combine } from "@atlaskit/pragmatic-drag-and-drop/combine";
import { draggable, dropTargetForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { attachClosestEdge, extractClosestEdge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";
// Vector
import type { TDraggableData } from "@plane/constants";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { IState, TStateGroups, TStateOperationsCallbacks } from "@plane/types";
import { DropIndicator } from "@plane/ui";
import { cn } from "@plane/utils/common";
import { getCurrentStateSequence } from "@plane/utils/work-item";
// components
import { StateItemTitle, StateUpdate } from "@/components/project-states";
// helpers
type TStateItem = {
  groupKey: TStateGroups;
  groupedStates: Record<string, IState[]>;
  totalStates: number;
  state: IState;
  stateOperationsCallbacks: TStateOperationsCallbacks;
  shouldTrackEvents: boolean;
  disabled?: boolean;
  stateItemClassName?: string;
};

export function StateItem(props: TStateItem) {
  const {
    groupKey,
    groupedStates,
    totalStates,
    state,
    stateOperationsCallbacks,
    shouldTrackEvents,
    disabled = false,
    stateItemClassName,
  } = props;
  // ref
  const draggableElementRef = useRef<HTMLDivElement | null>(null);
  // states
  const [updateStateModal, setUpdateStateModal] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const [isDraggedOver, setIsDraggedOver] = useState(false);
  const [closestEdge, setClosestEdge] = useState<string | null>(null);
  // derived values
  const isDraggable = totalStates !== 1;
  const commonStateItemListProps = {
    stateCount: totalStates,
    statePercentage:
      (((groupedStates[state.group]?.findIndex((row) => row.id === state.id) ?? -1) + 1) /
        Math.max(groupedStates[state.group]?.length ?? 0, 1)) *
      100,
    state: state,
    setUpdateStateModal: setUpdateStateModal,
  };

  const handleStateSequence = useCallback(
    async (payload: Partial<IState>) => {
      try {
        if (!payload.id) return;
        await stateOperationsCallbacks.moveStatePosition(payload.id, payload);
      } catch {
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message: "State order could not be updated. Please try again.",
        });
      }
    },
    [stateOperationsCallbacks]
  );

  useEffect(() => {
    const elementRef = draggableElementRef.current;
    const initialData: TDraggableData = { groupKey: groupKey, id: state.id };

    const getSourceState = (data: Record<string | symbol, unknown>) => {
      if (typeof data.id !== "string" || typeof data.groupKey !== "string") return undefined;
      if (!Object.hasOwn(groupedStates, data.groupKey)) return undefined;
      return groupedStates[data.groupKey].find((candidate) => candidate.id === data.id);
    };
    const canAcceptSource = (data: Record<string | symbol, unknown>) => {
      const sourceState = getSourceState(data);
      if (disabled || !sourceState || sourceState.id === state.id) return false;
      return sourceState.group === groupKey || groupedStates[sourceState.group].length > 1;
    };

    if (elementRef && state) {
      return combine(
        draggable({
          element: elementRef,
          getInitialData: () => initialData,
          onDragStart: () => setIsDragging(true),
          onDrop: () => setIsDragging(false),
          canDrag: () => isDraggable && !disabled,
        }),
        dropTargetForElements({
          element: elementRef,
          canDrop: ({ source }) => canAcceptSource(source.data),
          getData: ({ input, element }) =>
            attachClosestEdge(initialData, {
              input,
              element,
              allowedEdges: ["top", "bottom"],
            }),
          onDragEnter: (args) => {
            setIsDraggedOver(true);
            setClosestEdge(extractClosestEdge(args.self.data));
          },
          onDragLeave: () => {
            setIsDraggedOver(false);
            setClosestEdge(null);
          },
          onDrop: (data) => {
            setIsDraggedOver(false);
            const { self, source } = data;
            if (!canAcceptSource(source.data)) return;
            const sourceState = getSourceState(source.data);
            if (!sourceState) return;
            const edge = extractClosestEdge(self.data) || undefined;
            void handleStateSequence({
              id: sourceState.id,
              group: groupKey,
              sequence: getCurrentStateSequence(groupedStates[groupKey], initialData, edge),
            });
          },
        })
      );
    }

    return undefined;
  }, [draggableElementRef, state, groupKey, isDraggable, groupedStates, handleStateSequence, disabled]);
  // DND ends

  if (updateStateModal)
    return (
      <StateUpdate
        state={state}
        updateStateCallback={stateOperationsCallbacks.updateState}
        shouldTrackEvents={shouldTrackEvents}
        handleClose={() => setUpdateStateModal(false)}
      />
    );

  return (
    <Fragment>
      {/* draggable drop top indicator */}
      <DropIndicator isVisible={isDraggedOver && closestEdge === "top"} />
      <div
        ref={draggableElementRef}
        className={cn(
          "group relative rounded-sm border border-subtle bg-surface-1 px-3.5 py-3",
          isDragging ? `opacity-50` : `opacity-100`,
          totalStates === 1 ? `cursor-auto` : `cursor-grab`,
          stateItemClassName
        )}
      >
        {disabled ? (
          <StateItemTitle {...commonStateItemListProps} disabled />
        ) : (
          <StateItemTitle
            {...commonStateItemListProps}
            disabled={false}
            stateOperationsCallbacks={{
              markStateAsDefault: stateOperationsCallbacks.markStateAsDefault,
              deleteState: stateOperationsCallbacks.deleteState,
            }}
            shouldTrackEvents={shouldTrackEvents}
          />
        )}
      </div>
      {/* draggable drop bottom indicator */}
      <DropIndicator isVisible={isDraggedOver && closestEdge === "bottom"} />
    </Fragment>
  );
}

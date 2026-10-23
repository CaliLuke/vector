/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { monitorForElements } from "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter";
import { extractClosestEdge } from "@atlaskit/pragmatic-drag-and-drop-hitbox/closest-edge";
import React, { Fragment, useEffect } from "react";
import { Draggable, isSortableDragData, type SortableDragData } from "./draggable";

type Props<T> = {
  data: T[];
  render: (item: T, index: number) => React.ReactNode;
  onChange: (data: T[], movedItem?: T) => void;
  keyExtractor: (item: T, index: number) => string;
  containerClassName?: string;
  id?: string;
};

const moveItem = <T,>(
  data: T[],
  source: SortableDragData,
  destination: SortableDragData & Record<string | symbol, unknown>,
  keyExtractor: (item: T, index: number) => string
): {
  newData: T[];
  movedItem: T | undefined;
} => {
  const sourceIndex = data.findIndex((item, index) => keyExtractor(item, index) === source.sortableId);
  if (sourceIndex === -1) return { newData: data, movedItem: undefined };

  const destinationIndex = data.findIndex((item, index) => keyExtractor(item, index) === destination.sortableId);

  if (destinationIndex === -1) return { newData: data, movedItem: undefined };

  const position = extractClosestEdge(destination);

  // Calculate final position before removing source item
  const finalIndex = position === "bottom" ? destinationIndex + 1 : destinationIndex;

  // Adjust for the fact that we're removing the source item first
  // If source is before destination, removing it shifts everything back by 1
  const adjustedDestinationIndex = finalIndex > sourceIndex ? finalIndex - 1 : finalIndex;

  const newData = [...data];
  const movedItem = newData.splice(sourceIndex, 1)[0];
  if (!movedItem) return { newData: data, movedItem: undefined };

  // Insert at the calculated position (bounds check is implicit in splice)
  newData.splice(adjustedDestinationIndex, 0, movedItem);

  return {
    newData,
    movedItem,
  };
};

export function Sortable<T>({ data, render, onChange, keyExtractor, containerClassName, id }: Props<T>) {
  const generatedId = React.useId();
  useEffect(() => {
    const unsubscribe = monitorForElements({
      onDrop({ source, location }) {
        const destination = location?.current?.dropTargets[0];
        if (!destination) return;
        if (!isSortableDragData(source.data) || !isSortableDragData(destination.data)) return;
        if (source.data.sortableGroupId !== destination.data.sortableGroupId) return;
        const { newData, movedItem } = moveItem(data, source.data, destination.data, keyExtractor);
        onChange(newData, movedItem);
      },
    });

    // Clean up the subscription on unmount
    return () => {
      if (unsubscribe) unsubscribe();
    };
  }, [data, keyExtractor, onChange]);

  const groupId = id ?? generatedId;

  return (
    <>
      {data.map((item, index) => (
        <Draggable
          key={keyExtractor(item, index)}
          data={{ sortableId: keyExtractor(item, index), sortableGroupId: groupId }}
          className={containerClassName}
        >
          <Fragment>{render(item, index)}</Fragment>
        </Draggable>
      ))}
    </>
  );
}

export default Sortable;

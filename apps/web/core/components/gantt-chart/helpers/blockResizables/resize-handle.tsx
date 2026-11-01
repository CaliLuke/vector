/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import type { IGanttBlock } from "@plane/types";
import { cn } from "@plane/utils/common";
import { renderFormattedDate } from "@plane/utils/datetime";
import { useGanttChartState } from "../../contexts/chart-state";
import { ganttResizeRange } from "../../state";
import { getDateFromPositionOnGantt } from "../../views/helpers";

type Props = {
  block: IGanttBlock;
  direction: "left" | "right";
  handleBlockDrag: (e: React.MouseEvent<HTMLElement>, direction: "left" | "right" | "move") => void;
  handleResizeToDay: (direction: "left" | "right", day: number) => Promise<void>;
  isMoving: "left" | "right" | "move" | undefined;
  isSaving: boolean;
};

function nextResizeDay(key: string, range: { value: number; min: number; max: number }) {
  switch (key) {
    case "ArrowRight":
    case "ArrowUp":
      return range.value + 1;
    case "ArrowLeft":
    case "ArrowDown":
      return range.value - 1;
    case "Home":
      return range.min;
    case "End":
      return range.max;
    default:
      return undefined;
  }
}

export function GanttResizeHandle({ block, direction, handleBlockDrag, handleResizeToDay, isMoving, isSaving }: Props) {
  const [hovering, setHovering] = useState(false);
  const [focused, setFocused] = useState(false);
  const { viewData } = useGanttChartState();
  if (!viewData) return null;
  const range = ganttResizeRange(block, viewData, direction);
  if (!range) return null;
  const date = getDateFromPositionOnGantt(range.value * viewData.data.dayWidth, viewData);
  const dateString = renderFormattedDate(date);
  const active = isMoving === direction || isMoving === "move";
  const left = direction === "left";
  return (
    <>
      {(hovering || focused || active) && (
        <div
          className={cn(
            "absolute z-[10] flex h-full w-32 items-center text-11 font-regular text-tertiary",
            left ? "-left-36 justify-end" : "-right-36 justify-start"
          )}
        >
          <div className="rounded-sm bg-accent-subtle px-2 py-1">{dateString}</div>
        </div>
      )}
      <input
        type="range"
        tabIndex={0}
        aria-label={`${left ? "Start date" : "Due date"}: ${block.name}`}
        value={range.value}
        min={range.min}
        max={range.max}
        step={1}
        onChange={(event) => {
          if (!isSaving) void handleResizeToDay(direction, event.currentTarget.valueAsNumber);
        }}
        aria-valuetext={dateString}
        aria-disabled={isSaving}
        onKeyDown={(event) => {
          const next = nextResizeDay(event.key, range);
          if (next === undefined) return;
          event.preventDefault();
          event.stopPropagation();
          if (!isSaving) void handleResizeToDay(direction, next);
        }}
        onMouseDown={(event) => {
          event.currentTarget.focus();
          event.preventDefault();
          handleBlockDrag(event, direction);
        }}
        onMouseEnter={() => setHovering(true)}
        onMouseLeave={() => setHovering(false)}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        className={cn(
          "absolute top-1/2 z-[6] m-0 h-full w-3 -translate-y-1/2 cursor-col-resize appearance-none rounded-md border-0 bg-transparent p-0 focus-visible:outline-2 focus-visible:outline-accent-strong [&::-moz-range-thumb]:border-0 [&::-moz-range-thumb]:bg-transparent [&::-webkit-slider-thumb]:appearance-none",
          left ? "-left-1.5" : "-right-1.5"
        )}
      />
      <div
        aria-hidden="true"
        className={cn(
          "absolute top-1/2 z-[5] h-7 w-1 -translate-y-1/2 rounded-xs bg-surface-1 opacity-0 transition-all duration-300 group-hover:opacity-100",
          left ? "left-1" : "right-1",
          { "opacity-100": active || focused }
        )}
      />
    </>
  );
}

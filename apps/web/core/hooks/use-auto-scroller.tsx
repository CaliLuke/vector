/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
import { useEffect } from "react";

const SCROLL_BY = 3;
const AUTO_SCROLL_THRESHOLD = 15;
const MAX_SPEED_THRESHOLD = 5;

function axisScrollSpeed(position: number, extent: number, movement: number): number {
  const threshold = (extent * AUTO_SCROLL_THRESHOLD) / 100;
  const maxSpeedThreshold = (extent * MAX_SPEED_THRESHOLD) / 100;
  if (position < threshold && movement <= 0) {
    return -SCROLL_BY * (position < maxSpeedThreshold ? 2 : 1);
  }
  if (position > extent - threshold && movement >= 0) {
    return SCROLL_BY * (position > extent - maxSpeedThreshold ? 2 : 1);
  }
  return 0;
}

export const useAutoScroller = (
  containerRef: RefObject<HTMLDivElement | null>,
  shouldScroll = false,
  leftOffset = 0,
  topOffset = 0
) => {
  useEffect(() => {
    const container = containerRef.current;
    if (!container || !shouldScroll) return undefined;
    const { left, top, width, height } = container.getBoundingClientRect();
    let interval: ReturnType<typeof setInterval> | undefined;
    let previousPosition: { clientX: number; clientY: number } | undefined;
    const stopScrolling = () => {
      clearInterval(interval);
      interval = undefined;
      previousPosition = undefined;
    };
    const handleAutoScroll = (event: MouseEvent) => {
      clearInterval(interval);
      if (event.clientX === 0 && event.clientY === 0) return;
      const dx = previousPosition ? event.clientX - previousPosition.clientX : 0;
      const dy = previousPosition ? event.clientY - previousPosition.clientY : 0;
      previousPosition = { clientX: event.clientX, clientY: event.clientY };
      const x = axisScrollSpeed(event.clientX - left - leftOffset, width - leftOffset, dx);
      const y = axisScrollSpeed(event.clientY - top - topOffset, height - topOffset, dy);
      if (!x && !y) return;
      container.scrollBy(x, y);
      interval = setInterval(() => container.scrollBy(x, y), 16);
    };

    container.addEventListener("drag", handleAutoScroll);
    container.addEventListener("mousemove", handleAutoScroll);
    document.addEventListener("mouseup", stopScrolling);
    document.addEventListener("dragend", stopScrolling);
    return () => {
      stopScrolling();
      container.removeEventListener("drag", handleAutoScroll);
      container.removeEventListener("mousemove", handleAutoScroll);
      document.removeEventListener("mouseup", stopScrolling);
      document.removeEventListener("dragend", stopScrolling);
    };
  }, [containerRef, shouldScroll, leftOffset, topOffset]);
};

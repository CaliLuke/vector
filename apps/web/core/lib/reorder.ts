/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import orderBy from "lodash-es/orderBy.js";
export function reorderedSortOrder(
  items: { id: string; sort_order: number }[],
  movingId: string,
  targetId: string,
  edge: "reorder-above" | "reorder-below"
): number {
  const sorted = orderBy(
    items.filter((item) => item.id !== movingId),
    "sort_order",
    "desc"
  );
  const targetIndex = sorted.findIndex((item) => item.id === targetId);
  const target = sorted[targetIndex];
  if (!target) throw new Error("The drop target is no longer available");
  const neighbor = sorted[targetIndex + (edge === "reorder-above" ? -1 : 1)];
  if (neighbor) return (target.sort_order + neighbor.sort_order) / 2;
  return target.sort_order + (edge === "reorder-above" ? 10000 : -10000);
}

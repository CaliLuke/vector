/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ChartDataType, IBlockUpdateDependencyData, IGanttBlock } from "@plane/types";
import { renderFormattedPayloadDate } from "@plane/utils/datetime";
import { getItemPositionWidth, getDateFromPositionOnGantt, getNumberOfDaysBetweenTwoDates } from "./views/helpers";

export type TGanttItemData = {
  id: string;
  name?: string;
  sort_order?: number | null;
  start_date?: string | null;
  target_date?: string | null;
  project_id?: string | null;
};

export type TGanttBlock<TItem extends TGanttItemData = TGanttItemData> = IGanttBlock<TItem>;

export type TGanttBlockPosition = { marginLeft: number; width: number };

export function projectGanttBlocks<TItem extends TGanttItemData>(
  itemIds: readonly string[],
  items: readonly TItem[],
  viewData: ChartDataType | undefined,
  positions: Readonly<Record<string, TGanttBlockPosition>>
): Record<string, TGanttBlock<TItem>> {
  const itemById = new Map(items.map((item) => [item.id, item]));
  const blocks: Record<string, TGanttBlock<TItem>> = {};

  for (const id of itemIds) {
    const data = itemById.get(id);
    if (!data) continue;
    const block: TGanttBlock<TItem> = {
      data,
      id: data.id,
      name: data.name ?? "",
      sort_order: data.sort_order ?? undefined,
      start_date: data.start_date ?? undefined,
      target_date: data.target_date ?? undefined,
      meta: { project_id: data.project_id },
    };
    const position = positions[id] ?? (viewData ? getItemPositionWidth(viewData, block) : undefined);
    if (position) block.position = position;
    blocks[id] = block;
  }

  return blocks;
}

export function daysFromGanttWidth(viewData: ChartDataType | undefined, width: number | undefined) {
  if (!viewData || !width) return undefined;
  return Math.round(width / viewData.data.dayWidth);
}

export function blockDateUpdatesAfterDrag(
  block: IGanttBlock | undefined,
  viewData: ChartDataType | undefined,
  shouldUpdateHalfBlock: boolean
): IBlockUpdateDependencyData[] {
  if (!block?.position || !viewData) return [];

  const update: IBlockUpdateDependencyData = { id: block.id, meta: block.meta };
  if (shouldUpdateHalfBlock || block.start_date) {
    update.start_date = renderFormattedPayloadDate(getDateFromPositionOnGantt(block.position.marginLeft, viewData));
  }
  if (shouldUpdateHalfBlock || block.target_date) {
    update.target_date = renderFormattedPayloadDate(
      getDateFromPositionOnGantt(block.position.marginLeft + block.position.width, viewData, -1)
    );
  }
  return [update];
}

export function shiftBlockPositions(
  positions: Readonly<Record<string, TGanttBlockPosition>>,
  addedWidth: number
): Record<string, TGanttBlockPosition> {
  return Object.fromEntries(
    Object.entries(positions).map(([id, position]) => [
      id,
      { ...position, marginLeft: position.marginLeft + addedWidth },
    ])
  );
}

export function ganttResizeRange(block: IGanttBlock, viewData: ChartDataType, direction: "left" | "right") {
  if (!block.position) return undefined;
  const { dayWidth, startDate, endDate } = viewData.data;
  const start = Math.round(block.position.marginLeft / dayWidth);
  const end = Math.round((block.position.marginLeft + block.position.width) / dayWidth) - 1;
  const lastDay = getNumberOfDaysBetweenTwoDates(new Date(endDate), new Date(startDate));
  if (direction === "left") {
    return { value: start, min: Math.min(0, start), max: block.target_date ? end : Math.max(lastDay, start) };
  }
  return { value: end, min: block.start_date ? start : Math.min(0, end), max: Math.max(lastDay, end) };
}

export function resizedGanttPosition(
  block: IGanttBlock,
  viewData: ChartDataType,
  direction: "left" | "right",
  day: number
) {
  const range = ganttResizeRange(block, viewData, direction);
  if (!range || !block.position || !Number.isFinite(day)) return undefined;
  const nextDay = Math.max(range.min, Math.min(range.max, Math.round(day)));
  if (nextDay === range.value) return undefined;
  const { dayWidth } = viewData.data;
  const { marginLeft, width } = block.position;
  if (direction === "left") {
    const nextLeft = nextDay * dayWidth;
    return { marginLeft: nextLeft, width: block.target_date ? marginLeft + width - nextLeft : width };
  }
  const nextRight = (nextDay + 1) * dayWidth;
  return {
    marginLeft: block.start_date ? marginLeft : nextRight - width,
    width: block.start_date ? nextRight - marginLeft : width,
  };
}

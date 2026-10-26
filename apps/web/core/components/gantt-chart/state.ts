/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ChartDataType, IBlockUpdateDependencyData, IGanttBlock } from "@plane/types";
import { renderFormattedPayloadDate } from "@plane/utils/datetime";
import { getItemPositionWidth, getDateFromPositionOnGantt } from "./views/helpers";

export type TGanttItemData = {
  id: string;
  name?: string;
  sort_order?: number | null;
  start_date?: string | null;
  target_date?: string | null;
  project_id?: string | null;
};

export type TGanttBlock<TItem extends TGanttItemData = TGanttItemData> = Omit<IGanttBlock, "data"> & {
  data: TItem;
};

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

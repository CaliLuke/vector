/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { extractInstruction } from "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item";
import type { IPragmaticPayloadLocation, InstructionType, TDropTarget } from "@plane/types";

type TreeDropTargetData = {
  id: string;
  parentId: string | null;
  isGroup: boolean;
  isChild: boolean;
};

export function isTreeDropTargetData(data: Record<string | symbol, unknown> | undefined): data is TreeDropTargetData {
  return (
    typeof data?.id === "string" &&
    data.id.length > 0 &&
    (data.parentId === null || typeof data.parentId === "string") &&
    typeof data.isGroup === "boolean" &&
    typeof data.isChild === "boolean"
  );
}

/** Resolve placement for a flat list with one level of groups. */
export const getTreeDropInstruction = (
  dropTarget: TDropTarget,
  source: TDropTarget,
  location: IPragmaticPayloadLocation
): InstructionType | undefined => {
  const targetData = dropTarget?.data;
  const sourceData = source?.data;

  if (!isTreeDropTargetData(targetData)) return undefined;

  const allDropTargets = location?.current?.dropTargets;

  // Overlapping parent/child hitboxes give the parent the grouping instruction.
  if (allDropTargets?.length > 1 && targetData.isGroup) return "make-child";

  if (!isTreeDropTargetData(sourceData)) return undefined;

  let instruction = extractInstruction(targetData)?.type;

  // A blocked instruction falls back according to the target hierarchy.
  if (instruction === "instruction-blocked") {
    instruction = targetData.isChild ? "reorder-above" : "make-child";
  }

  // Keep groups at the top level.
  if (instruction === "make-child" && sourceData.isGroup) instruction = "reorder-above";

  return instruction;
};

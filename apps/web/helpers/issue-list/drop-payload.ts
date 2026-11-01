/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { extractInstruction } from "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item";
import type { IPragmaticDropPayload } from "@plane/types";

export type GroupDropLocation = {
  columnId?: string;
  groupId: string;
  subGroupId?: string;
  id: string | undefined;
  canAddIssueBelow?: boolean;
};

function readColumn(data: Readonly<Record<string | symbol, unknown>> | undefined) {
  if (!data || typeof data.groupId !== "string" || !data.groupId) return undefined;
  if (data.subGroupId != null && typeof data.subGroupId !== "string") return undefined;
  if (data.columnId != null && typeof data.columnId !== "string") return undefined;
  return {
    groupId: data.groupId,
    subGroupId: typeof data.subGroupId === "string" ? data.subGroupId : undefined,
    columnId: typeof data.columnId === "string" ? data.columnId : undefined,
  };
}

export function getSourceFromDropPayload({ location, source }: IPragmaticDropPayload): GroupDropLocation | undefined {
  const targets = location?.initial?.dropTargets ?? [];
  const column = readColumn(targets.findLast((target) => target.data?.type === "COLUMN")?.data);
  const id = source.data?.id;
  if (!column || typeof id !== "string" || !id) return undefined;
  return { ...column, id };
}

export function getDestinationFromDropPayload({ location }: IPragmaticDropPayload): GroupDropLocation | undefined {
  const targets = location?.current?.dropTargets ?? [];
  const column = readColumn(
    targets.findLast((target) => target.data?.type === "COLUMN" || target.data?.type === "DELETE")?.data
  );
  const issue = targets.findLast((target) => target.data?.type === "ISSUE")?.data;
  if (!column || (issue && (typeof issue.id !== "string" || !issue.id))) return undefined;
  return {
    ...column,
    id: typeof issue?.id === "string" ? issue.id : undefined,
    canAddIssueBelow: issue ? extractInstruction(issue)?.type === "reorder-below" : false,
  };
}

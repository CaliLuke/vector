/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EStartOfTheWeek } from "@plane/types";
import type { TIssuePriorities } from "@plane/types";

export function selectionText(value: unknown): string {
  if (typeof value !== "string") throw new Error("Invalid command selection");
  return value;
}

export function selectionEntityId(value: unknown): string {
  if (value && typeof value === "object" && "id" in value && typeof value.id === "string" && value.id) return value.id;
  throw new Error("Invalid entity selection");
}

export function selectionWorkspaceSlug(value: unknown): string {
  if (value && typeof value === "object" && "slug" in value && typeof value.slug === "string" && value.slug)
    return value.slug;
  throw new Error("Invalid workspace selection");
}

export function selectionPriority(value: unknown): TIssuePriorities {
  switch (value) {
    case "urgent":
    case "high":
    case "medium":
    case "low":
    case "none":
      return value;
    default:
      throw new Error("Invalid priority selection");
  }
}

export function selectionWeekStart(value: unknown): EStartOfTheWeek {
  const selected = Object.values(EStartOfTheWeek).find(
    (day): day is EStartOfTheWeek => typeof day === "number" && day === value
  );
  if (selected === undefined) throw new Error("Invalid week start selection");
  return selected;
}

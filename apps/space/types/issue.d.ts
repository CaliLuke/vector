/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TIssuePriorities } from "@plane/types";
export type TIssueLayout = "list" | "kanban" | "calendar" | "spreadsheet" | "gantt";
export type TIssueFilterKeys = "priority" | "state" | "labels";
export type TFilters = { state: string[]; priority: TIssuePriorities[]; labels: string[] };
export type IPeekMode = "side" | "modal" | "full";
export interface IIssueFilterOptions {
  state?: string[] | null;
  labels?: string[] | null;
  priority?: string[] | null;
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { EIssuesStoreType } from "@plane/types";
import type { TIssueKanbanFilters } from "@plane/types";
import { storage } from "@/lib/local-storage";
import type { ParentDomain } from "@/queries/cycle-modules";
type ColumnScope = { domain: ParentDomain; accountId: string; workspaceSlug: string; parentId: string };
function storedRecords(): unknown[] {
  const raw = storage.get("issue_local_filters");
  if (!raw) return [];
  try {
    const value: unknown = JSON.parse(raw);
    return Array.isArray(value) ? value : [];
  } catch (error) {
    console.warn("Issue column preferences could not be read.", error);
    return [];
  }
}
function matches(row: unknown, scope: ColumnScope) {
  return Boolean(
    row &&
    typeof row === "object" &&
    "key" in row &&
    row.key === (scope.domain === "cycle" ? EIssuesStoreType.CYCLE : EIssuesStoreType.MODULE) &&
    "workspaceSlug" in row &&
    row.workspaceSlug === scope.workspaceSlug &&
    "viewId" in row &&
    row.viewId === scope.parentId &&
    "userId" in row &&
    row.userId === scope.accountId
  );
}
function stringArray(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === "string") : [];
}
export function readParentIssueColumns(scope: ColumnScope): TIssueKanbanFilters {
  const row = storedRecords().find((item) => matches(item, scope));
  if (
    !row ||
    typeof row !== "object" ||
    !("filters" in row) ||
    !row.filters ||
    typeof row.filters !== "object" ||
    !("kanban_filters" in row.filters)
  )
    return { group_by: [], sub_group_by: [] };
  const columns = row.filters.kanban_filters;
  if (!columns || typeof columns !== "object") return { group_by: [], sub_group_by: [] };
  return {
    group_by: stringArray("group_by" in columns ? columns.group_by : undefined),
    sub_group_by: stringArray("sub_group_by" in columns ? columns.sub_group_by : undefined),
  };
}
export function saveParentIssueColumns(scope: ColumnScope, columns: TIssueKanbanFilters) {
  const records = storedRecords(),
    index = records.findIndex((row) => matches(row, scope));
  const previous = index < 0 ? undefined : records[index];
  const base = previous && typeof previous === "object" ? previous : {};
  const filters = "filters" in base && base.filters && typeof base.filters === "object" ? base.filters : {};
  const next = {
    ...base,
    key: scope.domain === "cycle" ? EIssuesStoreType.CYCLE : EIssuesStoreType.MODULE,
    workspaceSlug: scope.workspaceSlug,
    viewId: scope.parentId,
    userId: scope.accountId,
    filters: { ...filters, kanban_filters: columns },
  };
  storage.set(
    "issue_local_filters",
    index < 0 ? [...records, next] : records.map((row, i) => (i === index ? next : row))
  );
}

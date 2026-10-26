/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EIssueFilterType } from "@plane/constants";
import type { EIssuesStoreType, IIssueFilters, IIssueFiltersResponse, TIssueKanbanFilters } from "@plane/types";
import { storage } from "@/lib/local-storage";

type LocalIssueFilters = {
  key: EIssuesStoreType;
  workspaceSlug: string;
  viewId: string | undefined;
  userId: string | undefined;
  filters: Partial<IIssueFiltersResponse & { kanban_filters: TIssueKanbanFilters }>;
};

function isLocalIssueFilters(value: unknown): value is LocalIssueFilters {
  if (!isRecord(value)) return false;
  return (
    typeof value.key === "string" &&
    typeof value.workspaceSlug === "string" &&
    (typeof value.viewId === "string" || value.viewId === undefined) &&
    (typeof value.userId === "string" || value.userId === undefined) &&
    isRecord(value.filters)
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readRecords(): unknown[] {
  const stored = storage.get("issue_local_filters");
  if (!stored) return [];
  try {
    const records: unknown = JSON.parse(stored);
    return Array.isArray(records) ? records : [];
  } catch {
    return [];
  }
}

function matchesLocalIssueFilters(
  value: unknown,
  key: EIssuesStoreType,
  workspaceSlug: string,
  viewId: string | undefined,
  userId: string | undefined
): value is LocalIssueFilters {
  return (
    isLocalIssueFilters(value) &&
    value.key === key &&
    value.workspaceSlug === workspaceSlug &&
    value.viewId === viewId &&
    value.userId === userId
  );
}

export function readLocalIssueFilters(
  key: EIssuesStoreType,
  workspaceSlug: string,
  viewId: string | undefined,
  userId: string | undefined
): Partial<IIssueFiltersResponse & { kanban_filters: TIssueKanbanFilters }> | undefined {
  const record = readRecords().find((value) => matchesLocalIssueFilters(value, key, workspaceSlug, viewId, userId));
  return isLocalIssueFilters(record) ? record.filters : undefined;
}

export function writeLocalIssueFilters(
  key: EIssuesStoreType,
  filterType: EIssueFilterType,
  workspaceSlug: string,
  viewId: string | undefined,
  userId: string | undefined,
  patch: Partial<IIssueFiltersResponse & { kanban_filters: TIssueKanbanFilters }>
) {
  const records = readRecords();
  const index = records.findIndex((record) => matchesLocalIssueFilters(record, key, workspaceSlug, viewId, userId));
  const previousRecord = index < 0 ? undefined : records[index];
  const previous = isLocalIssueFilters(previousRecord) ? previousRecord : undefined;
  const filterKey =
    filterType === EIssueFilterType.FILTERS
      ? "rich_filters"
      : filterType === EIssueFilterType.DISPLAY_FILTERS
        ? "display_filters"
        : filterType === EIssueFilterType.DISPLAY_PROPERTIES
          ? "display_properties"
          : "kanban_filters";
  const next: LocalIssueFilters = {
    key,
    workspaceSlug,
    viewId,
    userId,
    filters: { ...previous?.filters, [filterKey]: patch[filterKey] },
  };
  if (index < 0) records.push(next);
  else records[index] = next;
  storage.set("issue_local_filters", JSON.stringify(records));
}

export function writeLocalIssueFilterSnapshot(
  key: EIssuesStoreType,
  workspaceSlug: string,
  viewId: string | undefined,
  userId: string | undefined,
  filters: IIssueFilters
) {
  const records = readRecords();
  const index = records.findIndex((value) => matchesLocalIssueFilters(value, key, workspaceSlug, viewId, userId));
  const record: LocalIssueFilters = {
    key,
    workspaceSlug,
    viewId,
    userId,
    filters: {
      rich_filters: filters.richFilters,
      display_filters: filters.displayFilters,
      display_properties: filters.displayProperties,
      kanban_filters: filters.kanbanFilters,
    },
  };
  if (index < 0) records.push(record);
  else records[index] = record;
  storage.set("issue_local_filters", JSON.stringify(records));
}

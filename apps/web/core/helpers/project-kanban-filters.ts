/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EIssuesStoreType } from "@plane/types";
import type { TIssueKanbanFilters } from "@plane/types";
import { storage } from "@/lib/local-storage";

type StoredIssueLocalFilters = {
  key: string;
  workspaceSlug: string;
  viewId?: string;
  userId?: string;
  filters?: Record<string, unknown>;
};

const EMPTY_KANBAN_FILTERS: TIssueKanbanFilters = { group_by: [], sub_group_by: [] };
const PROJECT_ISSUE_STORE_KEY: string = EIssuesStoreType.PROJECT;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isStringArray(value: unknown): value is string[] {
  return Array.isArray(value) && value.every((item) => typeof item === "string");
}

function readKanbanFilters(value: unknown): Partial<TIssueKanbanFilters> | undefined {
  if (!isRecord(value)) return undefined;
  return {
    ...(isStringArray(value.group_by) ? { group_by: value.group_by } : {}),
    ...(isStringArray(value.sub_group_by) ? { sub_group_by: value.sub_group_by } : {}),
  };
}

function readStoredFilters(): StoredIssueLocalFilters[] {
  const value = storage.get("issue_local_filters");
  if (!value) return [];

  try {
    const parsed: unknown = JSON.parse(value);
    if (!Array.isArray(parsed)) return [];
    return parsed.flatMap((entry) => {
      const parsedEntry = parseStoredIssueLocalFiltersEntry(entry);
      return parsedEntry ? [parsedEntry] : [];
    });
  } catch {
    return [];
  }
}

function parseStoredIssueLocalFiltersEntry(value: unknown): StoredIssueLocalFilters | undefined {
  if (!isRecord(value) || typeof value.key !== "string" || typeof value.workspaceSlug !== "string") return undefined;
  if (value.viewId !== undefined && typeof value.viewId !== "string") return undefined;
  if (value.userId !== undefined && typeof value.userId !== "string") return undefined;

  const storedFilters = isRecord(value.filters) ? value.filters : undefined;
  return {
    key: value.key,
    workspaceSlug: value.workspaceSlug,
    ...(typeof value.viewId === "string" ? { viewId: value.viewId } : {}),
    ...(typeof value.userId === "string" ? { userId: value.userId } : {}),
    ...(storedFilters ? { filters: storedFilters } : {}),
  };
}

function normalizeKanbanFilters(filters: Partial<TIssueKanbanFilters> | undefined): TIssueKanbanFilters {
  return {
    group_by: Array.isArray(filters?.group_by) ? filters.group_by : [],
    sub_group_by: Array.isArray(filters?.sub_group_by) ? filters.sub_group_by : [],
  };
}

export function getProjectKanbanFilters(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  userId: string | undefined
): TIssueKanbanFilters {
  if (!workspaceSlug || !projectId || !userId) return { ...EMPTY_KANBAN_FILTERS };
  const stored = readStoredFilters().find(
    (entry) =>
      entry.key === PROJECT_ISSUE_STORE_KEY &&
      entry.workspaceSlug === workspaceSlug &&
      entry.viewId === projectId &&
      entry.userId === userId
  );
  return normalizeKanbanFilters(readKanbanFilters(stored?.filters?.kanban_filters));
}

export function saveProjectKanbanFilters(
  workspaceSlug: string,
  projectId: string,
  userId: string,
  update: Partial<TIssueKanbanFilters>
): TIssueKanbanFilters {
  const storedFilters = readStoredFilters();
  const entryIndex = storedFilters.findIndex(
    (entry) =>
      entry.key === PROJECT_ISSUE_STORE_KEY &&
      entry.workspaceSlug === workspaceSlug &&
      entry.viewId === projectId &&
      entry.userId === userId
  );
  const current = normalizeKanbanFilters(readKanbanFilters(storedFilters[entryIndex]?.filters?.kanban_filters));
  const kanbanFilters = normalizeKanbanFilters({ ...current, ...update });
  const entry = storedFilters[entryIndex];

  if (entry) {
    storedFilters[entryIndex] = {
      ...entry,
      filters: { ...entry.filters, kanban_filters: kanbanFilters },
    };
  } else {
    storedFilters.push({
      key: PROJECT_ISSUE_STORE_KEY,
      workspaceSlug,
      viewId: projectId,
      userId,
      filters: { kanban_filters: kanbanFilters },
    });
  }

  storage.set("issue_local_filters", JSON.stringify(storedFilters));
  return kanbanFilters;
}

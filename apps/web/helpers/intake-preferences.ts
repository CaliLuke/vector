/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { EPastDurationFilters } from "@plane/constants";
import type { TInboxIssue, TInboxIssueFilter, TInboxIssueSorting, TInboxIssueCurrentTab } from "@plane/types";
import { EInboxIssueCurrentTab, EInboxIssueStatus } from "@plane/types";
import { getCustomDates } from "@plane/utils/intake";

export type IntakePreferences = {
  tab: TInboxIssueCurrentTab;
  filters: Partial<TInboxIssueFilter>;
  sorting: Partial<TInboxIssueSorting>;
};

export function defaultIntakePreferences(tab: TInboxIssueCurrentTab = EInboxIssueCurrentTab.OPEN): IntakePreferences {
  return {
    tab,
    filters: {
      status:
        tab === EInboxIssueCurrentTab.OPEN
          ? [EInboxIssueStatus.PENDING]
          : [EInboxIssueStatus.ACCEPTED, EInboxIssueStatus.DECLINED, EInboxIssueStatus.DUPLICATE],
    },
    sorting: { order_by: "issue__created_at", sort_by: "desc" },
  };
}

function resolveDateFilter(value: string) {
  const duration = Object.values(EPastDurationFilters).find((candidate) => candidate.valueOf() === value);
  return duration === undefined ? value : getCustomDates(duration);
}

export function intakeQueryParams(preferences: IntakePreferences) {
  const params: Record<string, string> = {};
  for (const [key, values] of Object.entries(preferences.filters)) {
    if (!values?.length) continue;
    params[key] =
      key === "created_at" || key === "updated_at"
        ? values.map((value) => resolveDateFilter(String(value))).join(",")
        : values.join(",");
  }
  const { order_by = "issue__created_at", sort_by = "desc" } = preferences.sorting;
  params.order_by = `${sort_by === "desc" ? "-" : ""}${order_by}`;
  return params;
}

export function selectIntakeRows(rows: readonly TInboxIssue[], preferences: IntakePreferences, now: number) {
  const statuses = preferences.filters.status ?? [];
  const open = preferences.tab === EInboxIssueCurrentTab.OPEN;
  const selected = statuses.filter((status) =>
    open
      ? status === EInboxIssueStatus.PENDING || status === EInboxIssueStatus.SNOOZED
      : status === EInboxIssueStatus.ACCEPTED ||
        status === EInboxIssueStatus.DECLINED ||
        status === EInboxIssueStatus.DUPLICATE
  );
  return [...new Map(rows.map((row) => [row.issue.id, row])).values()].filter((row) => {
    if (!open) return selected.includes(row.status);
    if (selected.length === 2)
      return row.status === EInboxIssueStatus.PENDING || row.status === EInboxIssueStatus.SNOOZED;
    const futureSnooze =
      row.status === EInboxIssueStatus.SNOOZED && !!row.snoozed_till && new Date(row.snoozed_till).getTime() > now;
    return selected.includes(EInboxIssueStatus.SNOOZED)
      ? futureSnooze
      : selected.includes(EInboxIssueStatus.PENDING) &&
          (row.status === EInboxIssueStatus.PENDING || (row.status === EInboxIssueStatus.SNOOZED && !futureSnooze));
  });
}

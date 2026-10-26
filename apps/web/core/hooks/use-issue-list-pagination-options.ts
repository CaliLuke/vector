/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IIssueDisplayFilterOptions, IssuePaginationOptions } from "@plane/types";
import { EIssueGroupByToServerOptions } from "@plane/constants";
import { EIssueLayoutTypes, EIssuesStoreType } from "@plane/types";
import { getCalendarState } from "@/components/issues/issue-layouts/calendar/calendar-state";
import { useCurrentUserStartOfWeek } from "@/hooks/use-current-user-profile";
import { useCalendarState } from "@/providers/issue-layout-ui-state";

function getCalendarPagination(
  displayFilters: IIssueDisplayFilterOptions | undefined,
  startDate: string | undefined,
  endDate: string | undefined
): IssuePaginationOptions | undefined {
  if (!startDate || !endDate) return undefined;

  const layout = displayFilters?.calendar?.layout ?? "month";
  return {
    canGroup: true,
    perPageCount: layout === "month" ? 4 : 30,
    before: endDate,
    after: startDate,
    groupedBy: EIssueGroupByToServerOptions["target_date"],
  };
}

export function getIssueListPaginationOptions(
  storeType: EIssuesStoreType,
  displayFilters: IIssueDisplayFilterOptions | undefined,
  calendarState: ReturnType<typeof getCalendarState>,
  layoutOverride?: EIssueLayoutTypes
) {
  if (storeType === EIssuesStoreType.GLOBAL) return ungroupedPagination();
  const layout = layoutOverride ?? displayFilters?.layout ?? EIssueLayoutTypes.LIST;
  if (layout === EIssueLayoutTypes.CALENDAR) return calendarPagination(displayFilters, calendarState);
  if (layout === EIssueLayoutTypes.KANBAN) return kanbanPagination(displayFilters);
  if (layout === EIssueLayoutTypes.LIST) return listPagination(displayFilters);
  return ungroupedPagination();
}

function ungroupedPagination(): IssuePaginationOptions {
  return { canGroup: false, perPageCount: 100 };
}

function calendarPagination(
  displayFilters: IIssueDisplayFilterOptions | undefined,
  calendarState: ReturnType<typeof getCalendarState>
) {
  const range =
    displayFilters?.calendar?.layout === "week" ? calendarState.weekDateRange : calendarState.monthDateRange;
  return getCalendarPagination(displayFilters, range?.startDate, range?.endDate);
}

function kanbanPagination(displayFilters: IIssueDisplayFilterOptions | undefined): IssuePaginationOptions {
  return { canGroup: true, perPageCount: displayFilters?.sub_group_by ? 10 : 30 };
}

function listPagination(displayFilters: IIssueDisplayFilterOptions | undefined): IssuePaginationOptions {
  return { canGroup: true, perPageCount: displayFilters?.group_by ? 50 : 100 };
}

/** Resolve the exact page shape used by the active issue layout. */
export function useIssueListPaginationOptions(
  storeType: EIssuesStoreType,
  displayFilters: IIssueDisplayFilterOptions | undefined,
  layoutOverride?: EIssueLayoutTypes
) {
  const { activeMonthDate, activeWeekDate } = useCalendarState();
  const startOfWeek = useCurrentUserStartOfWeek();
  const calendarState = getCalendarState(activeMonthDate, activeWeekDate, startOfWeek);
  return getIssueListPaginationOptions(storeType, displayFilters, calendarState, layoutOverride);
}

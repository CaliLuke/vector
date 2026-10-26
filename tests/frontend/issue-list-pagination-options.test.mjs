import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { getCalendarState } =
  await import("../../apps/web/core/components/issues/issue-layouts/calendar/calendar-state.ts");
const { getIssueListPaginationOptions } =
  await import("../../apps/web/core/hooks/use-issue-list-pagination-options.ts");
const { EIssueLayoutTypes, EIssuesStoreType } = await import("../../packages/types/dist/index.mjs");

const calendarState = getCalendarState(new Date(2026, 9, 26), new Date(2026, 9, 26), 1);

await test("list and Kanban pagination match their layout page sizes and grouping inputs", () => {
  assert.deepEqual(
    getIssueListPaginationOptions(
      EIssuesStoreType.PROJECT,
      { group_by: "state" },
      calendarState,
      EIssueLayoutTypes.LIST
    ),
    { canGroup: true, perPageCount: 50 }
  );
  assert.deepEqual(
    getIssueListPaginationOptions(
      EIssuesStoreType.PROJECT,
      { sub_group_by: "priority" },
      calendarState,
      EIssueLayoutTypes.KANBAN
    ),
    { canGroup: true, perPageCount: 10 }
  );
  assert.deepEqual(
    getIssueListPaginationOptions(EIssuesStoreType.PROJECT, {}, calendarState, EIssueLayoutTypes.KANBAN),
    { canGroup: true, perPageCount: 30 }
  );
});

await test("calendar count uses the selected date range and layout pagination", () => {
  const month = getIssueListPaginationOptions(
    EIssuesStoreType.PROJECT,
    { calendar: { layout: "month" } },
    calendarState,
    EIssueLayoutTypes.CALENDAR
  );
  const week = getIssueListPaginationOptions(
    EIssuesStoreType.PROJECT,
    { calendar: { layout: "week" } },
    calendarState,
    EIssueLayoutTypes.CALENDAR
  );

  assert.deepEqual(month, {
    canGroup: true,
    perPageCount: 4,
    before: calendarState.monthDateRange.endDate,
    after: calendarState.monthDateRange.startDate,
    groupedBy: "target_date",
  });
  assert.deepEqual(week, {
    canGroup: true,
    perPageCount: 30,
    before: calendarState.weekDateRange.endDate,
    after: calendarState.weekDateRange.startDate,
    groupedBy: "target_date",
  });
});

await test("spreadsheet and workspace-global reads keep their existing ungrouped page shape", () => {
  assert.deepEqual(
    getIssueListPaginationOptions(
      EIssuesStoreType.PROJECT,
      { group_by: "state" },
      calendarState,
      EIssueLayoutTypes.SPREADSHEET
    ),
    { canGroup: false, perPageCount: 100 }
  );
  assert.deepEqual(
    getIssueListPaginationOptions(
      EIssuesStoreType.GLOBAL,
      { layout: EIssueLayoutTypes.CALENDAR, calendar: { layout: "week" } },
      calendarState
    ),
    { canGroup: false, perPageCount: 100 }
  );
});

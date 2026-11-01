import assert from "node:assert/strict";
import test from "node:test";
import { runInNewContext } from "node:vm";
import {
  calculateTimeAgo,
  getDate,
  renderFormattedDate,
  isValidDate,
  generateDateArray,
  parseDateFilter,
} from "../../packages/utils/src/datetime.ts";

void test("relative dates accept epoch milliseconds and dates without throwing for invalid input", (t) => {
  t.mock.timers.enable({ apis: ["Date"], now: new Date("2026-10-31T12:00:00Z") });
  const earlier = "2026-10-29T12:00:00Z";
  assert.equal(calculateTimeAgo(earlier), "2 days ago");
  assert.equal(calculateTimeAgo(new Date(earlier)), "2 days ago");
  assert.equal(calculateTimeAgo(Date.parse(earlier)), "2 days ago");
  assert.notEqual(calculateTimeAgo(0), "");
  for (const invalid of [null, "", "invalid", NaN, new Date(NaN)]) assert.equal(calculateTimeAgo(invalid), "");
});

void test("calendar dates preserve their local day and formatting supports optional input", () => {
  const parsed = getDate("2024-01-01");
  assert.deepEqual([parsed.getFullYear(), parsed.getMonth(), parsed.getDate(), parsed.getHours()], [2024, 0, 1, 0]);
  assert.equal(renderFormattedDate("2024-01-01"), "Jan 01, 2024");
  assert.equal(renderFormattedDate("2024-01-01", "yyyy-MM-dd"), "2024-01-01");
  for (const value of [null, undefined, "", "invalid"]) assert.equal(renderFormattedDate(value), undefined);
});

void test("date validation rejects coercible objects and accepts valid cross-realm dates", () => {
  assert.equal(isValidDate("2026-10-31"), true);
  assert.equal(isValidDate(new Date(0)), true);
  assert.equal(isValidDate(runInNewContext("new Date(0)")), true);
  assert.equal(isValidDate(new Date(NaN)), false);
  assert.equal(isValidDate("invalid"), false);
  assert.equal(isValidDate({ toString: () => "2026-10-31" }), false);
  assert.equal(isValidDate(null), false);
  assert.equal(isValidDate(0), false);
});
void test("date sequences preserve the two-day trailing window without mutating input dates", () => {
  const start = new Date("2026-10-01T12:00:00Z");
  const end = new Date("2026-10-02T12:00:00Z");
  assert.deepEqual(generateDateArray(start, end), [
    { date: "2026-10-01" },
    { date: "2026-10-02" },
    { date: "2026-10-03" },
    { date: "2026-10-04" },
  ]);
  assert.equal(start.toISOString(), "2026-10-01T12:00:00.000Z");
  assert.equal(end.toISOString(), "2026-10-02T12:00:00.000Z");
  assert.deepEqual(generateDateArray("invalid", end), []);
});

void test("date filters validate their comparator and ignore malformed dates", (t) => {
  t.mock.timers.enable({ apis: ["Date"], now: new Date(2026, 9, 31, 12) });
  assert.deepEqual(parseDateFilter("2026-10-31;before"), { type: "before", date: new Date("2026-10-31") });
  const relative = parseDateFilter("1_weeks;after;fromnow");
  assert.equal(relative.type, "after");
  assert.deepEqual(relative.date, new Date(2026, 10, 7, 12));
  for (const invalid of [
    "",
    "2026-10-31",
    "2026-10-31;invalid",
    "invalid;before",
    "1_unknown;after",
    "x_days;before",
  ]) {
    assert.equal(parseDateFilter(invalid), undefined);
  }
});

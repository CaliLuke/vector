import * as ContractEnums from "../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { weekView } = await import("../../apps/web/core/components/gantt-chart/views/week-view.ts");
const { monthView } = await import("../../apps/web/core/components/gantt-chart/views/month-view.ts");
const { quarterView } = await import("../../apps/web/core/components/gantt-chart/views/quarter-view.ts");
const date = (value) => new Date(`${value}T00:00:00`);
const day = (value) => [value.getFullYear(), value.getMonth() + 1, value.getDate()];
const chart = (range) => ({
  data: {
    currentDate: date("2025-01-15"),
    startDate: date("2025-01-01"),
    endDate: date("2025-01-31"),
    approxFilterRange: range,
    dayWidth: 10,
  },
});

for (const { name, view } of [
  { name: "week", view: weekView },
  { name: "month", view: monthView },
]) {
  void test(`${name} charts extend one edge and report the added segment width without mutating the source`, () => {
    const original = chart(1);
    const before = structuredClone(original);
    const initial = view.generateChart(original, null);
    assert.deepEqual(day(initial.state.data.startDate), [2024, 12, 15]);
    assert.deepEqual(day(initial.state.data.endDate), [2025, 2, 15]);
    assert.equal(initial.scrollWidth, 630);
    const left = view.generateChart(initial.state, "left");
    assert.deepEqual(day(left.state.data.startDate), [2024, 10, 27]);
    assert.deepEqual(day(left.state.data.endDate), [2025, 2, 15]);
    assert.equal(left.scrollWidth, 490);
    const right = view.generateChart(initial.state, "right");
    assert.deepEqual(day(right.state.data.startDate), [2024, 12, 15]);
    assert.deepEqual(day(right.state.data.endDate), [2025, 3, 1]);
    assert.equal(right.scrollWidth, 140);
    const targeted = view.generateChart(initial.state, "right", date("2025-07-10"));
    assert.deepEqual(day(targeted.state.data.endDate), [2025, 8, 2]);
    const targetedLeft = view.generateChart(initial.state, "left", date("2024-09-10"));
    assert.deepEqual(day(targetedLeft.state.data.startDate), [2024, 7, 28]);
    assert.deepEqual(original, before);
    assert.deepEqual(day(initial.state.data.startDate), [2024, 12, 15]);
    const weeks = name === "month" ? initial.payload.weeks : initial.payload;
    assert.equal(weeks[0].children?.length, name === "week" ? 7 : undefined);
    assert.ok(view.generateChart(chart(0), null).scrollWidth > initial.scrollWidth);
  });
}

void test("weekly segments honor the configured start of week", () => {
  const result = weekView.generateChart(chart(1), null, undefined, ContractEnums.EStartOfTheWeek.MONDAY);
  assert.deepEqual(day(result.state.data.startDate), [2024, 12, 9]);
  assert.deepEqual(day(result.state.data.endDate), [2025, 2, 16]);
  assert.equal(result.payload[0].children[0].day, 1);
});

void test("quarter segments align whole months and extend by half the configured range", () => {
  const original = chart(2);
  const initial = quarterView.generateChart(original, null);
  assert.deepEqual(day(initial.state.data.startDate), [2024, 11, 1]);
  assert.deepEqual(day(initial.state.data.endDate), [2025, 2, 28]);
  assert.equal(initial.payload.length, 4);
  assert.equal(initial.scrollWidth, 1200);
  const left = quarterView.generateChart(initial.state, "left");
  assert.deepEqual(day(left.state.data.startDate), [2024, 10, 1]);
  assert.deepEqual(day(left.state.data.endDate), [2025, 2, 28]);
  assert.equal(left.scrollWidth, 310);
  const right = quarterView.generateChart(initial.state, "right");
  assert.deepEqual(day(right.state.data.startDate), [2024, 11, 1]);
  assert.deepEqual(day(right.state.data.endDate), [2025, 3, 31]);
  assert.equal(right.scrollWidth, 310);
  assert.deepEqual(
    day(quarterView.generateChart(initial.state, "left", date("2024-07-15")).state.data.startDate),
    [2024, 6, 1]
  );
  assert.deepEqual(
    day(quarterView.generateChart(initial.state, "right", date("2025-07-15")).state.data.endDate),
    [2025, 8, 31]
  );
  assert.equal(quarterView.generateChart(chart(0), null).payload.length, 24);
  assert.deepEqual(original, chart(2));
});

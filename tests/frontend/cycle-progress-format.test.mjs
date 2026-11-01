import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { formatActiveCycle } = await import("../../packages/utils/src/cycle.ts");

void test("cycle progress keeps recorded totals and uses finite pending counts for future dates and missing estimates", (t) => {
  t.mock.timers.enable({ apis: ["Date"], now: new Date("2026-10-03T12:00:00Z").getTime() });
  const cycle = {
    version: 2,
    start_date: "2026-10-01",
    end_date: "2026-10-05",
    progress: [
      {
        date: "2026-10-03",
        total_issues: 10,
        completed_issues: 4,
        cancelled_issues: 1,
        backlog_issues: 2,
        started_issues: 2,
        unstarted_issues: 1,
      },
    ],
  };
  const issues = formatActiveCycle({ cycle, isTypeIssue: true, isBurnDown: true });
  const today = issues.find((row) => row.date === "2026-10-03");
  assert.equal(today.pending, 5);
  assert.equal(today.actual, 5);
  assert.equal(today.completed, 4);
  const future = issues.find((row) => row.date === "2026-10-04");
  assert.equal(future.scope, 10);
  assert.equal(future.actual, undefined);
  assert.equal(future.pending, 0);
  const estimates = formatActiveCycle({ cycle, isTypeIssue: false, isBurnDown: true });
  assert.ok(estimates.every((row) => Number.isFinite(row.pending)));
  assert.equal(estimates[0].actual, 0);
});

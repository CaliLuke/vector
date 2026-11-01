import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { calculateCycleProgress } = await import("../../packages/utils/src/cycle.ts");
const { updateDistribution } = await import("../../packages/utils/src/distribution-update.ts");
const { getComputedDisplayProperties } = await import("../../packages/utils/src/work-item/base.ts");

await test("cycle progress selects issue or point counts and honors the current snapshot", () => {
  const cycle = {
    total_issues: 10,
    completed_issues: 3,
    cancelled_issues: 2,
    started_issues: 1,
    total_estimate_points: 20,
    completed_estimate_points: 6,
    cancelled_estimate_points: 5,
    started_estimate_points: 3,
  };
  assert.equal(calculateCycleProgress(cycle), 38);
  assert.equal(calculateCycleProgress(cycle, "issues", true), 50);
  assert.equal(calculateCycleProgress(cycle, "points"), 40);
  assert.equal(calculateCycleProgress(cycle, "points", true), 60);
  assert.equal(calculateCycleProgress({ ...cycle, progress_snapshot: { ...cycle, completed_issues: 8 } }), 100);
  assert.equal(calculateCycleProgress({ ...cycle, progress_snapshot: {} }), 38);
  assert.equal(calculateCycleProgress(undefined), 0);
  assert.equal(calculateCycleProgress({ total_issues: 0 }), 0);
  assert.equal(calculateCycleProgress({ total_issues: 1, cancelled_issues: 2 }), 0);
  assert.equal(calculateCycleProgress({ total_issues: 3, completed_issues: -1 }), 0);
  assert.equal(calculateCycleProgress({ total_issues: 3, completed_issues: 4 }), 100);
});

await test("distribution deltas update matching issue and point rows without creating missing projections", () => {
  const countRow = { completed_issues: 2, pending_issues: 3, total_issues: 5 };
  const pointRow = { completed_estimates: 4, pending_estimates: 6, total_estimates: 10 };
  const distribution = {
    total_issues: 5,
    distribution: {
      assignees: [
        { ...countRow, assignee_id: "member" },
        { ...countRow, assignee_id: "untouched" },
      ],
      labels: [{ ...countRow, label_id: "label" }],
      completion_chart: { "2026-10-31": null },
    },
    estimate_distribution: {
      assignees: [{ ...pointRow, assignee_id: "member" }],
      labels: [{ ...pointRow, label_id: "label" }],
    },
  };
  updateDistribution(distribution, {
    pathUpdates: [
      { path: ["total_issues"], value: -1 },
      { path: ["missing"], value: 8 },
      { path: ["distribution", "completion_chart", "2026-10-31"], value: 1 },
    ],
    assigneeUpdates: [
      {
        id: "member",
        completed_issues: 1,
        pending_issues: -1,
        total_issues: 0,
        completed_estimates: 2,
        pending_estimates: -2,
        total_estimates: 0,
      },
      { id: "absent", total_issues: 1, total_estimates: 2 },
    ],
    labelUpdates: [{ id: "label", pending_issues: -1, total_issues: -1, pending_estimates: -2, total_estimates: -2 }],
  });
  assert.equal(distribution.total_issues, 4);
  assert.equal(Object.hasOwn(distribution, "missing"), false);
  assert.equal(distribution.distribution.completion_chart["2026-10-31"], 1);
  assert.deepEqual(distribution.distribution.assignees, [
    { assignee_id: "member", completed_issues: 3, pending_issues: 2, total_issues: 5 },
    { assignee_id: "untouched", ...countRow },
  ]);
  assert.deepEqual(distribution.estimate_distribution.assignees, [
    { assignee_id: "member", completed_estimates: 6, pending_estimates: 4, total_estimates: 10 },
  ]);
  assert.deepEqual(distribution.distribution.labels, [
    { label_id: "label", completed_issues: 2, pending_issues: 2, total_issues: 4 },
  ]);
  assert.deepEqual(distribution.estimate_distribution.labels, [
    { label_id: "label", completed_estimates: 4, pending_estimates: 4, total_estimates: 8 },
  ]);
  const missing = {};
  updateDistribution(missing, {
    pathUpdates: [],
    assigneeUpdates: [{ id: "member", total_issues: 1, total_estimates: 1 }],
    labelUpdates: [],
  });
  assert.deepEqual(missing, {});
});

await test("display preferences keep false selections and valid custom fields while supplying standard defaults", () => {
  const preferences = {
    assignee: false,
    state: undefined,
    customproperty_score: false,
    customproperty_flag: true,
    customproperty_invalid: "true",
    unknown: true,
  };
  const computed = getComputedDisplayProperties(preferences);
  assert.equal(computed.assignee, false);
  assert.equal(computed.state, true);
  assert.equal(computed.issue_type, true);
  assert.equal(computed.customproperty_score, false);
  assert.equal(computed.customproperty_flag, true);
  assert.equal(Object.hasOwn(computed, "customproperty_invalid"), false);
  assert.equal(Object.hasOwn(computed, "unknown"), false);
  assert.equal(preferences.state, undefined);
});

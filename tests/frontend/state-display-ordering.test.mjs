import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { projectGroupedIssueIdsByDisplayOrder, projectIssueIdsByDisplayOrder } =
  await import("../../apps/web/core/components/issues/issue-layouts/issue-display-ordering.ts");

const issues = {
  beta: { id: "beta", project_id: "p1", state_id: "s2", created_at: "2026-10-01T00:00:00Z" },
  alpha: { id: "alpha", project_id: "p1", state_id: "s1", created_at: "2026-10-02T00:00:00Z" },
  missing: { id: "missing", project_id: "p1", state_id: "gone", created_at: "2026-10-03T00:00:00Z" },
};
const states = {
  workspaceSlug: "acme",
  projectId: "p1",
  scopeKey: "states/p1",
  statesById: {
    s1: { id: "s1", name: "Alpha" },
    s2: { id: "s2", name: "Beta" },
  },
  isCurrent: () => true,
};

await test("state name display ordering preserves the legacy empty-last and direction rules", () => {
  assert.deepEqual(
    projectIssueIdsByDisplayOrder(
      ["beta", "alpha", "missing"],
      (id) => issues[id],
      undefined,
      "state__name",
      undefined,
      undefined,
      states
    ),
    ["alpha", "beta", "missing"]
  );
  assert.deepEqual(
    projectIssueIdsByDisplayOrder(
      ["beta", "alpha", "missing"],
      (id) => issues[id],
      undefined,
      "-state__name",
      undefined,
      undefined,
      states
    ),
    ["beta", "alpha", "missing"]
  );
});

await test("state name ordering defers when the native state snapshot is stale", () => {
  const staleStates = { ...states, isCurrent: () => false };
  const issueIds = ["beta", "alpha"];

  assert.equal(
    projectIssueIdsByDisplayOrder(
      issueIds,
      (id) => issues[id],
      undefined,
      "state__name",
      undefined,
      undefined,
      staleStates
    ),
    issueIds
  );
  assert.deepEqual(
    projectGroupedIssueIdsByDisplayOrder(
      { all: issueIds },
      (id) => issues[id],
      undefined,
      "state__name",
      undefined,
      undefined,
      staleStates
    ),
    { all: issueIds }
  );
});

import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const { getGroupedIssueIds, projectGroupedIssueIdsByDisplayOrder, projectIssueIdsByDisplayOrder } =
  await import("../../apps/web/core/components/issues/issue-layouts/issue-display-ordering.ts");

function issue(id, assigneeIds, createdAt = "2026-10-25T00:00:00Z") {
  return { id, assignee_ids: assigneeIds, created_at: createdAt };
}

function roster(firstNames) {
  return new Map(Object.entries(firstNames).map(([id, first_name]) => [id, { member: { id, first_name } }]));
}

await test("assignee name ordering preserves multi-name array comparison and leaves empty assignees last", () => {
  const issues = new Map([
    ["multi", issue("multi", ["bob", "alice"])],
    ["carol", issue("carol", ["carol"])],
    ["unknown", issue("unknown", ["missing-member"])],
    ["empty", issue("empty", [])],
  ]);
  const ids = ["multi", "carol", "unknown", "empty"];
  const getIssue = (id) => issues.get(id);
  const membersById = roster({ bob: "Bob", alice: "Alice", carol: "Carol" });

  assert.deepEqual(projectIssueIdsByDisplayOrder(ids, getIssue, membersById, "assignees__first_name"), [
    "unknown",
    "multi",
    "carol",
    "empty",
  ]);
  assert.deepEqual(projectIssueIdsByDisplayOrder(ids, getIssue, membersById, "-assignees__first_name"), [
    "carol",
    "multi",
    "unknown",
    "empty",
  ]);
});

await test("assignee name ordering keeps created-at ties stable and preserves unavailable issue IDs", () => {
  const issues = new Map([
    ["first", issue("first", ["alice"], "2026-10-25T00:00:00Z")],
    ["second", issue("second", ["alice"], "2026-10-25T00:00:00Z")],
    ["newer", issue("newer", ["bob"], "2026-10-26T00:00:00Z")],
  ]);

  assert.deepEqual(
    projectIssueIdsByDisplayOrder(
      ["first", "missing-1", "second", "newer", "missing-2"],
      (id) => issues.get(id),
      roster({ alice: "Alice", bob: "Bob" }),
      "assignees__first_name"
    ),
    ["first", "second", "newer", "missing-1", "missing-2"]
  );
});

await test("grouped and swimlane projections sort each visible leaf and preserve all other sort modes", () => {
  const issues = new Map([
    ["z", issue("z", ["zulu"])],
    ["a", issue("a", ["alpha"])],
    ["b", issue("b", ["beta"])],
  ]);
  const getIssue = (id) => issues.get(id);
  const membersById = roster({ zulu: "Zulu", alpha: "Alpha", beta: "Beta" });
  const grouped = { open: ["z", "a"], closed: ["b"] };
  const swimlanes = { open: { urgent: ["z", "a"], normal: ["b"] } };

  const projectedGroups = projectGroupedIssueIdsByDisplayOrder(grouped, getIssue, membersById, "assignees__first_name");
  const projectedSwimlanes = projectGroupedIssueIdsByDisplayOrder(
    swimlanes,
    getIssue,
    membersById,
    "assignees__first_name"
  );

  assert.deepEqual(projectedGroups, { open: ["a", "z"], closed: ["b"] });
  assert.deepEqual(projectedSwimlanes, { open: { urgent: ["a", "z"], normal: ["b"] } });
  assert.deepEqual(getGroupedIssueIds(projectedSwimlanes, "open", "urgent"), ["a", "z"]);
  const mixedGroups = { open: { urgent: ["z", "a"] }, closed: ["b"] };
  const projectedMixedGroups = projectGroupedIssueIdsByDisplayOrder(
    mixedGroups,
    getIssue,
    membersById,
    "assignees__first_name"
  );
  assert.deepEqual(projectedMixedGroups, { open: { urgent: ["a", "z"] }, closed: ["b"] });
  assert.deepEqual(getGroupedIssueIds(projectedMixedGroups, "closed"), ["b"]);
  assert.equal(projectGroupedIssueIdsByDisplayOrder(grouped, getIssue, membersById, "priority"), grouped);
  const flatIds = ["z", "a"];
  assert.equal(projectIssueIdsByDisplayOrder(flatIds, getIssue, membersById, "priority"), flatIds);
});

await test("label display order uses only the supplied project catalogue and does not mutate grouped IDs", () => {
  const issues = new Map([
    ["z", { ...issue("z", []), project_id: "p1", label_ids: ["z"] }],
    ["a", { ...issue("a", []), project_id: "p1", label_ids: ["a"] }],
    ["unknown", { ...issue("unknown", []), project_id: "p1", label_ids: ["foreign"] }],
    ["empty", { ...issue("empty", []), project_id: "p1", label_ids: [] }],
  ]);
  const labels = new Map(
    Object.entries({
      z: Object.freeze({ id: "z", project_id: "p1", name: "Zulu" }),
      a: Object.freeze({ id: "a", project_id: "p1", name: "Alpha" }),
      foreign: Object.freeze({ id: "foreign", project_id: "other-project", name: "SECRET" }),
    })
  );
  const grouped = Object.freeze({ open: Object.freeze(["z", "a", "unknown", "empty"]) });
  const read = (id) => issues.get(id);
  assert.deepEqual(projectGroupedIssueIdsByDisplayOrder(grouped, read, undefined, "labels__name", labels), {
    open: ["unknown", "a", "z", "empty"],
  });
  assert.deepEqual(projectIssueIdsByDisplayOrder(grouped.open, read, undefined, "-labels__name", labels), [
    "z",
    "a",
    "unknown",
    "empty",
  ]);
  assert.deepEqual(grouped.open, ["z", "a", "unknown", "empty"]);
  assert.equal(projectGroupedIssueIdsByDisplayOrder(grouped, read, undefined, "labels__name"), grouped);
});

await test("canonical label replacement and an empty catalogue immediately replace derived names", () => {
  const issues = new Map([
    ["a", { ...issue("a", [], "2026-10-25T00:00:00Z"), project_id: "p1", label_ids: ["a"] }],
    ["b", { ...issue("b", [], "2026-10-26T00:00:00Z"), project_id: "p1", label_ids: ["b"] }],
  ]);
  const groups = { high: { urgent: ["b", "a"] } };
  const read = (id) => issues.get(id);
  assert.deepEqual(
    projectGroupedIssueIdsByDisplayOrder(
      groups,
      read,
      undefined,
      "labels__name",
      new Map(
        Object.entries({
          a: { name: "Alpha", project_id: "p1" },
          b: { name: "Zulu", project_id: "p1" },
        })
      )
    ),
    { high: { urgent: ["a", "b"] } }
  );
  assert.deepEqual(
    projectGroupedIssueIdsByDisplayOrder(
      groups,
      read,
      undefined,
      "labels__name",
      new Map(
        Object.entries({
          a: { name: "Zulu", project_id: "p1" },
          b: { name: "Alpha", project_id: "p1" },
        })
      )
    ),
    { high: { urgent: ["b", "a"] } }
  );
  assert.deepEqual(projectGroupedIssueIdsByDisplayOrder(groups, read, undefined, "labels__name", new Map()), groups);
});

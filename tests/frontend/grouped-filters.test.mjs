import assert from "node:assert/strict";
import test from "node:test";
import {
  addFilterCondition,
  conjunctiveFilterCondition,
  displayFilterConditions,
  filterConditions,
  readFilterExpression,
  removeFilterCondition,
  updateFilterConditionOperator,
  updateFilterConditionValue,
  updateFilterGroupOperator,
  workItemFiltersAdapter,
  writeFilterExpression,
} from "../../packages/shared-state/dist/index.js";

function read(raw) {
  return readFilterExpression(workItemFiltersAdapter, raw);
}

function write(expression, snapshot) {
  const result = writeFilterExpression(
    workItemFiltersAdapter,
    expression,
    snapshot.committedConditionIds,
    snapshot.error
  );
  assert.equal(result.ok, true, result.ok ? undefined : result.error);
  return result.value;
}

await test("OR, nested NOT, bare type equality and logical casing round-trip without flattening", () => {
  const snapshot = read({
    AnD: [{ OR: [{ type_id: "a" }, { type_id__in: ["b", "c"] }] }, { NoT: { priority: "low" } }],
  });
  assert.equal(snapshot.error, null);
  assert.deepEqual(write(snapshot.expression, snapshot), {
    and: [{ or: [{ type_id__exact: "a" }, { type_id__in: ["b", "c"] }] }, { not: { priority__exact: "low" } }],
  });
});

await test("changing a group's all/any operator preserves its children", () => {
  const snapshot = read({ or: [{ priority__exact: "high" }, { state_id__exact: "active" }] });
  const next = updateFilterGroupOperator(snapshot.expression, snapshot.expression.id, "and");
  assert.deepEqual(write(next, snapshot), { and: [{ priority__exact: "high" }, { state_id__exact: "active" }] });
});

await test("leaf operators inside an excluded group do not absorb the group's negation", () => {
  const snapshot = read({ not: { or: [{ priority__in: "high" }, { state_id__in: "active" }] } });
  assert.deepEqual(
    displayFilterConditions(snapshot.expression).map((node) => node.operator),
    ["in", "in"]
  );
  const priority = filterConditions(snapshot.expression).find((condition) => condition.property === "priority");
  const excluded = updateFilterConditionOperator(snapshot.expression, priority.id, "in", true, false);
  assert.deepEqual(write(excluded, snapshot), {
    not: { or: [{ not: { priority__in: "high" } }, { state_id__in: "active" }] },
  });
  const included = updateFilterConditionOperator(excluded, priority.id, "in", false, false);
  assert.deepEqual(write(included, snapshot), { not: { or: [{ priority__in: "high" }, { state_id__in: "active" }] } });
});

await test("removing OR children or an entire group preserves surrounding clauses", () => {
  const raw = { and: [{ or: [{ priority__in: "high" }, { priority__in: "low" }] }, { not: { state_id__in: "done" } }] };
  const one = read(raw);
  const removedChild = removeFilterCondition(one.expression, one.expression.children[0].children[0].id);
  assert.deepEqual(write(removedChild.expression, one), {
    and: [{ priority__in: "low" }, { not: { state_id__in: "done" } }],
  });
  const group = read(raw);
  const removedGroup = removeFilterCondition(group.expression, group.expression.children[0].id);
  assert.deepEqual(write(removedGroup.expression, group), { not: { state_id__in: "done" } });
});

await test("removing committed null or empty predicates remains serializable as an explicit clear", () => {
  for (const raw of [{ type_id__exact: null }, { type_id__in: [] }]) {
    const snapshot = read(raw);
    const result = removeFilterCondition(snapshot.expression, snapshot.expression.id);
    assert.deepEqual(write(result.expression, snapshot), {});
  }
});

await test("adding an OR condition preserves existing AND logic as a nested group", () => {
  const snapshot = read({ and: [{ priority__exact: "high" }, { state_id__exact: "active" }] });
  const next = addFilterCondition(
    snapshot.expression,
    "or",
    { property: "type_id", operator: "exact", value: "incident" },
    false
  );
  assert.deepEqual(write(next, snapshot), {
    or: [{ and: [{ priority__exact: "high" }, { state_id__exact: "active" }] }, { type_id__exact: "incident" }],
  });
});

await test("saved type CSV tokens are never trimmed or discarded during a round trip", () => {
  for (const value of ["00000000-0000-4000-8000-000000000001,", " 00000000-0000-4000-8000-000000000001", ",,"]) {
    const snapshot = read({ type_id__in: value });
    assert.deepEqual(write(snapshot.expression, snapshot), { type_id__in: value });
  }
});

await test("new unfinished controls do not remove committed empty selections", () => {
  const snapshot = read({ label_id__in: [] });
  const next = addFilterCondition(
    snapshot.expression,
    "and",
    { property: "state_id", operator: "in", value: [] },
    false
  );
  assert.deepEqual(write(next, snapshot), { label_id__in: [] });
});

await test("sidebar restrictions add or update a conjunctive leaf without rewriting an OR branch", () => {
  const { expression: original } = read({ or: [{ assignee_id__in: "alice" }, { priority__in: "high" }] });
  const appended = addFilterCondition(
    original,
    "and",
    { property: "assignee_id", operator: "in", value: ["bob"] },
    false
  );
  assert.deepEqual(workItemFiltersAdapter.toExternal(appended), {
    and: [{ or: [{ assignee_id__in: "alice" }, { priority__in: "high" }] }, { assignee_id__in: "bob" }],
  });
  const current = read({ and: [{ not: { assignee_id__in: "alice" } }, { priority__in: "high" }] });
  const excluded = conjunctiveFilterCondition(current.expression, "assignee_id", "not_in");
  assert.ok(excluded);
  const updated = updateFilterConditionValue(current.expression, excluded.id, ["carol"], true);
  assert.deepEqual(write(updated.expression, current), {
    and: [{ not: { assignee_id__in: "carol" } }, { priority__in: "high" }],
  });
});

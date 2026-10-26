import assert from "node:assert/strict";
import test from "node:test";
import {
  addFilterCondition,
  filterConditionHasValue,
  filterHasActiveValues,
  readFilterExpression,
  updateFilterConditionValue,
  workItemFiltersAdapter,
  writeFilterExpression,
} from "../../packages/shared-state/dist/index.js";

function read(expression, adapter = workItemFiltersAdapter) {
  return readFilterExpression(adapter, expression);
}

function write(snapshot, expression = snapshot.expression, adapter = workItemFiltersAdapter) {
  return writeFilterExpression(adapter, expression, snapshot.committedConditionIds, snapshot.error);
}

await test("unrepresentable saved filters remain active and cannot be replaced by a write", () => {
  for (const raw of [
    42,
    ["invalid"],
    { priority__in: [null] },
    { unknown__exact: "kept" },
    { state_id__unknown: "kept" },
    { and: [{ priority__exact: "high" }], state_id__in: "kept" },
    { xor: [{ priority__exact: "high" }, { priority__exact: "low" }] },
    { priority__exact: { invalid: true } },
    { not: { not: { not: { not: { not: { priority__exact: "high" } } } } } },
  ]) {
    const snapshot = read(raw);
    assert.ok(snapshot.error);
    assert.equal(filterHasActiveValues(snapshot.expression, snapshot.committedConditionIds, snapshot.error), true);
    assert.deepEqual(write(snapshot), { ok: false, error: snapshot.error });
  }
});

await test("saved null and empty selections survive unrelated edits", () => {
  const snapshot = read({ and: [{ state_id__exact: null }, { label_id__in: [] }, { priority__exact: "high" }] });
  const priority = snapshot.expression.children[2];
  const edited = updateFilterConditionValue(snapshot.expression, priority.id, "low");
  assert.deepEqual(write(snapshot, edited.expression), {
    ok: true,
    value: { and: [{ state_id__exact: null }, { label_id__in: [] }, { priority__exact: "low" }] },
  });
  assert.equal(filterConditionHasValue(snapshot.expression.children[0], snapshot.committedConditionIds), true);
  assert.equal(filterConditionHasValue(snapshot.expression.children[1], snapshot.committedConditionIds), true);
});

await test("unfinished new controls are omitted without dropping committed empty selections", () => {
  const snapshot = read({ label_id__in: [] });
  const expression = addFilterCondition(
    snapshot.expression,
    "and",
    { property: "state_id", operator: "in", value: undefined },
    false
  );
  assert.deepEqual(write(snapshot, expression), { ok: true, value: { label_id__in: [] } });
});

await test("a serializer failure reports an error and keeps the source expression available", () => {
  let fail = true;
  const adapter = {
    toInternal: (raw) => workItemFiltersAdapter.toInternal(raw),
    toExternal: (expression) => {
      if (fail) throw new Error("Unsupported serialization");
      return workItemFiltersAdapter.toExternal(expression);
    },
  };
  const snapshot = read({ priority__exact: "high" }, adapter);
  const updated = updateFilterConditionValue(snapshot.expression, snapshot.expression.id, "low");
  assert.deepEqual(write(snapshot, updated.expression, adapter), {
    ok: false,
    error: "These filters could not be saved. Your previous filters are still applied.",
  });
  fail = false;
  assert.deepEqual(write(snapshot, updated.expression, adapter), { ok: true, value: { priority__exact: "low" } });
});

await test("explicitly clearing a malformed source creates a valid empty expression", () => {
  const malformed = read({ unsupported__exact: "kept" });
  assert.ok(malformed.error);
  const cleared = read(undefined);
  assert.equal(cleared.error, null);
  assert.deepEqual(write(cleared), { ok: true, value: {} });
});

await test("imported arrays retain their wire representation through edits", () => {
  for (const selected of [["high", "low"], [false, true], []]) {
    const snapshot = read({ and: [{ priority__in: selected }, { state_id__exact: "a" }] });
    const state = snapshot.expression.children[1];
    const edited = updateFilterConditionValue(snapshot.expression, state.id, "b");
    assert.deepEqual(write(snapshot, edited.expression), {
      ok: true,
      value: { and: [{ priority__in: selected }, { state_id__exact: "b" }] },
    });
  }
});

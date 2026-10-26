import assert from "node:assert/strict";
import test from "node:test";
import {
  displayFilterConditions,
  displayOperatorOptions,
  operatorLabel,
  readFilterExpression,
  updateFilterConditionOperator,
  updateFilterConditionValue,
  workItemFiltersAdapter,
  writeFilterExpression,
} from "../../packages/shared-state/dist/index.js";
import {
  getStateFilterConfig,
  getTargetDateFilterConfig,
  getOperatorForPayload,
} from "../../packages/utils/dist/index.js";

const configOptions = { isEnabled: true, allowNegative: true, allowedOperators: new Set(["in", "exact", "range"]) };
const stateConfig = getStateFilterConfig("state_id")({ ...configOptions, states: [] });
const targetDateConfig = getTargetDateFilterConfig("target_date")(configOptions);

function read(expression) {
  return readFilterExpression(workItemFiltersAdapter, expression);
}

function write(snapshot, expression = snapshot.expression) {
  const result = writeFilterExpression(
    workItemFiltersAdapter,
    expression,
    snapshot.committedConditionIds,
    snapshot.error
  );
  assert.equal(result.ok, true, result.ok ? undefined : result.error);
  return result.value;
}

void test("State offers inclusion and exclusion with labels for one or multiple values", () => {
  assert.deepEqual(displayOperatorOptions(stateConfig, ["a", "b"]), [
    { value: "in", label: "is any of" },
    { value: "not_in", label: "is none of" },
  ]);
  assert.deepEqual(displayOperatorOptions(stateConfig, ["a"]), [
    { value: "in", label: "is" },
    { value: "not_in", label: "is not" },
  ]);
  assert.deepEqual(getOperatorForPayload("not_in"), { operator: "in", isNegation: true });
});

void test("changing to exclusion keeps selections through serialization", () => {
  const snapshot = read({ state_id__in: "a,b" });
  const condition = snapshot.expression;
  const excluded = updateFilterConditionOperator(snapshot.expression, condition.id, "in", true, false);
  assert.deepEqual(write(snapshot, excluded), { not: { state_id__in: "a,b" } });
  assert.equal(displayFilterConditions(excluded)[0].operator, "not_in");
  assert.deepEqual(condition.value, ["a", "b"]);
  const included = updateFilterConditionOperator(excluded, condition.id, "in", false, false);
  assert.deepEqual(write(snapshot, included), { state_id__in: "a,b" });
});

void test("editing an excluded condition preserves surrounding conditions", () => {
  const snapshot = read({ and: [{ not: { state_id__in: "a,b" } }, { priority__in: "high" }] });
  const state = snapshot.expression.children[0].child;
  const updated = updateFilterConditionValue(snapshot.expression, state.id, ["c"]);
  assert.deepEqual(write(snapshot, updated.expression), {
    and: [{ not: { state_id__in: "c" } }, { priority__in: "high" }],
  });
});

void test("date exclusions retain date-specific labels and survive serialization", () => {
  const snapshot = read({ not: { target_date__range: "2026-10-01,2026-10-31" } });
  assert.equal(operatorLabel(targetDateConfig, "not_range"), "not between");
  assert.equal(operatorLabel(targetDateConfig, "not_exact"), "is not");
  assert.deepEqual(write(snapshot), { not: { target_date__range: "2026-10-01,2026-10-31" } });
});

void test("nested NOT groups round-trip and display each leaf's effective operator", () => {
  const expression = { not: { and: [{ not: { state_id__in: "a" } }, { priority__in: "high" }] } };
  const snapshot = read(expression);
  assert.deepEqual(write(snapshot), expression);
  assert.deepEqual(
    displayFilterConditions(snapshot.expression).map((condition) => condition.operator),
    ["not_in", "in"]
  );
});

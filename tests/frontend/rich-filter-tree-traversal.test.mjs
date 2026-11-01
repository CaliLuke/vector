import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { FILTER_NODE_TYPE, LOGICAL_OPERATOR } = await import("../../packages/types/dist/index.mjs");
const {
  TreeTraversalMode,
  traverseExpressionTree,
  findParentChain,
  findImmediateParent,
  findConditionsByPropertyAndOperator,
} = await import("../../packages/utils/src/rich-filters/operations/traversal/core.ts");

function expressionFixture() {
  const priority = {
    id: "priority",
    type: FILTER_NODE_TYPE.CONDITION,
    property: "priority",
    operator: "exact",
    value: 0,
  };
  const state = { id: "state", type: FILTER_NODE_TYPE.CONDITION, property: "state", operator: "exact", value: "open" };
  const negation = {
    id: "negated",
    type: FILTER_NODE_TYPE.GROUP,
    logicalOperator: LOGICAL_OPERATOR.NOT,
    child: priority,
  };
  const root = {
    id: "root",
    type: FILTER_NODE_TYPE.GROUP,
    logicalOperator: LOGICAL_OPERATOR.AND,
    children: [state, negation],
  };
  return { priority, state, negation, root };
}

void test("group traversal retains depth and immediate parents through negated conditions", () => {
  const { root } = expressionFixture();
  assert.deepEqual(
    traverseExpressionTree(root, (node, parent, depth) => [node.id, parent?.id, depth], TreeTraversalMode.GROUPS),
    [
      ["root", undefined, 0],
      ["negated", "root", 1],
    ]
  );
});

void test("a visitor can omit group results without pruning their descendants", () => {
  const { root } = expressionFixture();
  assert.deepEqual(
    traverseExpressionTree(root, (node) => (node.type === FILTER_NODE_TYPE.CONDITION ? node.id : null)),
    ["state", "priority"]
  );
  assert.deepEqual(
    traverseExpressionTree(null, () => {
      throw new Error("an absent tree has no nodes");
    }),
    []
  );
});

void test("parent lookup returns closest ancestors first and rejects missing nodes", () => {
  const { root, priority, negation } = expressionFixture();
  assert.deepEqual(findParentChain(root, "priority"), [negation, root]);
  assert.equal(findImmediateParent(root, "priority"), negation);
  assert.deepEqual(findParentChain(root, "state"), [root]);
  assert.equal(findParentChain(root, "missing"), null);
  assert.equal(findImmediateParent(root, "root"), null);
  assert.equal(findImmediateParent(null, "priority"), null);
  assert.equal(findParentChain(priority, "priority"), null);
});

void test("condition lookup respects the display negation and requested property", () => {
  const { root, priority, state } = expressionFixture();
  assert.deepEqual(findConditionsByPropertyAndOperator(root, "priority", "not_exact"), [
    { ...priority, operator: "not_exact" },
  ]);
  assert.deepEqual(findConditionsByPropertyAndOperator(root, "priority", "exact"), []);
  assert.deepEqual(findConditionsByPropertyAndOperator(root, "state", "exact"), [state]);
});

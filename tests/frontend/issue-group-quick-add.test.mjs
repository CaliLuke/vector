import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { canQuickAddToGroup, quickAddGroupPayload } =
  await import("../../apps/web/core/components/issues/issue-layouts/quick-add/grouping.ts");

await test("quick creation respects the feature flag and type grouping at either level", () => {
  assert.equal(canQuickAddToGroup(false, "state"), false);
  assert.equal(canQuickAddToGroup(undefined, "state"), false);
  assert.equal(canQuickAddToGroup(true, "state"), true);
  assert.equal(canQuickAddToGroup(true, null), true);
  assert.equal(canQuickAddToGroup(true, "type_id"), false);
  assert.equal(canQuickAddToGroup(true, "state", "type_id"), false);
  assert.equal(canQuickAddToGroup(true, "type_id", "state"), false);
});

await test("quick creation carries group values, with subgroup precedence and no creator override", () => {
  for (const [key, expected] of [
    ["state", { state_id: "group" }],
    ["priority", { priority: "group" }],
    ["cycle", { cycle_id: "group" }],
    ["module", { module_ids: ["group"] }],
    ["labels", { label_ids: ["group"] }],
    ["assignees", { assignee_ids: ["group"] }],
    ["created_by", {}],
    ["customproperty_one", { customproperty_one: "group" }],
  ])
    assert.deepEqual(quickAddGroupPayload("default", key, "group", undefined, "unused"), {
      state_id: "default",
      ...expected,
    });
  assert.deepEqual(quickAddGroupPayload("default", "state", "outer", "state", "inner"), { state_id: "inner" });
  assert.deepEqual(quickAddGroupPayload("default", "module", "outer", "labels", "inner"), {
    state_id: "default",
    module_ids: ["outer"],
    label_ids: ["inner"],
  });
  assert.deepEqual(quickAddGroupPayload("default", undefined, "outer", "state", "inner"), {});
  for (const key of ["labels", "assignees"]) {
    const payload = quickAddGroupPayload("default", key, "None", null, "unused");
    assert.equal(payload.label_ids, undefined);
    assert.equal(payload.assignee_ids, undefined);
  }
});

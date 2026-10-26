import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { canQuickAddToGroup } =
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

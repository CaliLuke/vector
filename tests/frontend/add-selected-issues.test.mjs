import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const toasts = [];
globalThis.selectedIssueToasts = toasts;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/propel/toast" && context.parentURL?.endsWith("/add-selected-issues.ts")) {
      return {
        shortCircuit: true,
        url: "data:text/javascript,export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'};export const setToast=t=>globalThis.selectedIssueToasts.push(t);",
      };
    }
    return nextResolve(specifier, context);
  },
});
const { addSelectedIssuesToView } =
  await import("../../apps/web/core/components/issues/issue-layouts/add-selected-issues.ts");
test.beforeEach(() => {
  toasts.length = 0;
});

await test("adding selected work items waits for persistence before reporting success", async () => {
  const saved = [];
  let finish;
  const pending = new Promise((resolve) => {
    finish = resolve;
  });
  const result = addSelectedIssuesToView([{ id: "second" }, { id: "first" }], async (ids) => {
    await pending;
    saved.push(...ids);
  });
  assert.deepEqual(toasts, []);
  assert.deepEqual(saved, []);
  finish();
  await result;
  assert.deepEqual(saved, ["second", "first"]);
  assert.equal(toasts.length, 1);
  assert.equal(toasts[0].type, "success");
});

await test("a rejected add reports failure without retrying or announcing success", async () => {
  let attempts = 0;
  await addSelectedIssuesToView([{ id: "rejected" }], async () => {
    attempts++;
    throw new Error("Write rejected");
  });
  assert.equal(attempts, 1);
  assert.equal(toasts.length, 1);
  assert.equal(toasts[0].type, "error");
});

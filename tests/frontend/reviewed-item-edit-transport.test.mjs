import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { APIService } = await import("../../apps/web/core/services/api.service.ts");
const { authService } = await import("../../apps/web/core/services/auth.service.ts");
const { workItemEditsService, itemEditChanges } =
  await import("../../apps/web/core/services/work-item-edits.service.ts");
const { issueFieldErrors } = await import("../../apps/web/core/components/issues/issue-modal/issue-field-errors.ts");
const { propertyFieldErrors } = await import("../../apps/web/core/components/work-item-types/item-property-draft.ts");

await test("a replaced scope during CSRF lookup cannot send an old reviewed edit", async (t) => {
  const csrf = Promise.withResolvers();
  t.mock.method(authService, "requestCSRFToken", () => csrf.promise);
  const post = t.mock.method(APIService.prototype, "post", async () => ({ data: {} }));
  let current = true;
  const pending = workItemEditsService.commit(
    "acme",
    { mode: "edit", item_id: "item", project_id: "project", revision: 1, changes: {}, review_id: "receipt" },
    () => {
      if (!current) throw new Error("Replaced scope");
    }
  );
  current = false;
  csrf.resolve({ csrf_token: "test-token" });
  await assert.rejects(pending, /Replaced scope/);
  assert.equal(post.mock.callCount(), 0);
});

await test("reviewed requests keep explicit clears and values while excluding lifecycle metadata", () => {
  assert.deepEqual(
    itemEditChanges({
      id: "item",
      is_draft: true,
      project_id: null,
      name: "Edited",
      type_id: null,
      custom_properties: { number: 0, checkbox: false },
      module_ids: [],
    }),
    {
      project_id: null,
      name: "Edited",
      type_id: null,
      custom_properties: { number: 0, checkbox: false },
      module_ids: [],
    }
  );
});

await test("reviewed field errors reach ordinary and custom controls", () => {
  const error = {
    response: {
      status: 400,
      data: { message: "Check fields", fields: { name: ["Required"], custom_properties: { notes: "Too long" } } },
    },
  };
  assert.deepEqual(issueFieldErrors(error), { name: "Required" });
  assert.deepEqual(propertyFieldErrors(error), { notes: "Too long" });
});

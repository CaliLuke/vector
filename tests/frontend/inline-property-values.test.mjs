import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { reloadPropertyDraft } =
  await import("../../apps/web/core/components/work-item-types/inline-property-values.ts");
const field = (id, kind = "Text", format = "Single line") => ({ id, title: id, kind, configuration: { format } });
const schema = {
  enabled: true,
  properties: [field("text"), field("amount", "Number"), field("flag", "Boolean"), field("clear"), field("missing")],
};

await test("reload preserves only explicit edits including zero, false, clears and raw numeric typing", () => {
  const result = reloadPropertyDraft(
    { text: "old", amount: "0.", flag: false, clear: null },
    { text: "old", amount: 5, flag: true, clear: "saved" },
    schema,
    { text: "new elsewhere", amount: 9, flag: true, clear: "remote", missing: "new" },
    schema
  );
  assert.deepEqual(result.values, { text: "new elsewhere", amount: "0.", flag: false, clear: null, missing: "new" });
  assert.deepEqual(result.changed, ["amount", "flag", "clear"]);
  assert.deepEqual(result.unavailable, []);
});

await test("equivalent numeric input does not overwrite a newer remote value", () => {
  const result = reloadPropertyDraft({ amount: "5." }, { amount: 5 }, schema, { amount: 7 }, schema);
  assert.deepEqual(result.values, { amount: 7 });
  assert.deepEqual(result.changed, []);
});

await test("reload identifies edited fields that became read-only or detached", () => {
  const result = reloadPropertyDraft(
    { text: "local", amount: "2" },
    { text: "saved", amount: 1 },
    schema,
    {},
    { enabled: true, properties: [field("text", "Text", "Read-only")] }
  );
  assert.deepEqual(
    result.unavailable.map((f) => f.id),
    ["text", "amount"]
  );
  assert.deepEqual(result.values, { text: "local", amount: "2" });
});

await test("disabled rollout retains dirty values for explicit discard", () => {
  const result = reloadPropertyDraft({ text: "local" }, { text: "saved" }, schema, {}, { ...schema, enabled: false });
  assert.deepEqual(
    result.unavailable.map((f) => f.id),
    ["text"]
  );
  assert.deepEqual(result.values, { text: "local" });
});

import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { resolvePropertyContext } = await import("../../apps/web/core/components/work-item-types/context-resolution.ts");
const { newProperty } = await import("../../apps/web/core/components/work-item-types/preview-content.ts");
const context = (fields) => ({
  id: "context",
  name: "Context",
  projects: [],
  types: [],
  required: null,
  active: null,
  defaultValue: null,
  format: null,
  ...fields,
});
await test("a specific context inherits only base settings and preserves explicit falsy overrides", () => {
  const property = {
    ...newProperty("Number"),
    defaultValue: "5",
    required: false,
    contexts: [
      context({ id: "broad", projects: ["p"], defaultValue: "9" }),
      context({ id: "specific", projects: ["p"], types: ["t"], active: false }),
    ],
  };
  const resolved = resolvePropertyContext(property, "p", "t");
  assert.equal(resolved.context.id, "specific");
  assert.equal(resolved.effective.defaultValue, "5");
  assert.equal(resolved.effective.active, false);
  assert.equal(resolvePropertyContext(property, "other", "t").context, undefined);
});
await test("required overrides clear inherited defaults and conflicting scopes are surfaced", () => {
  const property = {
    ...newProperty("Number"),
    defaultValue: "5",
    contexts: [context({ projects: ["p"], required: true })],
  };
  assert.equal(resolvePropertyContext(property, "p", "t").effective.defaultValue, "");
  property.contexts.push(context({ id: "other", types: ["t"] }));
  assert.throws(() => resolvePropertyContext(property, "p", "t"), /conflicting scopes/);
});
await test("inherited option catalogs and explicit empty overrides remain distinct", () => {
  const property = {
    ...newProperty("Cascading"),
    options: [{ id: "root", name: "Root", level: 0, parent: "" }],
    contexts: [context({ projects: ["p"] })],
  };
  assert.equal(resolvePropertyContext(property, "p", "t").effective.options.length, 1);
  property.contexts[0].options = [];
  assert.deepEqual(resolvePropertyContext(property, "p", "t").effective.options, []);
});

await test("scope preview follows the domain resolver's case-insensitive UUID comparison", () => {
  const id = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee";
  const property = {
    ...newProperty("Number"),
    contexts: [context({ projects: [id.toUpperCase()], defaultValue: "7" })],
  };
  assert.equal(resolvePropertyContext(property, id, "").effective.defaultValue, "7");
});

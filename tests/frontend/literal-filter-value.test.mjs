import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { pathToFileURL } from "node:url";
import path from "node:path";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const typesURL = pathToFileURL(path.resolve("packages/types/dist/index.mjs")).href;
const extendedTypesURL = pathToFileURL(path.resolve("packages/types/src/rich-filters/field-types/extended.ts")).href;
const typedRuntime = `export * from ${JSON.stringify(typesURL)}; import { FILTER_FIELD_TYPE as core } from ${JSON.stringify(typesURL)}; import { EXTENDED_FILTER_FIELD_TYPE } from ${JSON.stringify(extendedTypesURL)}; export const FILTER_FIELD_TYPE = {...core,...EXTENDED_FILTER_FIELD_TYPE};`;
registerHooks({
  resolve(specifier, context, next) {
    if (specifier === "@plane/types")
      return { url: `data:text/javascript,${encodeURIComponent(typedRuntime)}`, shortCircuit: true };
    if (
      context.parentURL?.endsWith("/filter-value-input/root.tsx") &&
      (specifier.startsWith("./date/") || specifier.startsWith("./select/"))
    )
      return {
        url: "data:text/javascript,export const SingleDateFilterValueInput=()=>null,DateRangeFilterValueInput=()=>null,SingleSelectFilterValueInput=()=>null,MultiSelectFilterValueInput=()=>null;",
        shortCircuit: true,
      };
    return next(specifier, context);
  },
});
// Initialize DOM before Headless UI/React DOM choose their client behavior.
const bootstrap = await createReactDomTestEnvironment();
const { LiteralFilterValueInput } =
  await import("../../apps/web/core/components/rich-filters/filter-value-input/literal/input.tsx");
const { parseLiteralValue } =
  await import("../../apps/web/core/components/rich-filters/filter-value-input/literal/value.ts");
const { FilterValueInput } = await import("../../apps/web/core/components/rich-filters/filter-value-input/root.tsx");
const { createFilterConfig, createFilterFieldConfig } =
  await import("../../packages/utils/src/rich-filters/factories/configs/shared.ts");
await bootstrap.unmount();

function element(props = {}) {
  return React.createElement(LiteralFilterValueInput, {
    condition: { id: "condition", type: "condition", property: "customproperty_number", operator: "exact", value: 7 },
    config: { type: "number" },
    ...props,
  });
}
function button(label) {
  return [...document.querySelectorAll("button")].find((node) => node.textContent.trim() === label);
}
async function fill(input, value) {
  await React.act(() => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

void test("invalid numeric drafts do not change filters; zero applies and closes", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const values = [];
  await environment.render(element({ onChange: (value) => values.push(value) }));
  await React.act(() => button("7").click());
  await fill(document.querySelector("input"), "1e-");
  await React.act(() => button("Apply").click());
  assert.deepEqual(values, []);
  assert.match(document.body.textContent, /Enter a finite number/);
  assert.equal(document.querySelector("input").value, "1e-");
  await fill(document.querySelector("input"), "0");
  await React.act(() => button("Apply").click());
  assert.deepEqual(values, [0]);
  assert.equal(document.querySelector("input"), null);
});

void test("range rejects reversed bounds and commits both endpoints together", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const values = [];
  await environment.render(
    element({
      condition: { id: "range", type: "condition", property: "effort", operator: "range", value: [1, 4] },
      config: { type: "number_range" },
      onChange: (value) => values.push(value),
    })
  );
  await React.act(() => button("1 – 4").click());
  const [start, end] = document.querySelectorAll("input");
  await fill(start, "9");
  await React.act(() => button("Apply").click());
  assert.match(document.body.textContent, /lower bound/);
  assert.deepEqual(values, []);
  await fill(end, "12.5");
  await React.act(() =>
    end.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }))
  );
  assert.deepEqual(values, [[9, 12.5]]);
});

void test("cancel and second trigger click discard text drafts without applying", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const values = [];
  await environment.render(
    element({
      condition: { id: "text", type: "condition", property: "name", operator: "exact", value: "Saved" },
      config: { type: "text" },
      onChange: (value) => values.push(value),
    })
  );
  await React.act(() => button("Saved").click());
  await fill(document.querySelector("input"), "Unsaved");
  await React.act(() => button("Cancel").click());
  await React.act(() => button("Saved").click());
  assert.equal(document.querySelector("input").value, "Saved");
  await React.act(() => button("Saved").click());
  assert.equal(document.querySelector("input"), null);
  assert.deepEqual(values, []);
});

void test("disabling an open editor removes its editable controls", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const values = [];
  await environment.render(element({ onChange: (value) => values.push(value) }));
  await React.act(() => button("7").click());
  assert.ok(document.querySelector("input"));
  await environment.render(element({ isDisabled: true, onChange: (value) => values.push(value) }));
  assert.equal(document.querySelector("input"), null);
  assert.equal(button("7").disabled, true);
  assert.deepEqual(values, []);
});

void test("numeric parsing accepts decimals and exponents but rejects coercion and overflow", () => {
  assert.deepEqual(parseLiteralValue("number", { start: "-.25", end: "" }), { value: -0.25 });
  assert.deepEqual(parseLiteralValue("number", { start: "1e3", end: "" }), { value: 1000 });
  assert.ok(parseLiteralValue("number", { start: "", end: "" }).error);
  assert.ok(parseLiteralValue("number", { start: "0x10", end: "" }).error);
  assert.ok(parseLiteralValue("number", { start: "Infinity", end: "" }).error);
  assert.ok(parseLiteralValue("number", { start: "1e999", end: "" }).error);
  assert.deepEqual(parseLiteralValue("text", { start: "  words, kept  ", end: "" }), { value: "  words, kept  " });
});

void test("registered numeric config reaches the real generic filter renderer", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const changes = [];
  const condition = {
    id: "effort",
    type: "condition",
    property: "customproperty_effort",
    operator: "exact",
    value: 7,
  };
  const config = createFilterConfig({
    id: "customproperty_effort",
    label: "Effort",
    isEnabled: true,
    supportedOperatorConfigsMap: new Map([["exact", createFilterFieldConfig({ type: "number", allowNegative: true })]]),
  }).supportedOperatorConfigsMap.get("exact");
  await environment.render(
    React.createElement(FilterValueInput, {
      condition,
      filterFieldConfig: config,
      onChange: (value) => changes.push(value),
    })
  );
  await React.act(() => button("7").click());
  await fill(document.querySelector("input"), "42");
  await React.act(() => button("Apply").click());
  assert.deepEqual(changes, [42]);
});

void test("opening another value editor closes the previous one", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(
    React.createElement(
      React.Fragment,
      null,
      element({ onChange: () => {} }),
      element({
        condition: { id: "second", type: "condition", property: "customproperty_other", operator: "exact", value: 8 },
        onChange: () => {},
      })
    )
  );
  await React.act(() => button("7").click());
  assert.equal(document.querySelectorAll("input").length, 1);
  await React.act(() => button("8").click());
  assert.equal(document.querySelectorAll("input").length, 1);
  assert.equal(document.querySelector("input").value, "8");
});

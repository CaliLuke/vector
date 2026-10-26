import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { ItemPropertyValue } = await import("../../apps/web/core/components/work-item-types/item-property-value.tsx");
const base = { id: "value", title: "Value", configuration: { format: "Single line", options: [] } };
async function fixture(t) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  return async (kind, value, config = {}, rest = {}) => {
    const property = { ...base, kind, configuration: { ...base.configuration, ...config } };
    const schema = { stored_values: { value }, computed: {}, members: [], releases: [], ...rest };
    await env.render(
      React.createElement(ItemPropertyValue, {
        property,
        schema,
        renderRichText: (html) => React.createElement("article", { "data-readonly-rich-text": true }, html),
      })
    );
    return document.body.textContent;
  };
}
await test("readonly fields distinguish empty, zero, false and configured content without applying defaults", async (t) => {
  const render = await fixture(t);
  assert.equal(await render("Number", 0), "0");
  assert.equal(await render("Boolean", false), "No");
  assert.equal(await render("Text", undefined, { default_value: "Never apply" }), "—");
  assert.equal(await render("Text", "retained", { format: "Read-only", content: "Guidance" }), "Guidance");
  assert.equal(await render("Text", "line one\nline two", { format: "Paragraph" }), "line one\nline two");
  assert.equal(await render("Rich text", "<p>Saved rich content</p>"), "<p>Saved rich content</p>");
  assert.equal(document.querySelectorAll("input,textarea,select,button").length, 0);
});
await test("references retain full labels and paths, icons, and generic unavailable fallbacks", async (t) => {
  const render = await fixture(t);
  const options = [
    { id: "parent", name: "Platform", parent: "", icon: "bug" },
    { id: "child", name: "API", parent: "parent", icon: "task" },
  ];
  assert.equal(await render("Dropdown", "child", { options }), "Platform → API");
  assert.ok(document.querySelector("svg"));
  assert.equal(await render("Cascading", ["parent", "child"], { options }), "Platform → API");
  assert.equal(
    await render("Member picker", ["member", "gone"], {}, { members: [{ id: "member", name: "Ada" }] }),
    "Ada, Unavailable selection"
  );
  assert.equal(
    await render("Release picker", "release", {}, { releases: [{ id: "release", name: "October release" }] }),
    "October release"
  );
});
await test("date presentations and safe URL links render as saved", async (t) => {
  const render = await fixture(t);
  assert.equal(await render("Date", "2026-10-25", { format: "15/01/2025" }), "25/10/2026");
  assert.equal(await render("Date", "2026-10-25", { format: "01/15/2025" }), "10/25/2026");
  assert.equal(await render("Date", "2026-10-25", { format: "2025/01/15" }), "2026/10/25");
  assert.equal(await render("Date", "2026-10-25", { format: "Jan 15, 2025" }), "Oct 25, 2026");
  await render("URL", "https://example.com");
  assert.equal(document.querySelector("a").rel, "noopener noreferrer");
  await render("URL", "javascript:alert(1)");
  assert.equal(document.querySelector("a"), null);
});
await test("formula empty, false, zero, pending and error remain distinct", async (t) => {
  const render = await fixture(t);
  const result = async (computed) => render("Formula", null, {}, { computed: { value: computed } });
  assert.equal(await result(undefined), "Calculating…");
  assert.equal(await result({ status: "empty" }), "—");
  assert.equal(await result({ status: "value", value: false }), "No");
  assert.equal(await result({ status: "value", value: 0 }), "0");
  assert.equal(await result({ status: "error", error: "Division by zero" }), "Division by zero");
});

import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { PropertyCellValue } =
  await import("../../apps/web/core/components/issues/issue-layouts/spreadsheet/property-cell-value.tsx");
async function fixture(t) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  return async (kind, value, field = {}, projection = {}) => {
    await env.render(
      React.createElement(PropertyCellValue, {
        field: { kind, options: [], catalog_id: "", ...field },
        projection: { kind, status: "value", value, catalog_id: "", ...projection },
      })
    );
    return document.body.textContent;
  };
}
await test("table projections preserve false, zero, date format and formula errors", async (t) => {
  const render = await fixture(t);
  assert.equal(await render("Number", 0), "0");
  assert.equal(await render("Boolean", false), "No");
  assert.equal(await render("Date", "2026-10-25", {}, { format: "15/01/2025" }), "25/10/2026");
  assert.equal(await render("Formula", false, { result_kind: "Boolean" }), "No");
  assert.equal(await render("Formula", null, {}, { status: "error", error: "Division by zero" }), "Calculation error");
  assert.equal(document.querySelector('[title="Division by zero"]')?.textContent, "Calculation error");
});
await test("table references use authorized labels, complete paths and matching cascade catalogue levels", async (t) => {
  const render = await fixture(t);
  const options = [
    { id: "parent", name: "Platform" },
    { id: "child", name: "API", parent: "parent" },
  ];
  assert.equal(await render("Dropdown", "child", { options }), "Platform → API");
  assert.equal(await render("Member picker", ["parent", "private"], { options }), "Platform, Unavailable selection");
  assert.equal(
    await render("Cascading", ["parent", "child"], { options, level_id: "stable" }, { levels: { stable: "child" } }),
    "API"
  );
  assert.equal(
    await render(
      "Cascading",
      ["parent", "child"],
      { options, level_id: "stable", catalog_id: "other" },
      { levels: { stable: "child" } }
    ),
    "—"
  );
  assert.equal(document.querySelector('[title="Not applicable"]')?.textContent, "—");
});
await test("rich text preview decodes text safely and URL links cannot execute script", async (t) => {
  const render = await fixture(t);
  assert.equal(
    await render("Rich text", "<p>A &amp; B <b>bold</b><script>alert(1)</script><img src=x onerror=alert(1)></p>"),
    "A & B bold"
  );
  assert.equal(document.querySelector("script,img,b,p"), null);
  await render("URL", "javascript:alert(1)");
  assert.equal(document.querySelector("a"), null);
  await render("URL", "https://example.com");
  assert.equal(document.querySelector("a").rel, "noopener noreferrer");
});

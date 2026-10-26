import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { PropertyColumn } =
  await import("../../apps/web/core/components/issues/issue-layouts/spreadsheet/property-column.tsx");
const { TablePropertyValuesContext } = await import("../../apps/web/core/hooks/use-work-item-table-values.ts");
const { useTableKeyboardNavigation } = await import("../../apps/web/core/hooks/use-table-keyboard-navigation.tsx");
const issue = { id: "item", name: "Test", project_id: "project" };
const field = { property_id: "value", name: "Value", kind: "Number", catalog_id: "", options: [] };
async function fixture(t) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  let opens = 0;
  function Table({ result, column }) {
    return React.createElement(
      TablePropertyValuesContext,
      { value: { item: result } },
      React.createElement(
        "table",
        { onKeyDown: useTableKeyboardNavigation() },
        React.createElement(
          "tbody",
          null,
          React.createElement(
            "tr",
            null,
            React.createElement(PropertyColumn, { issue, field: column, onEdit: () => opens++, disabled: false })
          )
        )
      )
    );
  }
  return {
    env,
    opens: () => opens,
    render: (result, column = field) => env.render(React.createElement(Table, { result, column })),
  };
}
await test("unattached and foreign-catalog properties are not loading and cannot be edited", async (t) => {
  const f = await fixture(t);
  await f.render({ status: "ready", data: { property_values: {} } });
  assert.equal(f.env.container.textContent, "—");
  assert.equal(f.env.container.querySelector("button"), null);
  await f.render(
    {
      status: "ready",
      data: {
        property_values: {
          value: { status: "value", kind: "Cascading", catalog_id: "other", levels: { country: "x" } },
        },
      },
    },
    { ...field, kind: "Cascading", level_id: "country" }
  );
  assert.equal(f.env.container.textContent, "—");
  assert.equal(f.env.container.querySelector("button"), null);
  await f.render({ status: "error" });
  assert.equal(f.env.container.textContent, "Unavailable");
});
await test("Enter and Space activate the selected table cell editor", async (t) => {
  const f = await fixture(t);
  await f.render({
    status: "ready",
    data: { property_values: { value: { status: "value", kind: "Number", value: 0 } } },
  });
  const cell = f.env.container.querySelector("td");
  await act(async () => {
    cell.focus();
    cell.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
  });
  assert.equal(f.opens(), 1);
  await act(async () => cell.dispatchEvent(new KeyboardEvent("keydown", { key: " ", bubbles: true })));
  assert.equal(f.opens(), 2);
});

function cells(tag, prefix) {
  return [0, 1].map((n) => React.createElement(tag, { key: n, tabIndex: 0 }, `${prefix}${n}`));
}
function Grid() {
  return React.createElement(
    "table",
    { onKeyDown: useTableKeyboardNavigation() },
    React.createElement("thead", null, React.createElement("tr", null, ...cells("th", "h"))),
    React.createElement("tbody", null, React.createElement("tr", null, ...cells("td", "r")))
  );
}
await test("arrow navigation crosses header/body boundaries and retains the column", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(React.createElement(Grid));
  env.container.querySelectorAll("th,td").forEach((cell) => {
    cell.scrollIntoView = () => {};
  });
  await act(async () => {
    env.container.querySelector("th").focus();
    document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true }));
    document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true }));
  });
  assert.equal(document.activeElement.textContent, "r1");
  await act(async () =>
    document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowUp", bubbles: true }))
  );
  assert.equal(document.activeElement.textContent, "h1");
});

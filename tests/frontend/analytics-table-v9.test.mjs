import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    return resolvePackageStub(
      specifier,
      { "@plane/i18n": "export const useTranslation = () => ({ t: key => key });" },
      context,
      nextResolve
    );
  },
});

void test("analytics table features preserve filtering, sorting, visibility, and export metadata", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  const { analyticsTableFeatures } =
    await import("../../apps/web/core/components/analytics/insight-table/table-features.ts");
  const { flexRender, useTable } = webRequire("@tanstack/react-table");
  const data = [
    { name: "Alpha", score: 1 },
    { name: "Beta", score: 2 },
    { name: "Zed", score: 3 },
  ];
  const columns = [
    {
      accessorKey: "name",
      header: "Name",
      meta: {
        export: {
          key: "name",
          value: (row) => row.original.name.toUpperCase(),
        },
      },
    },
    { accessorKey: "score", header: "Score" },
  ];

  function Harness() {
    const table = useTable({
      features: analyticsTableFeatures,
      columns,
      data,
      state: {
        columnFilters: [{ id: "name", value: "a" }],
        sorting: [{ id: "score", desc: true }],
        columnVisibility: { score: false },
      },
    });
    const rows = table.getRowModel().rows;
    const firstRow = rows[0];
    const nameColumn = table.getColumn("name");
    return React.createElement(
      "section",
      null,
      React.createElement("output", { "data-testid": "rows" }, rows.map((row) => row.original.name).join(",")),
      React.createElement(
        "output",
        { "data-testid": "visible-columns" },
        table
          .getVisibleLeafColumns()
          .map((column) => column.id)
          .join(",")
      ),
      React.createElement(
        "output",
        { "data-testid": "export-value" },
        nameColumn.columnDef.meta.export.value(firstRow)
      ),
      React.createElement(
        "output",
        { "data-testid": "rendered-header" },
        flexRender(nameColumn.columnDef.header, { column: nameColumn, table })
      )
    );
  }

  await environment.render(React.createElement(Harness));
  assert.equal(environment.container.querySelector('[data-testid="rows"]').textContent, "Beta,Alpha");
  assert.equal(environment.container.querySelector('[data-testid="visible-columns"]').textContent, "name");
  assert.equal(environment.container.querySelector('[data-testid="export-value"]').textContent, "BETA");
  assert.equal(environment.container.querySelector('[data-testid="rendered-header"]').textContent, "Name");
  assert.deepEqual(errors, []);
});

void test("analytics search filters rendered rows without switching input control", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const errors = [];
  t.mock.method(console, "error", (...args) => {
    if (errors.length < 20) errors.push(args.map(String).join(" "));
  });
  const { DataTable } = await import("../../apps/web/core/components/analytics/insight-table/data-table.tsx");
  await environment.render(
    React.createElement(DataTable, {
      columns: [{ accessorKey: "name", header: "Priority" }],
      data: [{ name: "High" }, { name: "Low" }],
      searchPlaceholder: "Priority",
    })
  );
  const input = environment.container.querySelector('input[placeholder="Search"]');
  assert.ok(input);
  const change = async (value) => {
    await React.act(async () => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
  };
  const cells = () => [...environment.container.querySelectorAll("td")].map((cell) => cell.textContent);
  assert.deepEqual(cells(), ["High", "Low"]);
  await change("High");
  assert.deepEqual(cells(), ["High"]);
  await change("");
  assert.deepEqual(cells(), ["High", "Low"]);
  assert.deepEqual(errors, []);
});

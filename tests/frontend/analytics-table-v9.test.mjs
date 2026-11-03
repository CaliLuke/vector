import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "./helpers/module-hooks.mjs";
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

void test("analytics column renderers preserve displayed values and DOM identity across data and label updates", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { DataTable } = await import("../../apps/web/core/components/analytics/insight-table/data-table.tsx");
  const { AnalyticsColumnHeader, AnalyticsValueCell } =
    await import("../../apps/web/core/components/analytics/insight-table/column-renderers.tsx");
  const column = (key, label, align) => ({
    accessorKey: key,
    header: AnalyticsColumnHeader,
    cell: AnalyticsValueCell,
    meta: { align, export: { key, label, value: (row) => row.original[key] } },
  });
  const render = (label, count) =>
    environment.render(
      React.createElement(DataTable, {
        columns: [
          column("count", label, "right"),
          column("name", undefined, "left"),
          column("missing", undefined, "right"),
        ],
        data: [{ name: "High", count, missing: null }],
        searchPlaceholder: "Count",
      })
    );
  await render("Total", 0);
  const header = environment.container.querySelector("th div");
  const value = environment.container.querySelector("td div");
  assert.equal(header.textContent, "Total");
  assert.equal(header.className, "text-right");
  assert.deepEqual(
    [...environment.container.querySelectorAll("td")].map((cell) => cell.textContent),
    ["0", "High", ""]
  );
  assert.equal(environment.container.querySelectorAll("th div")[1].textContent, "name");
  assert.equal(environment.container.querySelectorAll("th div")[1].className, "text-left");
  await render("Updated total", 12);
  assert.equal(environment.container.querySelector("th div"), header);
  assert.equal(environment.container.querySelector("td div"), value);
  assert.equal(header.textContent, "Updated total");
  assert.equal(value.textContent, "12");
});

const loadingHeader = () => "Custom header";

void test("analytics loading rows use the same grouped column structure as the table", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  const { TableLoader } = await import("../../apps/web/core/components/analytics/insight-table/loader.tsx");
  const columns = [
    {
      id: "group",
      header: "Measures",
      columns: [
        { id: "count", header: loadingHeader },
        { id: "points", header: loadingHeader },
      ],
    },
  ];
  await env.render(React.createElement(TableLoader, { columns, rows: 3 }));
  assert.equal(env.container.querySelectorAll("thead tr").length, 2);
  assert.equal(env.container.querySelector("th").colSpan, 2);
  assert.match(env.container.textContent, /Measures/);
  assert.equal(env.container.querySelectorAll("tbody tr").length, 3);
  assert.equal(env.container.querySelectorAll("td").length, 6);
  await env.render(React.createElement(TableLoader, { columns: columns[0].columns, rows: 1 }));
  assert.equal(env.container.querySelectorAll("thead tr").length, 1);
  assert.equal(env.container.querySelectorAll("td").length, 2);
  assert.deepEqual(errors, []);
});

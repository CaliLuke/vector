import assert from "node:assert/strict";
import test from "node:test";
import { createRequire, registerHooks } from "node:module";
import { createModuleStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";

registerWebModuleHooks();
const catalogFixture = createModuleStub(`
  export const fixture = { fields: [], identities: {enabled:true,revision:1}, supported:true };
  export const useWorkItemTableCatalog = (enabled=true) => enabled ? fixture : {...fixture, fields:[]};
`);
const filterOption = new URL(
  "../../apps/web/core/components/issues/issue-layouts/filters/header/helpers/filter-option.tsx",
  import.meta.url
).href;
registerHooks({
  resolve(specifier, context, next) {
    if (specifier === "@/hooks/use-work-item-table-catalog") return catalogFixture;
    if (specifier === "@/components/issues/issue-layouts/filters")
      return createModuleStub(`export { FilterOption } from ${JSON.stringify(filterOption)};`);
    return next(specifier, context);
  },
});
const { fixture } = await import(catalogFixture.url);
const { CustomPropertyGrouping } =
  await import("../../apps/web/core/components/issues/issue-layouts/filters/header/display-filters/custom-property-grouping.tsx");
const { propertyGroupColumns, propertyGroupingIds } =
  await import("../../apps/web/core/components/issues/issue-layouts/property-group-columns.tsx");
const { getProjectIssueRequestParams } = await import("../../apps/web/helpers/issue-list/request.ts");
const { getComputedDisplayFilters } = await import(
  createRequire(new URL("../../apps/web/package.json", import.meta.url)).resolve("@plane/utils/work-item")
);
const { canQuickAddToGroup } =
  await import("../../apps/web/core/components/issues/issue-layouts/quick-add/grouping.ts");

await test("custom grouping menu selects eligible fields, excludes the other axis and reports deleted selections", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  fixture.fields = [
    { key: "customproperty_flag", name: "Ready", kind: "Boolean", groupable: true },
    { key: "customproperty_owner", name: "Owner", kind: "Member", groupable: true },
    { key: "customproperty_many", name: "Multiple owners", kind: "Member", groupable: false },
    { key: "customproperty_notes", name: "Notes", kind: "Rich text", groupable: false },
  ];
  let selected;
  const props = {
    axis: "group_by",
    displayFilters: { layout: "kanban", group_by: "priority", sub_group_by: "customproperty_owner" },
    onChange: (value) => {
      selected = value;
    },
  };
  await env.render(React.createElement(CustomPropertyGrouping, props));
  assert.deepEqual(
    [...env.container.querySelectorAll("button")].map((button) => button.textContent),
    ["Ready"]
  );
  await act(async () => env.container.querySelector("button").click());
  assert.equal(selected, "customproperty_flag");
  await env.render(
    React.createElement(CustomPropertyGrouping, {
      ...props,
      displayFilters: { ...props.displayFilters, group_by: "customproperty_deleted" },
    })
  );
  assert.match(env.container.querySelector('[role="alert"]').textContent, /unavailable/);
  await env.render(
    React.createElement(CustomPropertyGrouping, {
      ...props,
      displayFilters: { ...props.displayFilters, layout: "list" },
    })
  );
  assert.deepEqual(
    [...env.container.querySelectorAll("button")].map((button) => button.textContent),
    ["Ready", "Owner"]
  );
  await env.render(React.createElement(CustomPropertyGrouping, { ...props, isEpic: true }));
  assert.equal(env.container.textContent, "");
});

await test("custom group and subgroup pagination preserves exact saved expressions and stable field keys", () => {
  const expression = { or: [{ type_id__exact: "task" }, { not: { customproperty_flag__exact: true } }] };
  const display = getComputedDisplayFilters({
    layout: "kanban",
    group_by: "customproperty_flag",
    sub_group_by: "customproperty_location_level_country_catalog_base",
  });
  const input = { workspaceSlug: "acme", projectId: "p", richFilters: expression, displayFilters: display };
  const page = { canGroup: true, perPageCount: 10 };
  const first = getProjectIssueRequestParams(input, page);
  assert.equal(first.group_by, display.group_by);
  assert.equal(first.sub_group_by, display.sub_group_by);
  const next = getProjectIssueRequestParams(input, page, "10:1:0", "false", "NotApplicable");
  assert.equal(next.filters, JSON.stringify(expression));
  assert.equal(next.group_filter_by, "customproperty_flag");
  assert.equal(next.group_filter_value, "false");
  assert.equal(next.sub_group_filter_by, display.sub_group_by);
  assert.equal(next.sub_group_filter_value, "NotApplicable");
  assert.equal(next.group_by, undefined);
  assert.equal(next.sub_group_by, undefined);
  assert.equal(canQuickAddToGroup(true, display.group_by), false);
  assert.equal(canQuickAddToGroup(true, "priority", display.sub_group_by), false);
});

await test("custom column labels preserve false, absent applicability and unavailable retained identities", () => {
  const columns = propertyGroupColumns({ kind: "Boolean", icon: "", options: [] }, ["false", "NotApplicable"]);
  assert.equal(columns.find((column) => column.id === "false").name, "False");
  assert.equal(columns.find((column) => column.id === "None").name, "No value");
  assert.equal(columns.find((column) => column.id === "NotApplicable").name, "Not applicable");
  assert.ok(columns.every((column) => column.isDropDisabled));
  const refs = propertyGroupColumns(
    { kind: "Member", icon: "", options: [{ id: "visible", name: "Visible member" }] },
    ["retained"]
  );
  assert.equal(refs.find((column) => column.id === "retained").name, "Unavailable value");
  assert.deepEqual(propertyGroupingIds({ urgent: { false: ["a"] }, high: { true: ["b"] } }, true), ["false", "true"]);
});

await test("parent grouping round-trips its type and includes children without nested expansion", async () => {
  const { getIssueListProjection } = await import("../../apps/web/helpers/issue-list/request.ts");
  const { parentGroupColumns } =
    await import("../../apps/web/core/components/issues/issue-layouts/parent-group-columns.ts");
  const display = getComputedDisplayFilters({ layout: "list", group_by: "parent_type_epic", sub_issue: true });
  assert.equal(display.group_by, "parent_type_epic");
  assert.equal(display.sub_issue, false);
  assert.equal(getIssueListProjection(display).includeSubIssues, false);
  const input = {
    workspaceSlug: "acme",
    projectId: "p",
    richFilters: { not: { priority__exact: "low" } },
    displayFilters: display,
  };
  const first = getProjectIssueRequestParams(input, { canGroup: true, perPageCount: 10 });
  assert.equal(first.group_by, "parent_type_epic");
  assert.equal(first.sub_issue, true);
  const next = getProjectIssueRequestParams(input, { canGroup: true, perPageCount: 10 }, "10:1:0", "parent-id");
  assert.equal(next.group_filter_by, "parent_type_epic");
  assert.equal(next.group_filter_value, "parent-id");
  assert.equal(next.filters, JSON.stringify(input.richFilters));
  assert.equal(next.sub_issue, true);
  assert.equal(canQuickAddToGroup(true, display.group_by), false);
  const columns = parentGroupColumns([{ id: "parent-id", identifier: "APP-1", name: "Release", project_id: "p" }]);
  assert.equal(columns[0].name, "APP-1 Release");
  assert.equal(columns[1].name, "No matching parent");
  assert.ok(columns.every((column) => column.isDropDisabled));
});

await test("inactive parent grouping preserves Table and Calendar sub-item preferences", async () => {
  const { availableExtraOptions } =
    await import("../../apps/web/core/components/issues/issue-layouts/filters/header/display-filters/extra-options.tsx");
  for (const layout of ["spreadsheet", "calendar"]) {
    const display = getComputedDisplayFilters({ layout, group_by: "parent_type_epic", sub_issue: false });
    const request = getProjectIssueRequestParams(
      { workspaceSlug: "acme", projectId: "p", richFilters: {}, displayFilters: display },
      { canGroup: false, perPageCount: 10 }
    );
    assert.equal(display.sub_issue, false);
    assert.notEqual(request.sub_issue, true);
    assert.ok(availableExtraOptions(["sub_issue", "show_empty_groups"], display).includes("sub_issue"));
    assert.equal(getComputedDisplayFilters({ ...display, sub_issue: true }).sub_issue, true);
  }
  assert.deepEqual(
    availableExtraOptions(["sub_issue", "show_empty_groups"], { layout: "list", group_by: "parent_type_epic" }),
    ["show_empty_groups"]
  );
});

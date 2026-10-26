import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const { getIssueListProjection, getProjectIssueRequestParams, matchesProjectIssueRequestScope } =
  await import("../../apps/web/helpers/issue-list/request.ts");

await test("project request parameters use the supplied preference snapshot and keep cursor and date semantics", () => {
  const params = getProjectIssueRequestParams(
    {
      workspaceSlug: "acme",
      projectId: "project-a",
      richFilters: { and: [{ priority: { in: ["urgent"] } }] },
      displayFilters: {
        layout: "kanban",
        group_by: "state",
        sub_group_by: "priority",
        order_by: "-created_at",
        sub_issue: true,
      },
    },
    { canGroup: true, perPageCount: 25, after: "2026-01-01", before: "2026-01-31" },
    "next-cursor",
    "started",
    "urgent"
  );

  assert.deepEqual(params, {
    filters: JSON.stringify({ and: [{ priority: { in: ["urgent"] } }] }),
    order_by: "-created_at",
    sub_issue: true,
    layout: "kanban",
    cursor: "next-cursor",
    per_page: "25",
    target_date: "2026-01-01;after,2026-01-31;before",
    state: "started",
    priority: "urgent",
  });
});

await test("project request defaults normalize an empty preference response without a MobX filter store", () => {
  const params = getProjectIssueRequestParams(
    {
      workspaceSlug: "acme",
      projectId: "project-b",
      richFilters: {},
      displayFilters: {},
    },
    { canGroup: false, perPageCount: 50 },
    undefined
  );

  assert.equal(params.filters, "{}");
  assert.equal(params.layout, "list");
  assert.equal(params.cursor, "50:0:0");
  assert.equal(params.per_page, "50");
});

await test("empty rich-filter values normalize to the same empty expression as the legacy reader", () => {
  const params = getProjectIssueRequestParams(
    {
      workspaceSlug: "acme",
      projectId: "project-a",
      richFilters: null,
      displayFilters: { layout: "list" },
    },
    { canGroup: false, perPageCount: 25 },
    undefined
  );

  assert.equal(params.filters, "{}");
});

await test("a requested calendar grouping overrides the saved grouping in paginated requests", () => {
  const params = getProjectIssueRequestParams(
    {
      workspaceSlug: "acme",
      projectId: "project-a",
      richFilters: {},
      displayFilters: { layout: "calendar", group_by: "state" },
    },
    { canGroup: true, perPageCount: 100, groupedBy: "target_date" },
    undefined
  );

  assert.equal(params.group_by, "target_date");
  assert.equal(params.cursor, "100:0:0");
});

await test("project request inputs cannot be reused across workspace or project scope", () => {
  const input = {
    workspaceSlug: "acme",
    projectId: "project-a",
    richFilters: {},
    displayFilters: {},
  };

  assert.equal(matchesProjectIssueRequestScope(input, "acme", "project-a"), true);
  assert.equal(matchesProjectIssueRequestScope(input, "acme", "project-b"), false);
  assert.equal(matchesProjectIssueRequestScope(input, "beta", "project-a"), false);
});

await test("explicit list projection derives calendar grouping and keeps the requested sort", () => {
  assert.deepEqual(
    getIssueListProjection({ layout: "calendar", group_by: "state", order_by: "-created_at", sub_issue: true }),
    {
      orderBy: "-created_at",
      groupBy: "target_date",
      subGroupBy: undefined,
      includeSubIssues: true,
    }
  );
});

await test("explicit list projection uses subgrouping only for distinct Kanban columns", () => {
  assert.deepEqual(
    getIssueListProjection({ layout: "kanban", group_by: "state", sub_group_by: "priority", sub_issue: false }),
    {
      orderBy: "sort_order",
      groupBy: "state",
      subGroupBy: "priority",
      includeSubIssues: false,
    }
  );
  assert.equal(
    getIssueListProjection({ layout: "kanban", group_by: "state", sub_group_by: "state" }).subGroupBy,
    undefined
  );
});

await test("explicit list projection leaves other layouts ungrouped", () => {
  assert.deepEqual(getIssueListProjection({ layout: "gantt", group_by: "state", order_by: "priority" }), {
    orderBy: "priority",
    groupBy: undefined,
    subGroupBy: undefined,
    includeSubIssues: false,
  });
});

await test("property projections are explicit, spreadsheet-only, and survive pagination", async () => {
  const { computedFilteredParams, getPaginationParams } = await import("../../apps/web/helpers/issue-list/request.ts");
  for (const layout of ["list", "kanban", "calendar", "gantt_chart", "spreadsheet"]) {
    assert.equal(computedFilteredParams({}, { layout }, []).include_properties, undefined);
    const enabled = computedFilteredParams({}, { layout }, [], { includePropertyProjection: true });
    assert.equal(enabled.include_properties, layout === "spreadsheet" ? true : undefined);
    assert.equal(
      getPaginationParams(enabled, { perPageCount: 20 }, "next").include_properties,
      enabled.include_properties
    );
  }
  const input = { workspaceSlug: "acme", projectId: "a", richFilters: {}, displayFilters: { layout: "spreadsheet" } };
  assert.equal(getProjectIssueRequestParams(input, { perPageCount: 20 }, undefined).include_properties, true);
});

await test("custom column preferences retain both selected and deselected keys without accepting malformed values", async () => {
  const { getComputedDisplayProperties } = await import("../../apps/web/node_modules/@plane/utils/dist/index.js");
  const preferences = getComputedDisplayProperties({
    customproperty_abc: true,
    customproperty_def: false,
    customproperty_bad: "true",
    customproperty_: true,
    unrelated: true,
  });
  assert.equal(preferences.customproperty_abc, true);
  assert.equal(preferences.customproperty_def, false);
  assert.equal(preferences.customproperty_bad, undefined);
  assert.equal(preferences["customproperty_"], undefined);
  assert.equal(preferences.unrelated, undefined);
  assert.equal(preferences.state, true);
});

await test("type bucket pagination retains the original expression and intersects the selected bucket", () => {
  const richFilters = { or: [{ type_id__exact: "task" }, { priority__exact: "urgent" }] };
  const input = {
    workspaceSlug: "acme",
    projectId: "project-a",
    richFilters,
    displayFilters: { layout: "kanban", group_by: "type_id", sub_group_by: "priority" },
  };
  const first = getProjectIssueRequestParams(input, { canGroup: true, perPageCount: 25 });
  assert.equal(first.group_by, "type_id");
  const next = getProjectIssueRequestParams(input, { canGroup: true, perPageCount: 25 }, "25:1:0", "task", "urgent");
  assert.equal(next.filters, JSON.stringify(richFilters));
  assert.equal(next.group_filter_by, "type_id");
  assert.equal(next.group_filter_value, "task");
  assert.equal(next.priority, "urgent");
  assert.equal(next.group_by, undefined);
  assert.equal(next.cursor, "25:1:0");
  input.displayFilters = { layout: "kanban", group_by: "priority", sub_group_by: "type_id" };
  const sub = getProjectIssueRequestParams(input, { canGroup: true, perPageCount: 25 }, undefined, "urgent", "None");
  assert.equal(sub.sub_group_filter_by, "type_id");
  assert.equal(sub.sub_group_filter_value, "None");
  assert.equal(sub.filters, JSON.stringify(richFilters));
  assert.equal(sub.sub_group_by, undefined);
});

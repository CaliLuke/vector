import * as ContractEnums from "../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { getChangedIssuefields, convertWorkItemDataToSearchResponse } =
  await import("../../packages/utils/src/work-item/modal.ts");
const { createIssuePayload, handleIssueQueryParamsByLayout } =
  await import("../../packages/utils/src/work-item/base.ts");

await test("dirty work-item fields retain explicit clears without copying unchanged or inherited values", () => {
  const form = { name: "Changed", priority: "low", parent_id: null, assignee_ids: [], description_html: undefined };
  const dirty = { name: true, parent_id: true, assignee_ids: true, description_html: true, unknown: true };
  assert.deepEqual(getChangedIssuefields(form, dirty), {
    name: "Changed",
    parent_id: null,
    assignee_ids: [],
    description_html: undefined,
  });
  const inheritedFlags = Object.create({ priority: true });
  inheritedFlags.name = true;
  assert.deepEqual(getChangedIssuefields(form, inheritedFlags), { name: "Changed" });
  assert.equal(form.priority, "low");
});

void test("work-item search projections preserve supplied metadata and default unloaded relations", () => {
  const item = { id: "item", name: "Investigate", sequence_id: 12, start_date: null };
  assert.deepEqual(convertWorkItemDataToSearchResponse("acme", item, undefined, undefined), {
    id: "item",
    name: "Investigate",
    project_id: "",
    project__identifier: "",
    project__name: "",
    sequence_id: 12,
    type_id: "",
    state__color: "",
    start_date: null,
    state__group: "backlog",
    state__name: "",
    workspace__slug: "acme",
  });
  const supplied = convertWorkItemDataToSearchResponse(
    "acme",
    { ...item, project_id: "project", type_id: "type" },
    { identifier: "ENG", name: "Engineering" },
    { color: "#ff0000", group: "started", name: "In progress" }
  );
  assert.equal(supplied.project_id, "project");
  assert.equal(supplied.type_id, "type");
  assert.equal(supplied.project__identifier, "ENG");
  assert.equal(supplied.project__name, "Engineering");
  assert.equal(supplied.state__color, "#ff0000");
  assert.equal(supplied.state__group, "started");
  assert.equal(supplied.state__name, "In progress");
});

await test("quick-add payloads preserve selected properties and allocate distinct client identities", () => {
  const input = {
    name: "New work",
    priority: "high",
    parent_id: "parent",
    cycle_id: "cycle",
    module_ids: ["module"],
    custom_properties: { estimate: 0 },
  };
  const first = createIssuePayload("project", input);
  const second = createIssuePayload("project", input);
  assert.equal(first.project_id, "project");
  assert.equal(first.name, input.name);
  assert.equal(first.parent_id, "parent");
  assert.equal(first.cycle_id, "cycle");
  assert.deepEqual(first.module_ids, ["module"]);
  assert.deepEqual(first.custom_properties, { estimate: 0 });
  assert.equal(first.priority, "high");
  assert.deepEqual(first.label_ids, []);
  assert.notEqual(first.id, second.id);
  assert.notEqual(first.tempId, second.tempId);
  assert.equal(Object.hasOwn(input, "id"), false);
  assert.equal(Object.hasOwn(first, "created_at"), false, "server-owned fields must not be fabricated");
  assert.equal(handleIssueQueryParamsByLayout(undefined, "issues"), null);
  assert.ok(handleIssueQueryParamsByLayout(ContractEnums.EIssueLayoutTypes.LIST, "issues").includes("group_by"));
  assert.ok(handleIssueQueryParamsByLayout(ContractEnums.EIssueLayoutTypes.LIST, "issues").includes("filters"));
});

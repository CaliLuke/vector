import { filterEditorFixture } from "./helpers/filter-editor-fixture.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClientProvider, notifyManager } from "@tanstack/react-query";
import {
  displayFilterConditions,
  displayOperatorOptions,
  filterConditions,
  readFilterExpression,
  updateFilterConditionOperator,
  updateFilterConditionValue,
  workItemFiltersAdapter,
  writeFilterExpression,
} from "../../packages/shared-state/dist/index.js";
import { getOperatorForPayload } from "../../packages/utils/dist/index.js";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { createProjectQueryDomFixture } from "./helpers/query-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
notifyManager.setScheduler(queueMicrotask);
const stubs = {
  "@/hooks/use-current-user": `export const useCurrentUserId=()=>"alice"; export const useCurrentUser=()=>({data:{id:"alice"}});`,
  "@/hooks/use-permissions": `export const useProjectPermissions=()=>globalThis.propertyCatalogPermissions;`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const bootstrap = await createReactDomTestEnvironment();
const { propertyFilterConfig } = await import("../../apps/web/core/hooks/work-item-filters/property-filter-config.ts");
const { usePropertyFilterConfigs } =
  await import("../../apps/web/core/hooks/work-item-filters/use-property-filter-configs.ts");
const { FiltersRow } = await import("../../apps/web/core/components/rich-filters/filters-row.tsx");
const { workItemTypesService: service } = await import("../../apps/web/core/services/work-item-types.service.ts");
await bootstrap.unmount();
const base = {
  key: "customproperty_effort",
  property_id: "effort",
  name: "Effort",
  icon: "",
  kind: "Number",
  result_kind: "",
  level_id: "",
  catalog_id: "",
  multiple: false,
  groupable: false,
  operators: ["exact", "gte", "range", "isnull"],
  options: [],
};

await test("custom predicates retain extended operators and typed values across adapter round trips", () => {
  for (const expression of [
    { customproperty_effort__gte: 0 },
    { not: { customproperty_text__contains: " A,b " } },
    { customproperty_flag__exact: false },
    { customproperty_effort__isnull: true },
    { customproperty_date__range: ["2026-01-01", "2026-01-31"] },
  ])
    assert.deepEqual(workItemFiltersAdapter.toExternal(workItemFiltersAdapter.toInternal(expression)), expression);
  assert.throws(() => workItemFiltersAdapter.toInternal({ priority__gte: 0 }));
  assert.deepEqual(getOperatorForPayload("not_contains"), { operator: "contains", isNegation: true });
  assert.deepEqual(getOperatorForPayload("not_gte"), { operator: "gte", isNegation: true });
});

await test("property config selects typed controls and preserves inclusion with one selected option", () => {
  const number = propertyFilterConfig(base);
  assert.equal(number.supportedOperatorConfigsMap.get("gte").type, "number");
  assert.equal(number.supportedOperatorConfigsMap.get("range").type, "number_range");
  assert.equal(number.supportedOperatorConfigsMap.get("isnull").defaultValue, true);
  const operatorChoices = displayOperatorOptions(number, 2);
  assert.equal(operatorChoices.find((option) => option.value === "isnull").label, "presence is");
  assert.notEqual(operatorChoices.find((option) => option.value === "exact").label, "presence is");
  const date = propertyFilterConfig({ ...base, kind: "Date", operators: ["gte", "range"] });
  assert.equal(date.supportedOperatorConfigsMap.get("gte").operatorLabel, "on or after");
  assert.equal(date.supportedOperatorConfigsMap.get("range").type, "date_range");
  const selection = propertyFilterConfig({
    ...base,
    kind: "Cascading",
    operators: ["in"],
    options: [{ id: "root", name: "Root" }],
  });
  assert.equal(selection.supportedOperatorConfigsMap.get("in").singleValueOperator, "in");
  const formula = propertyFilterConfig({ ...base, kind: "Formula", result_kind: "Boolean", operators: ["exact"] });
  assert.equal(formula.supportedOperatorConfigsMap.get("exact").getOptions()[1].value, false);
});

await test("switching between selected values and presence checks never sends the previous operand", () => {
  const snapshot = readFilterExpression(workItemFiltersAdapter, { customproperty_choice__exact: "option-id" });
  const id = snapshot.expression.id;
  let expression = updateFilterConditionOperator(snapshot.expression, id, "isnull", false, true);
  let result = writeFilterExpression(workItemFiltersAdapter, expression, snapshot.committedConditionIds, null);
  assert.deepEqual(result, { ok: true, value: {} });
  expression = updateFilterConditionValue(expression, id, false).expression;
  result = writeFilterExpression(workItemFiltersAdapter, expression, snapshot.committedConditionIds, null);
  assert.deepEqual(result, { ok: true, value: { customproperty_choice__isnull: false } });
  expression = updateFilterConditionOperator(expression, id, "exact", false, true);
  result = writeFilterExpression(workItemFiltersAdapter, expression, snapshot.committedConditionIds, null);
  assert.deepEqual(result, { ok: true, value: {} });
  expression = updateFilterConditionValue(expression, id, "option-id").expression;
  result = writeFilterExpression(workItemFiltersAdapter, expression, snapshot.committedConditionIds, null);
  assert.deepEqual(result, { ok: true, value: { customproperty_choice__exact: "option-id" } });
});

function PropertyHarness({ snapshot, allowed = true }) {
  const result = usePropertyFilterConfigs("acme", "p", allowed);
  const editor = filterEditorFixture({
    expression: snapshot.expression,
    conditions: filterConditions(snapshot.expression),
    displayConditions: displayFilterConditions(snapshot.expression),
    configs: result.configs,
    conversionError: snapshot.error,
    hasActiveFilters: true,
  });
  return React.createElement(
    React.Fragment,
    null,
    result.isError && React.createElement("button", { onClick: result.retry }, "Retry properties"),
    React.createElement(FiltersRow, { editor })
  );
}

await test("disabling a property catalog hides choices without discarding saved predicates", async (t) => {
  const f = await createProjectQueryDomFixture(t, 15);
  globalThis.propertyCatalogPermissions = {
    workspaceRole: 15,
    projectRole: 15,
    membershipQuery: { isSuccess: true, data: { updated_at: "v1" } },
    rolesQuery: { isError: false },
  };
  const snapshot = readFilterExpression(workItemFiltersAdapter, { customproperty_effort__gte: 0 });
  t.mock.method(service, "propertyIdentities", async () => ({ enabled: true, revision: 1, fields: [base] }));
  const draw = (allowed) =>
    f.env.render(
      React.createElement(
        QueryClientProvider,
        { client: f.client },
        React.createElement(PropertyHarness, { snapshot, allowed })
      )
    );
  await draw(true);
  assert.match(f.env.container.textContent, /Effort/);
  assert.match(f.env.container.textContent, /at least/);
  await draw(false);
  assert.doesNotMatch(f.env.container.textContent, /Effort/);
  assert.deepEqual(workItemFiltersAdapter.toExternal(snapshot.expression), { customproperty_effort__gte: 0 });
});

import { registerWebTestModules, reactModuleUrl as reactUrl } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";

import test from "node:test";

import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { createModuleStub } from "./helpers/module-stubs.mjs";

registerWebTestModules(
  {
    "@/hooks/use-work-item-grouping-catalog": "export const useTypeGroupingOptions=()=>options=>options;",
    "lodash-es": "export const isEmpty = (value) => value == null || Object.keys(value).length === 0;",
    "@/components/issues/issue-layouts/filters": `
        import React from "${reactUrl}";
        export const FilterOption = () => null;
        export const FilterDisplayProperties = ({handleUpdate}) => React.createElement('button',{onClick:()=>handleUpdate({priority:false})},'Hide priority');
        export const FilterExtraOptions = ({handleUpdate}) => React.createElement('button',{onClick:()=>handleUpdate('show_empty_groups',false)},'Hide empty groups');
        export const FilterOrderBy = ({handleUpdate}) => React.createElement('button',{onClick:()=>handleUpdate('-created_at')},'Order newest');
        export const FilterSubGroupBy = ({handleUpdate}) => React.createElement('button',{onClick:()=>handleUpdate('priority')},'Subgroup priority');
        export const FilterGroupBy = ({ ignoreGroupedFilters,handleUpdate }) =>
          React.createElement("button", { "data-ignored-groups": JSON.stringify(ignoreGroupedFilters),onClick:()=>handleUpdate('state') },'Group by state');
      `,
  },
  (specifier, context, nextResolve) => {
    if (specifier === "./custom-property-columns" && context.parentURL?.includes("display-filters-selection.tsx")) {
      return createModuleStub(`export const CustomPropertyColumns = () => null;`);
    }
    return nextResolve(specifier, context);
  }
);

await test("display filter selection does not mutate grouped options across disable and re-enable", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { DisplayFiltersSelection } =
    await import("../../apps/web/core/components/issues/issue-layouts/filters/header/display-filters/display-filters-selection.tsx");
  const ignoreGroupedFilters = ["priority"];
  const baseProps = {
    displayFilters: { group_by: "priority", layout: "list" },
    displayProperties: {},
    handleDisplayFiltersUpdate() {},
    handleDisplayPropertiesUpdate() {},
    layoutDisplayFiltersOptions: {
      display_filters: { group_by: [{ key: "priority" }, { key: "cycle" }, { key: "module" }] },
      extra_options: {},
    },
    ignoreGroupedFilters,
  };
  const render = (disabled) =>
    environment.render(
      React.createElement(DisplayFiltersSelection, {
        ...baseProps,
        cycleViewDisabled: disabled,
        moduleViewDisabled: disabled,
      })
    );

  await render(true);
  assert.deepEqual(ignoreGroupedFilters, ["priority"], "rendering must not mutate caller-owned options");
  assert.deepEqual(JSON.parse(environment.container.querySelector("[data-ignored-groups]").dataset.ignoredGroups), [
    "priority",
    "cycle",
    "module",
  ]);

  await render(false);
  assert.deepEqual(ignoreGroupedFilters, ["priority"], "re-enabling must not retain injected exclusions");
  assert.deepEqual(JSON.parse(environment.container.querySelector("[data-ignored-groups]").dataset.ignoredGroups), [
    "priority",
  ]);
});

await test("display controls emit only their changed field while retaining kanban subgroup availability", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { DisplayFiltersSelection } =
    await import("../../apps/web/core/components/issues/issue-layouts/filters/header/display-filters/display-filters-selection.tsx");
  const updates = [],
    properties = [];
  await environment.render(
    React.createElement(DisplayFiltersSelection, {
      displayFilters: { layout: "kanban", group_by: "state" },
      displayProperties: { priority: true },
      handleDisplayFiltersUpdate: (update) => updates.push(update),
      handleDisplayPropertiesUpdate: (update) => properties.push(update),
      layoutDisplayFiltersOptions: {
        display_properties: ["priority"],
        display_filters: { group_by: ["state"], sub_group_by: ["priority"], order_by: ["-created_at"] },
        extra_options: { access: true, values: ["show_empty_groups"] },
      },
    })
  );
  const click = async (label) =>
    React.act(async () =>
      [...environment.container.querySelectorAll("button")].find((item) => item.textContent === label).click()
    );
  await click("Group by state");
  await click("Subgroup priority");
  await click("Order newest");
  await click("Hide empty groups");
  await click("Hide priority");
  assert.deepEqual(updates, [
    { group_by: "state" },
    { sub_group_by: "priority" },
    { order_by: "-created_at" },
    { show_empty_groups: false },
  ]);
  assert.deepEqual(properties, [{ priority: false }]);
});

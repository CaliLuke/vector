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
        export const FilterDisplayProperties = () => null;
        export const FilterExtraOptions = () => null;
        export const FilterOrderBy = () => null;
        export const FilterSubGroupBy = () => null;
        export const FilterGroupBy = ({ ignoreGroupedFilters }) =>
          React.createElement("output", { "data-ignored-groups": JSON.stringify(ignoreGroupedFilters) });
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
  assert.deepEqual(JSON.parse(environment.container.querySelector("output").dataset.ignoredGroups), [
    "priority",
    "cycle",
    "module",
  ]);

  await render(false);
  assert.deepEqual(ignoreGroupedFilters, ["priority"], "re-enabling must not retain injected exclusions");
  assert.deepEqual(JSON.parse(environment.container.querySelector("output").dataset.ignoredGroups), ["priority"]);
});

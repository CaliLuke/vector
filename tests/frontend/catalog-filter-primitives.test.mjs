import { registerWebTestModules, reactModuleUrl as reactUrl } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";

import test from "node:test";

import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";

registerWebTestModules({
  "@plane/propel/icons": `
          export const CloseIcon = () => null;
          export const SearchIcon = () => null;
          export const ChevronUpIcon = () => null;
          export const ChevronDownIcon = () => null;
        `,
  "@plane/ui": "export const Loader = Object.assign(() => null, { Item: () => null });",
  "@/hooks/use-platform-os": "export const usePlatformOS = () => ({ isMobile: false });",
  "@/components/issues/issue-layouts/filters/header/helpers/filter-option": `
        import React from "${reactUrl}";
        export const FilterOption = ({ title, isChecked, onClick }) =>
          React.createElement("button", { type: "button", onClick, "data-checked": isChecked }, title);
      `,
});

await test("catalog filter panel shares search, focus, status preview, and selection behavior", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const [{ FilterSelectionPanel }, { CatalogStatusFilter }] = await Promise.all([
    import("../../apps/web/core/components/core/filters/filter-selection-panel.tsx"),
    import("../../apps/web/core/components/core/filters/catalog-status-filter.tsx"),
  ]);
  const updates = [];
  const options = [
    { value: "current", title: "Current" },
    { value: "completed", title: "Completed" },
  ];
  await environment.render(
    React.createElement(FilterSelectionPanel, null, (searchQuery) =>
      React.createElement(CatalogStatusFilter, {
        title: "Status",
        appliedFilters: ["current"],
        handleUpdate: (value) => updates.push(value),
        searchQuery,
        options,
      })
    )
  );

  const search = environment.container.querySelector("input");
  assert.equal(document.activeElement, search, "desktop search should autofocus");
  assert.match(environment.container.textContent, /Status \(1\)/);
  assert.deepEqual(
    [...environment.container.querySelectorAll("button[data-checked]")].map((button) => button.textContent),
    ["Current", "Completed"]
  );
  assert.equal(environment.container.querySelectorAll('button[data-checked="true"]').length, 1);

  await React.act(async () => {
    Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, "value").set.call(search, "com");
    search.dispatchEvent(new Event("input", { bubbles: true }));
  });
  assert.deepEqual(
    [...environment.container.querySelectorAll("button[data-checked]")].map((button) => button.textContent),
    ["Completed"]
  );
  await React.act(() => {
    [...environment.container.querySelectorAll("button")].find((button) => button.textContent === "Completed").click();
  });
  assert.deepEqual(updates, ["completed"]);

  await React.act(() => search.parentElement.querySelector("button").click());
  assert.deepEqual(
    [...environment.container.querySelectorAll("button[data-checked]")].map((button) => button.textContent),
    ["Current", "Completed"]
  );

  await React.act(() => environment.container.querySelectorAll("button")[0].click());
  assert.equal(environment.container.textContent.includes("Completed"), false);
  assert.equal(environment.container.textContent.includes("Current"), false);
  assert.equal(environment.container.querySelectorAll("button[data-checked]").length, 0);
});

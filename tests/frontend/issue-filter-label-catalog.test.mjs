import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, webRequire } from "./helpers/react.mjs";
import { createModuleStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const reactUrl = pathToFileURL(webRequire.resolve("react")).href;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (
      specifier === "@/components/issues/issue-layouts/filters/header/helpers/filter-option" ||
      specifier === "@/components/issues/issue-layouts/filters/header/helpers/filter-option-section"
    ) {
      return createModuleStub(`
        import React from "${reactUrl}";
        export const FilterOption = ({ title, isChecked, onClick }) =>
          React.createElement("button", { type: "button", onClick, "data-checked": isChecked }, title);
        export const FilterOptionSection = ({ title, optionCount, renderOptions }) =>
          React.createElement("section", null,
            React.createElement("h2", null, title),
            ...renderOptions(optionCount)
          );
      `);
    }
    return nextResolve(specifier, context);
  },
});

await test("issue label filter tracks a late catalogue, renames, and selected IDs", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { FilterLabels } =
    await import("../../apps/web/core/components/issues/issue-layouts/filters/header/filters/labels.tsx");

  function Harness({ labels }) {
    const [selected, setSelected] = React.useState([]);
    const handleUpdate = (id) =>
      setSelected((current) => (current.includes(id) ? current.filter((value) => value !== id) : [...current, id]));
    return React.createElement(FilterLabels, {
      appliedFilters: selected,
      handleUpdate,
      labels,
      searchQuery: "",
    });
  }

  const labels = [
    { id: "label-a", name: "Alpha", color: "#111111" },
    { id: "label-b", name: "Beta", color: "#222222" },
  ];
  await environment.render(React.createElement(Harness, { labels: undefined }));
  assert.equal(environment.container.querySelectorAll("button").length, 0);

  await environment.render(React.createElement(Harness, { labels }));
  assert.deepEqual(
    [...environment.container.querySelectorAll("button")].map((button) => button.textContent),
    ["Alpha", "Beta"]
  );

  await React.act(() => {
    [...environment.container.querySelectorAll("button")].find((button) => button.textContent === "Beta").click();
  });
  let betaOption = [...environment.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Beta")
  );
  assert.equal(betaOption.dataset.checked, "true");

  const renamedLabels = [labels[0], { ...labels[1], name: "Renamed Beta" }];
  await environment.render(React.createElement(Harness, { labels: renamedLabels }));
  betaOption = [...environment.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Renamed Beta")
  );
  assert.ok(betaOption, "the same search should display the updated catalog name");
  assert.equal(betaOption.dataset.checked, "true", "selection should remain attached to the canonical label ID");
});

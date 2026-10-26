import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    return resolvePackageStub(
      specifier,
      {
        "@plane/propel/icons": `export const ChevronUpIcon = () => null; export const ChevronDownIcon = () => null;`,
        "@plane/ui": `export const Loader = Object.assign(() => null, { Item: () => null });`,
      },
      context,
      nextResolve
    );
  },
});

await test("filter option sections preserve collapsed defaults and expand only on request", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { FilterOptionSection } =
    await import("../../apps/web/core/components/issues/issue-layouts/filters/header/helpers/filter-option-section.tsx");
  const draw = () =>
    environment.render(
      React.createElement(FilterOptionSection, {
        title: "Labels (2)",
        optionCount: 7,
        renderOptions: (limit) =>
          Array.from({ length: Math.min(limit, 7) }, (_, index) =>
            React.createElement("div", { key: index }, `Option ${index + 1}`)
          ),
      })
    );
  await draw();

  assert.ok(environment.container.textContent.includes("Labels (2)"));
  assert.deepEqual(
    [...environment.container.querySelectorAll("div")]
      .filter((node) => /^Option \d+$/.test(node.textContent))
      .map((node) => node.textContent),
    ["Option 1", "Option 2", "Option 3", "Option 4", "Option 5"]
  );
  assert.ok(environment.container.textContent.includes("View all"));

  await React.act(() => {
    [...environment.container.querySelectorAll("button")].find((button) => button.textContent === "View all").click();
  });
  assert.equal(environment.container.textContent.includes("Option 7"), true);
  assert.ok(environment.container.textContent.includes("View less"));

  await React.act(() => {
    [...environment.container.querySelectorAll("button")].find((button) => button.textContent === "View less").click();
  });
  assert.equal(environment.container.textContent.includes("Option 7"), false);
  assert.ok(environment.container.textContent.includes("View all"));

  await React.act(() => environment.container.querySelector("button").click());
  assert.equal(environment.container.textContent.includes("Option 1"), false);
  assert.equal(environment.container.textContent.includes("Labels (2)"), true);
});

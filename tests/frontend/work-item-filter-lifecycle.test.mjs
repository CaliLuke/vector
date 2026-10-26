import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

await test("filter visibility is isolated by route scope and resets when the scoped provider changes", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { WorkItemFilterUiProvider } = await import("../../apps/web/core/providers/work-item-filter-ui.tsx");
  const { FiltersToggle } = await import("../../apps/web/core/components/rich-filters/filters-toggle.tsx");

  const view = (scopeKey, secondScope = false) =>
    React.createElement(
      React.Fragment,
      null,
      React.createElement(
        WorkItemFilterUiProvider,
        { scopeKey },
        React.createElement("div", { "data-scope": "first" }, React.createElement(FiltersToggle))
      ),
      secondScope &&
        React.createElement(
          WorkItemFilterUiProvider,
          { scopeKey: "another-view" },
          React.createElement("div", { "data-scope": "second" }, React.createElement(FiltersToggle))
        )
    );

  await env.render(view("account-a:workspace-a:project-a:view-a", true));
  const toggles = () => [...env.container.querySelectorAll("button[aria-pressed]")];
  assert.deepEqual(
    toggles().map((button) => button.getAttribute("aria-pressed")),
    ["false", "false"]
  );
  await act(async () => toggles()[0].click());
  assert.deepEqual(
    toggles().map((button) => button.getAttribute("aria-pressed")),
    ["true", "false"]
  );

  await env.render(view("account-a:workspace-b:project-b:view-b", false));
  assert.deepEqual(
    toggles().map((button) => button.getAttribute("aria-pressed")),
    ["false"]
  );
});

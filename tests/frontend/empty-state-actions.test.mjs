import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { EmptyStateCompact } = await import("../../packages/propel/src/empty-state/compact-empty-state.tsx");
const { EmptyStateDetailed } = await import("../../packages/propel/src/empty-state/detailed-empty-state.tsx");

for (const Component of [EmptyStateCompact, EmptyStateDetailed]) {
  void test(`${Component.name} preserves action callbacks, disabled state and custom actions`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const clicked = [];
    const actions = [
      { id: "create", label: "Create", onClick: () => clicked.push("create") },
      { id: "import", label: "Import", disabled: true, onClick: () => clicked.push("import") },
    ];
    await environment.render(React.createElement(Component, { title: "Empty", actions }));
    const buttons = [...environment.container.querySelectorAll("button")];
    assert.deepEqual(
      buttons.map((button) => button.textContent),
      ["Create", "Import"]
    );
    await React.act(async () => buttons.forEach((button) => button.click()));
    assert.deepEqual(clicked, ["create"]);
    await environment.render(
      React.createElement(Component, {
        actions,
        customButton: React.createElement("button", { onClick: () => clicked.push("custom") }, "Custom"),
      })
    );
    assert.equal(environment.container.querySelectorAll("button").length, 1);
    await React.act(async () => environment.container.querySelector("button").click());
    assert.deepEqual(clicked, ["create", "custom"]);
    await environment.render(React.createElement(Component, { actions: [] }));
    assert.equal(environment.container.querySelector("button"), null);
    await environment.render(React.createElement(Component));
    assert.equal(environment.container.querySelector("button"), null);
  });
}

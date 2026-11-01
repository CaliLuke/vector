import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

void test("quick actions share visibility, disabled state, and actions between button and context menus", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { QuickActionsMenu } = await import("../../apps/web/core/components/common/quick-actions-menu.tsx");
  const calls = [];
  const items = [
    { key: "edit", title: "Edit item", description: "Update details", action: () => calls.push("edit") },
    { key: "delete", title: "Delete item", disabled: true, action: () => calls.push("delete") },
    { key: "hidden", title: "Hidden item", shouldRender: false, action: () => calls.push("hidden") },
  ];
  const parentRef = React.createRef();
  await environment.render(
    React.createElement(
      "div",
      { ref: parentRef },
      React.createElement(QuickActionsMenu, { parentRef, items, maxHeight: "lg" })
    )
  );
  const trigger = environment.container.querySelector("button[aria-label='More actions']");
  assert.ok(trigger);
  await React.act(async () => trigger.click());
  assert.ok(document.body.textContent.includes("Update details"));
  assert.equal(document.body.textContent.includes("Hidden item"), false);
  const buttons = Array.from(document.querySelectorAll("[role='menuitem']"));
  const disabled = buttons.find((button) => button.textContent.includes("Delete item"));
  assert.ok(disabled);
  await React.act(async () => disabled.click());
  assert.deepEqual(calls, []);
  const edit = buttons.find((button) => button.textContent.includes("Edit item"));
  assert.ok(edit);
  await React.act(async () => edit.click());
  assert.deepEqual(calls, ["edit"]);
  await React.act(async () =>
    parentRef.current.dispatchEvent(
      new MouseEvent("contextmenu", { bubbles: true, cancelable: true, clientX: 20, clientY: 20 })
    )
  );
  assert.equal(document.body.textContent.includes("Hidden item"), false);
  const contextEdit = Array.from(document.querySelectorAll("button")).find((button) =>
    button.textContent.includes("Edit item")
  );
  assert.ok(contextEdit);
  await React.act(async () => contextEdit.click());
  assert.deepEqual(calls, ["edit", "edit"]);
});

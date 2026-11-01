import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const key = (value) =>
  act(async () => window.dispatchEvent(new KeyboardEvent("keydown", { key: value, cancelable: true })));
const onOpen = () => {};
const onClose = () => {};

async function menuFixture(t, items) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { ContextMenu } = await import("../../packages/ui/src/dropdowns/context-menu/root.tsx");
  const target = document.createElement("div");
  document.body.append(target);
  t.after(() => target.remove());
  const parentRef = { current: target };
  const render = (nextItems) => env.render(React.createElement(ContextMenu, { parentRef, items: nextItems }));
  await render(items);
  await act(async () => target.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, cancelable: true })));
  return { env, render };
}

await test("nested keyboard selection runs only the nested action and retains closeOnClick", async (t) => {
  const calls = [];
  const { env } = await menuFixture(t, [
    {
      key: "parent",
      title: "Parent",
      action: () => calls.push("parent"),
      nestedMenuItems: [{ key: "child", title: "Child", action: () => calls.push("child"), closeOnClick: false }],
    },
  ]);
  await act(async () => env.container.querySelector("button").click());
  const nested = document.querySelector('[data-context-submenu="true"] button');
  assert.ok(nested);
  await act(async () =>
    nested.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }))
  );
  assert.deepEqual(calls, ["child"]);
  assert.ok(document.querySelector('[data-context-submenu="true"]'));
  await act(async () => window.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape" })));
  assert.equal(document.querySelector('[data-context-submenu="true"]'), null);
});

await test("empty menus and removal of the active item cannot activate a missing action", async (t) => {
  let calls = 0;
  const item = (itemKey) => ({ key: itemKey, title: itemKey, action: () => calls++ });
  const { render } = await menuFixture(t, [item("first"), item("second")]);
  await key("ArrowDown");
  await render([item("first")]);
  await key("Enter");
  assert.equal(calls, 0);
  await render([]);
  await key("ArrowDown");
  await key("ArrowUp");
  await key("Enter");
  assert.equal(calls, 0);
  await render([item("replacement")]);
  await key("ArrowDown");
  await key("Enter");
  assert.equal(calls, 1);
});

await test("dropdown keyboard callbacks use the latest selected item", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { useDropdownKeyDown } = await import("../../packages/ui/src/hooks/use-dropdown-key-down.tsx");
  const calls = [];
  function Probe({ selection }) {
    const onKeyDown = useDropdownKeyDown(onOpen, onClose, true, () => calls.push(selection));
    return React.createElement("button", { onKeyDown }, "Choose");
  }
  await env.render(React.createElement(Probe, { selection: "first" }));
  await env.render(React.createElement(Probe, { selection: "second" }));
  await act(async () =>
    env.container.querySelector("button").dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }))
  );
  assert.deepEqual(calls, ["second"]);
});

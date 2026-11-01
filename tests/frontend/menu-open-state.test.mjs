import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment, waitForReactCondition } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const bootstrap = await createReactDomTestEnvironment();
const { Menu } = await import("../../packages/propel/src/menu/menu.tsx");
await bootstrap.unmount();

async function fixture(t, extra = {}) {
  const env = await createReactDomTestEnvironment();
  const calls = { selections: 0, closes: 0, openChanges: [], triggerClicks: 0 };
  const closeWindow = t.mock.method(window, "close", () =>
    assert.fail("a menu selection must never close its browser window")
  );
  t.after(async () => {
    closeWindow.mock.restore();
    await env.unmount();
  });
  const render = (props) =>
    env.render(
      React.createElement(
        "div",
        null,
        React.createElement("button", { id: "outside" }, "Outside"),
        React.createElement(
          Menu,
          {
            label: "Actions",
            ariaLabel: "Actions",
            onMenuClose: () => calls.closes++,
            handleOpenChange: (isOpen) => calls.openChanges.push(isOpen),
            menuButtonOnClick: () => calls.triggerClicks++,
            ...extra,
            ...props,
          },
          React.createElement(Menu.MenuItem, { onClick: () => calls.selections++ }, "Choose")
        )
      )
    );
  await render({});
  return { env, calls, render, trigger: () => env.container.querySelector('button[aria-label="Actions"]') };
}
async function open(f) {
  await act(async () => f.trigger().click());
  await waitForReactCondition(() => document.querySelector('[role="menuitem"]'), "menu opens");
}
await test("menu selection closes the popup once, reports native open state and never closes the window", async (t) => {
  const f = await fixture(t);
  await open(f);
  assert.equal(f.trigger().getAttribute("aria-expanded"), "true");
  await act(async () => document.querySelector('[role="menuitem"]').click());
  await waitForReactCondition(() => f.trigger().getAttribute("aria-expanded") === "false", "selection closes");
  assert.equal(f.calls.selections, 1);
  assert.equal(f.calls.closes, 1);
  assert.deepEqual(f.calls.openChanges, [true, false]);
  await open(f);
  assert.deepEqual(f.calls.openChanges, [true, false, true]);
});
await test("keyboard and outside dismissal restore menu state across repeated openings", async (t) => {
  const f = await fixture(t, { customButton: React.createElement("span", null, "Custom") });
  await act(async () => {
    f.trigger().focus();
    f.trigger().dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }));
  });
  await waitForReactCondition(() => document.querySelector('[role="menuitem"]'), "keyboard opens");
  await act(async () =>
    document.activeElement.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
  await waitForReactCondition(() => f.trigger().getAttribute("aria-expanded") === "false", "Escape closes");
  assert.equal(f.calls.closes, 1);
  assert.equal(document.activeElement, f.trigger());
  await open(f);
  const outside = f.env.container.querySelector("#outside");
  await act(async () => {
    outside.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, button: 0 }));
    outside.dispatchEvent(new MouseEvent("mouseup", { bubbles: true, button: 0 }));
    outside.click();
  });
  await waitForReactCondition(() => f.trigger().getAttribute("aria-expanded") === "false", "outside click closes");
  assert.equal(f.calls.closes, 2);
  assert.deepEqual(f.calls.openChanges, [true, false, true, false]);
});
await test("disabled icon menu cannot open or emit trigger callbacks", async (t) => {
  const f = await fixture(t, { verticalEllipsis: true, disabled: true });
  assert.equal(f.trigger().disabled, true);
  await act(async () => f.trigger().click());
  assert.deepEqual(f.calls.openChanges, []);
  assert.equal(f.calls.triggerClicks, 0);
  await f.render({ disabled: false });
  await open(f);
  assert.equal(f.calls.triggerClicks, 1);
});

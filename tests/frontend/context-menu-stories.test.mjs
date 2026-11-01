import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

async function openExample(t, storyName) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  t.mock.property(globalThis, "AbortController", window.AbortController);
  globalThis.DOMRect = window.DOMRect;
  t.after(() => {
    delete globalThis.DOMRect;
  });
  const stories = await import("../../packages/propel/src/context-menu/context-menu.stories.tsx");
  await env.render(React.createElement(stories[storyName].render));
  const trigger = env.container.firstElementChild;
  assert.ok(trigger);
  await act(async () => {
    trigger.dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, button: 2, clientX: 20, clientY: 20 }));
  });
  const menu = document.querySelector('[role="menu"]');
  assert.ok(menu, "right clicking the story opens its actual context menu");
  return menu;
}

await test("shared context-menu examples retain disabled actions and keep their menu open on disabled activation", async (t) => {
  const menu = await openExample(t, "DisabledItems");
  const items = [...menu.querySelectorAll('[role="menuitem"]')];
  const edit = items.find((item) => item.textContent.includes("Edit"));
  assert.equal(edit?.getAttribute("aria-disabled"), "true");
  await act(async () => edit.click());
  assert.ok(document.querySelector('[role="menu"]'));
  assert.equal(items.find((item) => item.textContent.includes("Copy"))?.getAttribute("aria-disabled"), null);
});

await test("shared context-menu actions retain shortcut hints", async (t) => {
  const menu = await openExample(t, "WithKeyboardShortcuts");
  for (const shortcut of ["⌘C", "⌘E", "⌘D", "⌘⌫"]) assert.ok(menu.textContent.includes(shortcut));
});

await test("a custom file-card trigger retains its file actions", async (t) => {
  const menu = await openExample(t, "OnFileCard");
  assert.match(menu.textContent, /Download/);
  assert.match(menu.textContent, /Copy Link/);
  assert.match(menu.textContent, /Archive/);
});

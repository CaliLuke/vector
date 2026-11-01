import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { ResizableSidebar } = await import("../../apps/web/core/components/sidebar/resizable-sidebar.tsx");

function Harness(props) {
  const [width, setWidth] = React.useState(280);
  return React.createElement(ResizableSidebar, { width, setWidth, ...props });
}

void test("sidebar pointer and keyboard resizing preserve bounds and restore document styles", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const toggles = [];
  document.body.style.cursor = "crosshair";
  document.body.style.userSelect = "text";
  await environment.render(
    React.createElement(Harness, { togglePeek() {}, toggleCollapsed: () => toggles.push(true) })
  );
  assert.equal(document.body.style.cursor, "crosshair", "an idle sidebar does not own document styles");
  const sidebar = environment.container.querySelector("aside[aria-label='Main sidebar']");
  const handle = sidebar.querySelector("button[aria-label='Resize sidebar']");
  assert.equal(handle.tabIndex, 0);
  await React.act(async () => handle.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowLeft", bubbles: true })));
  assert.equal(sidebar.style.width, "270px");
  await React.act(async () => handle.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true })));
  assert.equal(sidebar.style.width, "280px");
  await React.act(async () => handle.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true })));
  assert.equal(toggles.length, 1);
  await React.act(async () => handle.dispatchEvent(new MouseEvent("mousedown", { clientX: 100, bubbles: true })));
  assert.equal(document.body.style.cursor, "col-resize");
  await React.act(async () => document.dispatchEvent(new MouseEvent("mousemove", { clientX: 1000 })));
  assert.equal(sidebar.style.width, "350px");
  await React.act(async () => document.dispatchEvent(new MouseEvent("mousemove", { clientX: -1000 })));
  assert.equal(sidebar.style.width, "236px");
  await React.act(async () => document.dispatchEvent(new MouseEvent("mouseup")));
  assert.equal(document.body.style.cursor, "crosshair");
  assert.equal(document.body.style.userSelect, "text");
  await React.act(async () => handle.dispatchEvent(new MouseEvent("mousedown", { clientX: 100, bubbles: true })));
  await environment.render(null);
  assert.equal(document.body.style.cursor, "crosshair");
  assert.equal(document.body.style.userSelect, "text");
});

void test("peek dismissal uses current dropdown state and cancels on reentry, expansion and unmount", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const changes = [];
  const props = { isCollapsed: true, showPeek: true, toggleCollapsed() {}, togglePeek: (value) => changes.push(value) };
  await environment.render(React.createElement(Harness, props));
  const peek = environment.container.querySelector("aside[aria-label='Sidebar peek view'] > div");
  const leave = () => peek.dispatchEvent(new MouseEvent("pointerout", { bubbles: true, relatedTarget: null }));
  await React.act(async () => leave());
  await environment.render(React.createElement(Harness, { ...props, isAnySidebarDropdownOpen: true }));
  await React.act(async () => t.mock.timers.tick(500));
  assert.deepEqual(changes, [], "opening a dropdown protects an already pending peek dismissal");
  await environment.render(React.createElement(Harness, props));
  await React.act(async () => leave());
  await React.act(async () => t.mock.timers.tick(499));
  assert.deepEqual(changes, []);
  await React.act(async () => t.mock.timers.tick(1));
  assert.deepEqual(changes, [false]);
  changes.length = 0;
  await React.act(async () => leave());
  await React.act(async () =>
    peek.dispatchEvent(new MouseEvent("pointerover", { bubbles: true, relatedTarget: null }))
  );
  await React.act(async () => t.mock.timers.tick(500));
  assert.deepEqual(changes, []);
  await React.act(async () => leave());
  await environment.render(React.createElement(Harness, { ...props, isCollapsed: false }));
  assert.deepEqual(changes, [false]);
  changes.length = 0;
  await React.act(async () => t.mock.timers.tick(500));
  assert.deepEqual(changes, []);
  await environment.render(React.createElement(Harness, props));
  await React.act(async () => leave());
  await environment.render(null);
  await React.act(async () => t.mock.timers.tick(500));
  assert.deepEqual(changes, []);
});

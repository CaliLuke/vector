import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { useAutoScroller } = await import("../../apps/web/core/hooks/use-auto-scroller.tsx");
const { default: useOnlineStatus } = await import("../../apps/web/core/hooks/use-online-status.ts");

void test("auto-scrolling uses vertical proximity and cancels timers on disable, drag end, and unmount", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  t.mock.timers.enable({ apis: ["setInterval"] });
  const scrolls = [];
  const target = document.createElement("div");
  target.getBoundingClientRect = () => new window.DOMRect(0, 0, 200, 200);
  target.scrollBy = (x, y) => scrolls.push([x, y]);
  function Probe({ enabled, topOffset = 0 }) {
    const containerRef = React.useRef(target);
    useAutoScroller(containerRef, enabled, 0, topOffset);
    return null;
  }
  const move = (x, y) => target.dispatchEvent(new MouseEvent("mousemove", { clientX: x, clientY: y }));
  await env.render(React.createElement(Probe, { enabled: true }));
  move(100, 2);
  assert.deepEqual(scrolls, [[0, -6]], "vertical speed must depend on Y, not X");
  t.mock.timers.tick(16);
  assert.equal(scrolls.length, 2);
  await env.render(React.createElement(Probe, { enabled: false }));
  t.mock.timers.tick(64);
  assert.equal(scrolls.length, 2);
  await env.render(React.createElement(Probe, { enabled: true, topOffset: 40 }));
  move(100, 50);
  assert.deepEqual(scrolls.at(-1), [0, -3], "changed offsets must affect scroll thresholds");
  document.dispatchEvent(new Event("dragend"));
  t.mock.timers.tick(64);
  assert.equal(scrolls.length, 3);
  move(198, 100);
  assert.deepEqual(scrolls.at(-1), [6, 0]);
  await env.unmount();
  t.mock.timers.tick(64);
  assert.equal(scrolls.length, 4);
});

void test("connection status follows browser changes and releases its subscription", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  let online = false;
  Object.defineProperty(navigator, "onLine", { configurable: true, get: () => online });
  const remove = t.mock.method(window, "removeEventListener");
  await env.render(React.createElement(ConnectionProbe));
  assert.equal(env.container.textContent, "false");
  await act(async () => {
    online = true;
    window.dispatchEvent(new Event("online"));
  });
  assert.equal(env.container.textContent, "true");
  await act(async () => {
    online = false;
    window.dispatchEvent(new Event("offline"));
  });
  assert.equal(env.container.textContent, "false");
  await env.unmount();
  for (const event of ["online", "offline"]) {
    assert.ok(remove.mock.calls.some(({ arguments: args }) => args[0] === event));
  }
});

function ConnectionProbe() {
  return React.createElement("output", null, String(useOnlineStatus().isOnline));
}

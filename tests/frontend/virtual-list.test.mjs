import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";

registerWebModuleHooks();
const { VirtualList } = await import("../../apps/web/core/components/common/virtual-list.tsx");

function List({ rows }) {
  const scrollRef = React.useRef(null);
  return React.createElement(
    "div",
    { ref: scrollRef, "data-viewport": "true", style: { height: 400, overflowY: "auto" } },
    React.createElement(VirtualList, {
      items: rows,
      getKey: (item) => item.id,
      renderItem: (item) => React.createElement("button", { "data-id": item.id }, item.name),
      scrollRef,
      estimateSize: 40,
      label: "Records",
    })
  );
}

async function fixture(t) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  // Supply browser layout, keeping the real virtualizer, resize and scroll subscriptions.
  t.mock.method(HTMLElement.prototype, "getBoundingClientRect", function () {
    return {
      x: 0,
      y: 0,
      top: 0,
      left: 0,
      right: 400,
      bottom: 40,
      width: 400,
      height: this.dataset.viewport ? 400 : 40,
    };
  });
  Object.defineProperty(HTMLElement.prototype, "clientHeight", {
    configurable: true,
    get() {
      return this.dataset.viewport ? 400 : 40;
    },
  });
  Object.defineProperty(HTMLElement.prototype, "clientWidth", {
    configurable: true,
    get() {
      return 400;
    },
  });
  Object.defineProperty(HTMLElement.prototype, "offsetHeight", {
    configurable: true,
    get() {
      return this.dataset.viewport ? 400 : 40;
    },
  });
  const items = Array.from({ length: 1000 }, (_, index) => ({ id: `row-${index}`, name: `Record ${index}` }));
  const render = (rows = items) => env.render(React.createElement(List, { rows }));
  await render();
  const scroll = async (top) =>
    act(async () => {
      const viewport = env.container.querySelector("[data-viewport]");
      viewport.scrollTop = top;
      viewport.dispatchEvent(new Event("scroll"));
    });
  return { env, items, render, scroll };
}

await test("a long list mounts the viewport, retains focused rows, and reaches distant records", async (t) => {
  const { env, scroll } = await fixture(t);
  assert.ok(env.container.querySelector('[data-id="row-0"]'));
  assert.ok(env.container.querySelectorAll("li").length < 30);
  const focused = env.container.querySelector('[data-id="row-0"]');
  await act(async () => focused.focus());
  await scroll(20_000);
  assert.ok(env.container.querySelector('[data-id="row-500"]'));
  assert.equal(document.activeElement, focused);
  assert.ok(env.container.querySelectorAll("li").length < 35);
  assert.equal(env.container.querySelector('[data-id="row-500"]').parentElement.getAttribute("aria-posinset"), "501");
});

await test("reordering and removing rows preserve record identity and update accessible totals", async (t) => {
  const { env, items, render } = await fixture(t);
  const first = env.container.querySelector('[data-id="row-0"]');
  await render([items[1], items[0], ...items.slice(2, 20)]);
  assert.equal(env.container.querySelector('[data-id="row-0"]'), first);
  assert.equal(first.parentElement.getAttribute("aria-posinset"), "2");
  assert.equal(first.parentElement.getAttribute("aria-setsize"), "20");
  await render([items[1]]);
  assert.equal(env.container.querySelector('[data-id="row-0"]'), null);
  assert.equal(env.container.querySelector("li").getAttribute("aria-setsize"), "1");
  await render([]);
  assert.equal(env.container.querySelectorAll("li").length, 0);
});

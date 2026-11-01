import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { default: RenderIfVisible } = await import("../../apps/web/core/components/core/render-if-visible-HOC.tsx");

void test("virtual rows retain measured height and cancel observer deliveries after unmount", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const globals = ["IntersectionObserver", "requestIdleCallback", "cancelIdleCallback"];
  const descriptors = globals.map((name) => [name, Object.getOwnPropertyDescriptor(globalThis, name)]);
  t.after(async () => {
    await environment.unmount();
    for (const [name, descriptor] of descriptors) {
      if (descriptor) Object.defineProperty(globalThis, name, descriptor);
      else delete globalThis[name];
    }
  });
  const idle = new Map();
  let nextId = 0;
  let observer;
  class ObserverFixture {
    disconnected = false;
    constructor(callback) {
      this.deliver = callback;
      observer = this;
    }
    observe(target) {
      this.target = target;
    }
    disconnect() {
      this.disconnected = true;
    }
  }
  const values = {
    IntersectionObserver: ObserverFixture,
    requestIdleCallback: (callback) => {
      const id = ++nextId;
      idle.set(id, callback);
      return id;
    },
    cancelIdleCallback: (id) => idle.delete(id),
  };
  for (const [name, value] of Object.entries(values))
    Object.defineProperty(globalThis, name, { configurable: true, writable: true, value });
  const flushIdle = () =>
    React.act(async () => {
      const pending = [...idle.values()];
      idle.clear();
      pending.forEach((callback) => callback({ didTimeout: false, timeRemaining: () => 50 }));
    });
  await environment.render(React.createElement(RenderIfVisible, { useIdleTime: true }, "Visible content"));
  const wrapper = environment.container.firstElementChild;
  Object.defineProperty(wrapper, "offsetHeight", { configurable: true, value: 180 });
  assert.equal(wrapper.style.height, "300px");
  observer.deliver([]);
  observer.deliver([{ isIntersecting: true }]);
  assert.equal(wrapper.textContent, "");
  await flushIdle();
  assert.equal(wrapper.textContent, "Visible content");
  await flushIdle();
  observer.deliver([{ isIntersecting: false }]);
  await flushIdle();
  assert.equal(wrapper.textContent, "");
  assert.equal(wrapper.style.height, "180px");
  observer.deliver([{ isIntersecting: true }]);
  assert.equal(idle.size, 1);
  await environment.render(null);
  assert.equal(observer.disconnected, true);
  assert.equal(idle.size, 0);
  observer.deliver([{ isIntersecting: true }]);
  assert.equal(idle.size, 0);
});

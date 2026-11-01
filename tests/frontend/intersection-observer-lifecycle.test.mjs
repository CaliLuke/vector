import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { useIntersectionObserver } = await import("../../apps/web/core/hooks/use-intersection-observer.ts");

function Harness({ containerRef, element, onIntersect, rootMargin }) {
  useIntersectionObserver(containerRef, element, onIntersect, rootMargin);
  return null;
}

void test("intersection observers retain current callbacks and release pending deliveries on replacement and unmount", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const original = Object.getOwnPropertyDescriptor(globalThis, "IntersectionObserver");
  const observers = [];
  class IntersectionObserverFixture {
    disconnected = false;
    constructor(callback, options) {
      this.deliver = callback;
      this.options = options;
      observers.push(this);
    }
    observe(element) {
      this.element = element;
    }
    disconnect() {
      this.disconnected = true;
    }
  }
  Object.defineProperty(globalThis, "IntersectionObserver", { configurable: true, value: IntersectionObserverFixture });
  t.after(async () => {
    await environment.unmount();
    if (original) Object.defineProperty(globalThis, "IntersectionObserver", original);
    else delete globalThis.IntersectionObserver;
  });
  const containerRef = { current: environment.container };
  const element = document.createElement("div");
  const events = [];
  await environment.render(
    React.createElement(Harness, { containerRef, element, onIntersect: () => events.push("old") })
  );
  assert.equal(observers.length, 1);
  assert.equal(observers[0].element, element);
  assert.equal(observers[0].options.root, environment.container);
  observers[0].deliver([]);
  observers[0].deliver([{ isIntersecting: false }]);
  assert.deepEqual(events, []);
  await environment.render(
    React.createElement(Harness, { containerRef, element, onIntersect: () => events.push("current") })
  );
  assert.equal(observers.length, 1, "callback updates must not restart observation or trigger another load");
  observers[0].deliver([{ isIntersecting: true }]);
  assert.deepEqual(events, ["current"]);
  const replacement = document.createElement("div");
  await environment.render(
    React.createElement(Harness, {
      containerRef,
      element: replacement,
      rootMargin: "20%",
      onIntersect: () => events.push("replacement"),
    })
  );
  assert.equal(observers[0].disconnected, true);
  assert.equal(observers[1].options.rootMargin, "20%");
  observers[0].deliver([{ isIntersecting: true }]);
  observers[1].deliver([{ isIntersecting: true }]);
  assert.deepEqual(events, ["current", "replacement"]);
  await environment.render(React.createElement(Harness, { containerRef, element: null }));
  assert.equal(observers[1].disconnected, true);
  observers[1].deliver([{ isIntersecting: true }]);
  assert.deepEqual(events, ["current", "replacement"]);
  await environment.render(React.createElement(Harness, { containerRef, element }));
  observers[2].deliver([{ isIntersecting: true }]);
  await environment.render(null);
  assert.equal(observers[2].disconnected, true);
  observers[2].deliver([{ isIntersecting: true }]);
  assert.deepEqual(events, ["current", "replacement"]);
});

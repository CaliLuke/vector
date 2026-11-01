import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { useHashScroll } = await import("../../packages/hooks/src/use-hash-scroll.ts");

await test("hash scrolling follows route changes, cancels stale requests, and releases timers on unmount", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const scrolled = [];
  const current = React.createRef();
  function Harness({ elementId = "comment", pathname = "/first" }) {
    const value = useHashScroll({ elementId, pathname, scrollDelay: 20 });
    React.useImperativeHandle(current, () => value);
    return React.createElement("div", {
      id: "comment",
      ref: (node) => {
        if (node) node.scrollIntoView = (options) => scrolled.push(options);
      },
    });
  }
  window.history.replaceState(null, "", "/first#comment,other;third|fourth");
  await environment.render(React.createElement(React.StrictMode, null, React.createElement(Harness)));
  assert.deepEqual(current.current.hashIds, ["comment", "other", "third", "fourth"]);
  assert.equal(current.current.isHashMatch, true);
  await act(async () => t.mock.timers.tick(20));
  assert.deepEqual(scrolled, [{ behavior: "smooth", block: "nearest" }]);

  await act(async () => {
    current.current.scrollToElement();
    window.history.replaceState(null, "", "/first#other");
    window.dispatchEvent(new Event("hashchange"));
  });
  await act(async () => t.mock.timers.tick(20));
  assert.equal(current.current.isHashMatch, false);
  assert.equal(scrolled.length, 1);

  window.history.replaceState(null, "", "/second#comment");
  await environment.render(React.createElement(Harness, { pathname: "/second" }));
  await act(async () => t.mock.timers.tick(20));
  assert.equal(scrolled.length, 2);
  assert.equal(current.current.scrollToElement(), true);
  window.history.replaceState(null, "", "/third#comment");
  await act(async () => t.mock.timers.tick(20));
  assert.equal(scrolled.length, 2, "a queued scroll cannot affect a different route");
  assert.equal(current.current.scrollToElement(), false);
  await environment.render(React.createElement(Harness, { elementId: "missing", pathname: "/third" }));
  assert.equal(current.current.scrollToElement(), false);
  await environment.render(React.createElement(Harness, { elementId: "", pathname: "/third" }));
  assert.deepEqual(current.current.hashIds, []);
  await environment.render(React.createElement(Harness, { pathname: "/third" }));
  await environment.unmount();
  await act(async () => t.mock.timers.tick(20));
  assert.equal(scrolled.length, 2);
});

await test("a missing or removed target is harmless and an empty hash does not scroll", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const current = React.createRef();
  function Harness() {
    const value = useHashScroll({ elementId: "target", pathname: "/", scrollDelay: 10 });
    React.useImperativeHandle(current, () => value);
    return null;
  }
  await environment.render(React.createElement(Harness));
  assert.deepEqual(current.current.hashIds, []);
  assert.equal(current.current.scrollToElement(), false);
  const target = document.createElement("div");
  target.id = "target";
  document.body.append(target);
  let scrolled = false;
  target.scrollIntoView = () => {
    scrolled = true;
  };
  assert.equal(current.current.scrollToElement(), true);
  target.remove();
  await act(async () => t.mock.timers.tick(10));
  assert.equal(scrolled, false);
});

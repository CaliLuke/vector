import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();

void test("outside-click detection accepts text and SVG targets, respects protected portals, and releases its listener", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { useOutsideClickDetector } = await import("../../packages/hooks/src/use-outside-click-detector.tsx");
  const closed = [];
  function Fixture({ onClose }) {
    const ref = React.useRef(null);
    useOutsideClickDetector(ref, onClose, true);
    return React.createElement("div", { ref }, "Inside");
  }
  await environment.render(React.createElement(Fixture, { onClose: () => closed.push("first") }));
  const inside = environment.container.firstChild.firstChild;
  inside.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
  const portal = document.createElement("div");
  portal.setAttribute("data-prevent-outside-click", "true");
  portal.innerHTML = "<span>Protected text</span><svg><rect /></svg>";
  document.body.append(portal);
  portal.querySelector("span").firstChild.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
  portal.querySelector("rect").dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
  assert.deepEqual(closed, []);
  await environment.render(React.createElement(Fixture, { onClose: () => closed.push("latest") }));
  document.body.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
  assert.deepEqual(closed, ["latest"]);
  await environment.render(null);
  document.body.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
  assert.deepEqual(closed, ["latest"]);
  portal.remove();
});

void test("reload confirmation uses the latest message and callback and safely handles text and SVG link targets", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { default: useReloadConfirmations } = await import("../../apps/web/core/hooks/use-reload-confirmation.tsx");
  const prompts = [];
  const left = [];
  let allow = false;
  t.mock.method(window, "confirm", (message) => {
    prompts.push(message);
    return allow;
  });
  function Fixture({ message, onLeave, active = true }) {
    useReloadConfirmations(active, message, true, onLeave);
    return React.createElement(
      "a",
      { href: "#next" },
      "Leave",
      React.createElement("svg", null, React.createElement("rect"))
    );
  }
  await environment.render(
    React.createElement(Fixture, { message: "First prompt", onLeave: () => left.push("first") })
  );
  const anchor = environment.container.querySelector("a");
  const denied = new MouseEvent("click", { bubbles: true, cancelable: true });
  anchor.firstChild.dispatchEvent(denied);
  assert.equal(denied.defaultPrevented, true);
  assert.deepEqual(left, []);
  await environment.render(
    React.createElement(Fixture, { message: "Latest prompt", onLeave: () => left.push("latest") })
  );
  allow = true;
  anchor.querySelector("rect").dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }));
  assert.deepEqual(prompts, ["First prompt", "Latest prompt"]);
  assert.deepEqual(left, ["latest"]);
  anchor.dispatchEvent(new MouseEvent("click", { bubbles: true, ctrlKey: true }));
  anchor.target = "_blank";
  anchor.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  assert.equal(prompts.length, 2);
  const unload = new Event("beforeunload", { cancelable: true });
  window.dispatchEvent(unload);
  assert.equal(unload.defaultPrevented, true);
  await environment.render(
    React.createElement(Fixture, { message: "Inactive", active: false, onLeave: () => left.push("inactive") })
  );
  const inactive = new Event("beforeunload", { cancelable: true });
  window.dispatchEvent(inactive);
  assert.equal(inactive.defaultPrevented, false);
});

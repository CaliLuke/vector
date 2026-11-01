import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { default: usePeek } = await import("../../apps/web/core/hooks/use-peek-overview-outside-click.tsx");
const { default: useSidebar } =
  await import("../../apps/web/core/hooks/use-extended-sidebar-overview-outside-click.tsx");

function Panel({ panelRef, preventPanel = false, mounted = true }) {
  return React.createElement(
    "div",
    { "data-prevent-outside-click": preventPanel || undefined, id: "panel-group" },
    mounted &&
      React.createElement("aside", { ref: panelRef }, React.createElement("button", { id: "inside" }, "Inside")),
    React.createElement("button", { id: "group-sibling" }, "Sibling")
  );
}
function PeekHarness({ callback, exclusions, ...props }) {
  const ref = React.useRef(null);
  usePeek(ref, callback, "item", exclusions);
  return React.createElement(Panel, { ...props, panelRef: ref });
}
function SidebarHarness({ callback, ...props }) {
  const ref = React.useRef(null);
  useSidebar(ref, callback, "issue-item");
  return React.createElement(Panel, { ...props, panelRef: ref });
}
async function press(element) {
  await act(async () => element.dispatchEvent(new MouseEvent("mousedown", { bubbles: true })));
}

for (const { name, Harness } of [
  { name: "peek", Harness: PeekHarness },
  { name: "sidebar", Harness: SidebarHarness },
]) {
  void test(`${name} closes on outside clicks, preserves its item and guarded overlays, and uses the latest callback`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    let first = 0;
    let latest = 0;
    const outside = document.createElement("div");
    outside.innerHTML =
      '<button id="outside">Outside</button><div id="issue-item"><button id="row-child">Row</button></div><div data-prevent-outside-click><button id="guarded">Guarded</button></div><div data-delay-outside-click><button id="delayed">Delayed</button></div><svg><path /></svg>';
    document.body.append(outside);
    t.after(() => outside.remove());
    await env.render(React.createElement(Harness, { callback: () => first++ }));
    await press(document.getElementById("inside"));
    await press(document.getElementById("row-child"));
    await press(document.getElementById("guarded"));
    await press(outside.querySelector("path"));
    assert.equal(first, 0);
    await press(document.getElementById("outside"));
    assert.equal(first, 1);
    await env.render(React.createElement(Harness, { callback: () => latest++ }));
    await press(document.getElementById("outside"));
    assert.equal(first, 1);
    assert.equal(latest, 1);
    // The close is queued after the target's own mousedown handling.
    const order = [];
    document.getElementById("delayed").addEventListener("mousedown", () => order.push("target"));
    await env.render(React.createElement(Harness, { callback: () => order.push("close") }));
    document.getElementById("delayed").dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
    assert.deepEqual(order, ["target"]);
    await act(async () => new Promise((resolve) => setTimeout(resolve, 5)));
    assert.deepEqual(order, ["target", "close"]);
    await env.render(React.createElement(Harness, { callback: () => latest++, mounted: false }));
    await press(document.getElementById("outside"));
    assert.equal(latest, 1);
  });
}

void test("peek permits clicks within the panel's own prevention container and only named exclusions bypass other guards", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  let closures = 0;
  const callback = () => closures++;
  await env.render(React.createElement(PeekHarness, { callback, preventPanel: true }));
  await press(document.getElementById("group-sibling"));
  assert.equal(closures, 1);
  await env.render(React.createElement(SidebarHarness, { callback, preventPanel: true }));
  await press(document.getElementById("group-sibling"));
  assert.equal(closures, 1);
  const overlay = document.createElement("div");
  overlay.setAttribute("data-prevent-outside-click", "");
  overlay.innerHTML = "<button>Overlay</button>";
  document.body.append(overlay);
  t.after(() => overlay.remove());
  await env.render(React.createElement(PeekHarness, { callback, exclusions: ["", "allowed-overlay"] }));
  await press(overlay.firstElementChild);
  assert.equal(closures, 1);
  overlay.id = "allowed-overlay";
  await press(overlay.firstElementChild);
  assert.equal(closures, 2);
});

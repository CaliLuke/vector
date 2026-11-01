import assert from "node:assert/strict";
import test from "node:test";
import { React, webRequire } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { PortalWrapper } = await import("../../packages/propel/src/portal/portal-wrapper.tsx");
const portal = (key) => React.createElement(PortalWrapper, { key, portalId: "shared-portal" }, key);

await test("shared portals retain their container until the final owner unmounts", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(React.createElement(React.StrictMode, null, portal("first"), portal("second")));
  const container = document.getElementById("shared-portal");
  assert.ok(container);
  assert.equal(container.textContent, "firstsecond");
  await env.render(React.createElement(React.StrictMode, null, portal("second")));
  assert.equal(document.getElementById("shared-portal"), container);
  assert.equal(container.textContent, "second");
  await env.render(null);
  assert.equal(document.getElementById("shared-portal"), null);
});

await test("portal identity changes release owned containers and retain externally supplied containers", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const external = document.createElement("div");
  external.id = "external-portal";
  external.dataset.portal = "true";
  document.body.append(external);
  const events = [];
  const props = {
    onMount: () => events.push("mount"),
    onUnmount: () => events.push("unmount"),
    className: "portal-content",
  };
  await env.render(React.createElement(PortalWrapper, { ...props, portalId: "owned-portal" }, "Content"));
  assert.equal(document.querySelector("#owned-portal .portal-content")?.textContent, "Content");
  await env.render(React.createElement(PortalWrapper, { ...props, portalId: "external-portal" }, "Moved"));
  assert.equal(document.getElementById("owned-portal"), null);
  assert.equal(external.textContent, "Moved");
  await env.render(null);
  assert.equal(document.getElementById("external-portal"), external);
  assert.equal(external.textContent, "");
  assert.deepEqual(events, ["mount", "unmount", "mount", "unmount"]);
});

await test("portal cleanup preserves content added by another DOM owner", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(React.createElement(PortalWrapper, { portalId: "mixed-portal" }, "React content"));
  const container = document.getElementById("mixed-portal");
  assert.ok(container);
  const foreign = document.createElement("span");
  foreign.textContent = "Other owner";
  container.append(foreign);
  await env.render(null);
  assert.equal(container.isConnected, true);
  assert.equal(container.textContent, "Other owner");
});

await test("server rendering does not create or render a portal container", () => {
  const { renderToString } = webRequire("react-dom/server");
  assert.equal(renderToString(React.createElement(PortalWrapper, null, "Portal content")), "");
});

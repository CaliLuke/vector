import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerAdminModuleHooks } from "./helpers/web-module-hooks.mjs";

registerAdminModuleHooks();
const { renderToString } = webRequire("react-dom/server");
const { SidebarProvider, useSidebar } = await import("../../apps/admin/providers/sidebar.tsx");
function SidebarProbe() {
  const { isSidebarCollapsed, toggleSidebar } = useSidebar();
  return React.createElement(
    "button",
    { onClick: () => toggleSidebar(!isSidebarCollapsed) },
    isSidebarCollapsed ? "collapsed" : "expanded"
  );
}

await test("admin sidebar renders on the server and hydrates the persisted preference without mismatches", async (t) => {
  const element = React.createElement(SidebarProvider, null, React.createElement(SidebarProbe));
  const markup = renderToString(element);
  assert.match(markup, /expanded/);
  const environment = await createReactDomTestEnvironment();
  window.localStorage.setItem("god_mode_sidebar_collapsed", "true");
  const container = document.createElement("div");
  container.innerHTML = markup;
  document.body.appendChild(container);
  const errors = [];
  const { hydrateRoot } = webRequire("react-dom/client");
  let root;
  await act(async () => {
    root = hydrateRoot(container, element, { onRecoverableError: (error) => errors.push(error.message) });
  });
  t.after(async () => {
    await act(async () => root.unmount());
    container.remove();
    await environment.unmount();
  });
  assert.equal(container.textContent, "collapsed");
  assert.deepEqual(errors, []);
  await act(async () => container.querySelector("button").click());
  assert.equal(container.textContent, "expanded");
  assert.equal(window.localStorage.getItem("god_mode_sidebar_collapsed"), "false");
});

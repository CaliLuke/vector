import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createRouterFixture } from "./helpers/router.mjs";

registerWebModuleHooks();
const sidebarSource = new URL("../../apps/web/app/(all)/[workspaceSlug]/(projects)/_sidebar.tsx", import.meta.url).href;
const storageSource = new URL("../../packages/hooks/src/use-local-storage.tsx", import.meta.url).href;
const platformSource = new URL("../../packages/hooks/src/use-platform-os.tsx", import.meta.url).href;
const hooksSource = `export { useLocalStorage } from ${JSON.stringify(storageSource)}; export { usePlatformOS } from ${JSON.stringify(platformSource)};`;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/hooks") {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(hooksSource)}` };
    }
    if (
      context.parentURL?.startsWith("file:") &&
      fileURLToPath(context.parentURL) === fileURLToPath(sidebarSource) &&
      (specifier === "./sidebar" || specifier === "./extended-sidebar")
    ) {
      return {
        shortCircuit: true,
        url: "data:text/javascript,export const AppSidebar = () => null; export const ExtendedAppSidebar = () => null;",
      };
    }
    return nextResolve(specifier, context);
  },
});

const { ProjectAppSidebar } = await import(sidebarSource);
const { SidebarStateProvider, useSidebarActions } = await import("../../apps/web/core/providers/sidebar-state.tsx");

function PeekToggle() {
  const { toggleSidebarPeek } = useSidebarActions();
  return React.createElement("button", { id: "peek", onClick: () => toggleSidebarPeek(true) }, "Peek");
}

await test("project sidebar settles persisted width and saves resizing without writes on unrelated sidebar updates", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  window.localStorage.setItem("sidebarWidth", "280");
  const write = t.mock.method(Object.getPrototypeOf(window.localStorage), "setItem");
  const fixture = createRouterFixture("/acme/");
  let commits = 0;
  const onRender = () => {
    commits++;
    // Fail a render storm promptly, instead of waiting for a process timeout.
    if (commits > 25) throw new Error("Sidebar width persistence did not settle");
  };
  await fixture.render(
    environment,
    React.createElement(
      SidebarStateProvider,
      null,
      React.createElement(React.Profiler, { id: "sidebar", onRender }, React.createElement(ProjectAppSidebar)),
      React.createElement(PeekToggle)
    )
  );
  assert.ok(commits <= 25, "Sidebar width persistence must settle after mounting");
  const sidebar = environment.container.querySelector('[aria-label="Main sidebar"]');
  assert.equal(sidebar.style.width, "280px");
  const handle = sidebar.querySelector('[aria-label="Resize sidebar"]');

  await act(async () => handle.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, clientX: 0 })));
  await act(async () => document.dispatchEvent(new MouseEvent("mousemove", { bubbles: true, clientX: 40 })));
  await act(async () => document.dispatchEvent(new MouseEvent("mouseup", { bubbles: true })));

  assert.equal(sidebar.style.width, "320px");
  assert.equal(window.localStorage.getItem("sidebarWidth"), "320");
  const widthWrites = () => write.mock.calls.filter(({ arguments: args }) => args[0] === "sidebarWidth").length;
  const beforePeek = widthWrites();
  assert.ok(beforePeek > 0);
  await act(async () => environment.container.querySelector("#peek").click());
  assert.equal(widthWrites(), beforePeek);
});

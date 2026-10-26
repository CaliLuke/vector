import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerAdminModuleHooks } from "./helpers/web-module-hooks.mjs";

const { MemoryRouter } = createRequire(new URL("../../apps/admin/package.json", import.meta.url))("react-router");
globalThis.adminRoutingReact = React;
registerAdminModuleHooks();
const stubs = {
  "@/providers/sidebar": "export const useSidebar = () => ({isSidebarCollapsed: false, toggleSidebar() {}});",
  "@/hooks/use-sidebar-menu":
    "export const useSidebarMenu = () => [{href: '/authentication/', name: 'Authentication', Icon: () => null}];",
  "@plane/ui":
    "export const Breadcrumbs = Object.assign(({children}) => children, {Item: ({component}) => component});",
  "@plane/propel/tooltip": "export const Tooltip = ({children}) => children;",
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { AdminHeader } = await import("../../apps/admin/components/common/header/index.tsx");
const { AdminSidebarMenu } = await import("../../apps/admin/app/(all)/(dashboard)/sidebar-menu.tsx");

await test("native slashless routes retain their final breadcrumb and active sidebar item", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(
    React.createElement(
      MemoryRouter,
      { initialEntries: ["/authentication/google"] },
      React.createElement(React.Fragment, null, React.createElement(AdminHeader), React.createElement(AdminSidebarMenu))
    )
  );
  const links = [...environment.container.querySelectorAll("a")];
  assert.ok(
    links.some((link) => link.getAttribute("href") === "/authentication/google" && /google/i.test(link.textContent))
  );
  assert.equal(environment.container.querySelector('a[aria-current="page"]').getAttribute("href"), "/authentication/");
});

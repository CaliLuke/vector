import { createRouterFixture } from "./helpers/router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment, React } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const mockModule = new URL("./helpers/workspace-ui-test-primitives.mjs", import.meta.url).href;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (
      [
        "mobx-react",
        "@headlessui/react",
        "@plane/constants",
        "@plane/i18n",
        "@plane/propel/icons",
        "@plane/utils",
        "lucide-react",
      ].includes(specifier) ||
      (specifier === "../helpers/filter-header" &&
        context.parentURL?.endsWith("/display-filters/display-properties.tsx"))
    ) {
      return { shortCircuit: true, url: mockModule };
    }
    return nextResolve(specifier, context);
  },
});

const [{ SidebarDropdownItem }, { FilterDisplayProperties }, { WorkspaceMenuStateSync }] = await Promise.all([
  import("../../apps/web/core/components/workspace/sidebar/dropdown-item.tsx").then((module) => ({
    SidebarDropdownItem: module.default,
  })),
  import("../../apps/web/core/components/issues/issue-layouts/filters/header/display-filters/display-properties.tsx"),
  import("../../apps/web/core/components/workspace/sidebar/workspace-menu-state-sync.tsx"),
]);

test("workspace selector keeps its action links outside the workspace navigation link", async () => {
  const environment = await createReactDomTestEnvironment();
  const routerFixture = createRouterFixture("/workspace/");
  const workspace = { id: "workspace-id", slug: "default", name: "Default", role: "ADMIN", total_members: 2 };

  try {
    await routerFixture.render(
      environment,
      React.createElement(SidebarDropdownItem, {
        workspace,
        activeWorkspace: workspace,
        handleItemClick() {},
        handleWorkspaceNavigation() {},
        handleClose() {},
      })
    );

    assert.equal(environment.container.querySelectorAll("a").length, 3);
    assert.equal(environment.container.querySelector("a a"), null);
    assert.ok(environment.container.querySelector('a[href="/default/settings"]'));
    assert.ok(environment.container.querySelector('a[href="/default/settings/members"]'));
  } finally {
    await environment.unmount();
  }
});

test("workspace menu open state syncs after commit instead of during Headless UI render", async () => {
  const environment = await createReactDomTestEnvironment();
  const routerFixture = createRouterFixture("/workspace/");
  const errors = [];
  const originalConsoleError = console.error;
  console.error = (...args) => errors.push(args.join(" "));

  function Harness() {
    const [open, setOpen] = React.useState(false);
    const onOpenChange = React.useCallback((nextOpen) => setOpen(nextOpen), []);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement(WorkspaceMenuStateSync, { open: true, onOpenChange }),
      React.createElement("span", { "data-testid": "open-state" }, String(open))
    );
  }

  try {
    await routerFixture.render(environment, React.createElement(Harness));

    assert.equal(environment.container.querySelector('[data-testid="open-state"]').textContent, "true");
    assert.equal(
      errors.some((message) => message.includes("Cannot update a component")),
      false
    );
  } finally {
    console.error = originalConsoleError;
    await environment.unmount();
  }
});

test("display property buttons keep stable keys on the mapped elements", async () => {
  const environment = await createReactDomTestEnvironment();
  const routerFixture = createRouterFixture("/workspace/");
  const errors = [];
  const originalConsoleError = console.error;
  console.error = (...args) => errors.push(args.join(" "));

  try {
    await routerFixture.render(
      environment,
      React.createElement(FilterDisplayProperties, {
        displayProperties: { name: true, priority: false },
        displayPropertiesToRender: ["name", "priority"],
        handleUpdate() {},
      })
    );

    assert.equal(environment.container.querySelectorAll("button").length, 2);
    assert.equal(
      errors.some((message) => message.includes('unique "key" prop')),
      false
    );
  } finally {
    console.error = originalConsoleError;
    await environment.unmount();
  }
});

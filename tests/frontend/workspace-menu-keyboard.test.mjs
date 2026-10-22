import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { act, createReactDomTestEnvironment, React, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const primitives = new URL("./helpers/workspace-ui-test-primitives.mjs", import.meta.url).href;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "next/link" || specifier === "next/navigation") {
      const file = specifier === "next/link" ? "link.tsx" : "navigation.ts";
      return nextResolve(new URL(`../../apps/web/app/compat/next/${file}`, import.meta.url).href, context);
    }
    if (
      ["mobx-react", "@plane/constants", "@plane/i18n", "@plane/propel/icons", "@plane/utils", "lucide-react"].includes(
        specifier
      )
    ) {
      return { shortCircuit: true, url: primitives };
    }
    return nextResolve(specifier, context);
  },
});

const { MemoryRouter, useLocation } = webRequire("react-router");

function CurrentLocation() {
  const { pathname } = useLocation();
  return React.createElement("output", null, pathname);
}

test("workspace menu keyboard selection navigates through the real link", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { Menu } = await import("../../apps/web/node_modules/@headlessui/react/dist/headlessui.esm.js");
  const { default: SidebarDropdownItem } =
    await import("../../apps/web/core/components/workspace/sidebar/dropdown-item.tsx");
  let selectedWorkspace;
  const workspace = { id: "other", slug: "other", name: "Other", role: "MEMBER" };
  await environment.render(
    React.createElement(
      MemoryRouter,
      { initialEntries: ["/default/"] },
      React.createElement(CurrentLocation),
      React.createElement(
        Menu,
        null,
        React.createElement(Menu.Button, null, "Workspaces"),
        React.createElement(
          Menu.Items,
          null,
          React.createElement(SidebarDropdownItem, {
            workspace,
            activeWorkspace: null,
            handleWorkspaceNavigation(value) {
              selectedWorkspace = value;
            },
            handleItemClick() {},
            handleClose() {},
          })
        )
      )
    )
  );

  await act(() => environment.container.querySelector("button").click());
  const menu = document.querySelector("[role=menu]");
  await act(() =>
    menu.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }))
  );
  assert.ok(menu.getAttribute("aria-activedescendant"));
  await act(() => menu.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true })));

  assert.equal(environment.container.querySelector("output").textContent, "/other/");
  assert.equal(selectedWorkspace, workspace);
  assert.ok(!document.querySelector("[role=menu]"), "selection should close the menu");
});

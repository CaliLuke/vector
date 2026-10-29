import { routerApi } from "../../../tests/frontend/helpers/router.mjs";
import { createQueryRouterFixture } from "../../../tests/frontend/helpers/query-router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { act, React } from "../../../tests/frontend/helpers/react.mjs";
import { registerWebModuleHooks } from "../../../tests/frontend/helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const primitives = new URL("../../../tests/frontend/helpers/workspace-ui-test-primitives.mjs", import.meta.url).href;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (
      context.parentURL?.endsWith("/workspace/sidebar/dropdown-item.tsx") &&
      [
        "@plane/i18n",
        "@plane/propel/icons",
        "@plane/utils/common",
        "@plane/utils/file",
        "@plane/utils/permission",
        "lucide-react",
      ].includes(specifier)
    ) {
      return { shortCircuit: true, url: primitives };
    }
    return nextResolve(specifier, context);
  },
});

const { useLocation } = routerApi;
const identityQueries = await import("../core/queries/identity.ts");

function CurrentLocation() {
  const { pathname } = useLocation();
  return React.createElement("output", null, pathname);
}

await test("workspace menu keyboard selection navigates through the real link", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/default/");
  fixture.client.setQueryData(identityQueries.currentUserOptions().queryKey, { id: "alice" });
  const { Menu } = await import("@headlessui/react");
  const { default: SidebarDropdownItem } = await import("../core/components/workspace/sidebar/dropdown-item.tsx");
  let selectedWorkspace;
  const workspace = { id: "other", slug: "other", name: "Other", role: "MEMBER" };
  await fixture.render(
    React.createElement(
      React.Fragment,
      null,
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

  await act(() => fixture.environment.container.querySelector("button").click());
  const menu = document.querySelector("[role=menu]");
  await act(() =>
    menu.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }))
  );
  assert.ok(menu.getAttribute("aria-activedescendant"));
  await act(() => menu.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true })));

  assert.equal(fixture.environment.container.querySelector("output").textContent, "/other/");
  assert.equal(selectedWorkspace, workspace);
  assert.ok(!document.querySelector("[role=menu]"), "selection should close the menu");
});

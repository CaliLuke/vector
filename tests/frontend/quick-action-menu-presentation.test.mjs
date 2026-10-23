import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.React = React;
globalThis.quickActionMenuCalls = [];

const customMenuUrl = new URL("../../packages/ui/src/dropdowns/custom-menu.tsx", import.meta.url).href;
const packageStubs = {
  "@plane/ui": `export const ContextMenu = ({ items }) => globalThis.React.createElement("div", null, items.length); export { CustomMenu } from ${JSON.stringify(customMenuUrl)};`,
  "@plane/hooks": "export const useOutsideClickDetector = () => {};",
  "@plane/propel/icons": "export const ChevronDownIcon = () => null; export const ChevronRightIcon = () => null;",
  "lucide-react": "export const MoreHorizontal = () => null;",
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
  "@plane/utils": "export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');",
};

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "../utils" && context.parentURL?.includes("/packages/ui/src/dropdowns/")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(packageStubs["@plane/utils"])}`,
      };
    }
    return resolvePackageStub(specifier, packageStubs, context, nextResolve);
  },
});

const action = () => globalThis.quickActionMenuCalls.push("enabled");
const nestedAction = () => globalThis.quickActionMenuCalls.push("nested");
const Icon = () => React.createElement("span", { "data-menu-icon": "true" });

await test("quick action menu keeps visible states, nested actions, and selection behavior", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { QuickActionMenu } =
    await import("../../apps/web/core/components/issues/issue-layouts/quick-action-dropdowns/quick-action-menu.tsx");
  const parentRef = React.createRef();
  parentRef.current = document.createElement("div");
  const portalElement = document.createElement("div");
  document.body.append(portalElement);
  t.after(() => portalElement.remove());

  await environment.render(
    React.createElement(QuickActionMenu, {
      parentRef,
      customMenuProps: {
        placement: "bottom-end",
        portalElement,
        useCaptureForOutsideClick: true,
        customButtonAsChild: true,
        closeOnSelect: true,
        customButton: React.createElement("button", { type: "button" }, "Open actions"),
      },
      items: [
        { key: "hidden", title: "Hidden", action, shouldRender: false },
        { key: "disabled", title: "Disabled action", description: "Unavailable", action, disabled: true, icon: Icon },
        {
          key: "nested",
          title: "Move item",
          description: "Choose a destination",
          icon: Icon,
          action,
          nestedMenuItems: [
            { key: "nested-choice", title: "Cycle A", description: "Current cycle", action: nestedAction, icon: Icon },
          ],
        },
        { key: "enabled", title: "Enabled action", action },
      ],
    })
  );

  const menuTrigger = environment.container.querySelector("button");
  assert.ok(menuTrigger);
  await React.act(async () => menuTrigger.click());
  const menuButtons = () => [...portalElement.querySelectorAll("button")];
  const disabled = menuButtons().find((button) => button.textContent.includes("Disabled action"));
  assert.ok(disabled);
  assert.equal(disabled.disabled, true);
  assert.doesNotMatch(portalElement.textContent, /Hidden/);
  assert.match(portalElement.textContent, /Choose a destination/);

  const submenuTrigger = menuButtons().find((button) => button.textContent.includes("Choose a destination"));
  assert.ok(submenuTrigger);
  await React.act(async () => submenuTrigger.click());
  const nestedChoice = [...document.querySelectorAll("button")].find((button) =>
    button.textContent.includes("Current cycle")
  );
  assert.ok(nestedChoice, "the nested menu should open its choice");
  assert.ok(nestedChoice.querySelector('[data-menu-icon="true"]'));
  await React.act(async () => nestedChoice.click());
  assert.deepEqual(globalThis.quickActionMenuCalls, ["nested"]);

  await React.act(async () => menuTrigger.click());
  const enabled = menuButtons().find((button) => button.textContent.includes("Enabled action"));
  assert.ok(enabled);
  await React.act(async () => enabled.click());
  assert.deepEqual(globalThis.quickActionMenuCalls, ["nested", "enabled"]);
  assert.equal(portalElement.querySelector("[role='menuitem']"), null, "selecting an item should close the menu");
});

import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const toastCalls = [];
const packageStubs = {
  "@plane/hooks": "export const useOutsideClickDetector = () => {};",
  "@plane/propel/icons": `export const ChevronDownIcon = () => null; export const ChevronRightIcon = () => null;`,
  "@plane/propel/toast": `export const TOAST_TYPE = { SUCCESS: "success" }; export const setToast = (...args) => globalThis.__identifierToastCalls.push(args);`,
  "@plane/utils": `export const cn = (...values) => values.filter(value => typeof value === "string").join(" ");`,
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
};
globalThis.__identifierToastCalls = toastCalls;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "../utils" && context.parentURL?.includes("/packages/ui/src/dropdowns/")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');")}`,
      };
    }
    if (Object.hasOwn(packageStubs, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(packageStubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

const { CustomMenu } = await import("../../packages/ui/src/dropdowns/custom-menu.tsx");
const { IdentifierText } = await import("../../apps/web/core/components/issues/issue-detail/identifier-text.tsx");

test("passive identifier composes with a real CustomMenu button without nesting interactive elements", async () => {
  const environment = await createReactDomTestEnvironment();
  await environment.render(
    React.createElement(
      CustomMenu,
      {
        customButton: React.createElement(
          "button",
          { type: "button", "aria-label": "Choose parent" },
          React.createElement(IdentifierText, { identifier: "PLN-42" })
        ),
        customButtonAsChild: true,
      },
      React.createElement(CustomMenu.MenuItem, null, "Create child")
    )
  );

  const menuTrigger = environment.container.querySelector("button[aria-label='Choose parent']");
  assert.ok(menuTrigger);
  assert.equal(menuTrigger.querySelectorAll("button").length, 0);
  assert.equal(menuTrigger.textContent, "PLN-42");
  const passiveIdentifier = menuTrigger.querySelector("span");
  assert.ok(passiveIdentifier);
  assert.equal(passiveIdentifier.tabIndex, -1, "passive identifier text is not focusable");

  await React.act(() => {
    menuTrigger.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
  });
  assert.ok([...document.querySelectorAll("button")].some((button) => button.textContent.includes("Create child")));
  await environment.unmount();
});

test("enabled identifier keeps its real copy tooltip and copies the identifier", async () => {
  const environment = await createReactDomTestEnvironment();
  const writes = [];
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText: async (value) => writes.push(value) },
  });
  toastCalls.length = 0;
  await environment.render(
    React.createElement(IdentifierText, { identifier: "PLN-42", enableClickToCopyIdentifier: true })
  );

  const copyButton = environment.container.querySelector("button");
  assert.ok(copyButton);
  assert.equal(copyButton.textContent, "PLN-42");
  await React.act(async () => copyButton.click());
  assert.deepEqual(writes, ["PLN-42"]);
  assert.equal(toastCalls.length, 1);
  await environment.unmount();
});

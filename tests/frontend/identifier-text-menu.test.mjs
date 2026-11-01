import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerDropdownSourceModules } from "./helpers/dropdown-source-modules.mjs";

const toastCalls = [];
const packageStubs = {
  "@plane/hooks": "export const useOutsideClickDetector = () => {};",
  "@plane/propel/icons": `export const ChevronDownIcon = () => null; export const ChevronRightIcon = () => null;`,
  "@plane/propel/toast": `export const TOAST_TYPE = { SUCCESS: "success" }; export const setToast = (...args) => globalThis.identifierToastCalls.push(args);`,
  "@plane/utils/common":
    'export const cn = (...values) => values.filter(value => typeof value === "string").join(" ");',
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
};
globalThis.identifierToastCalls = toastCalls;
registerDropdownSourceModules(undefined, packageStubs);

// Register eagerly; node:test owns completion and reports failures before suite cleanup.
void test("passive identifier composes with a real CustomMenu button without nesting interactive elements", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { CustomMenu } = await import("../../packages/ui/src/dropdowns/custom-menu.tsx");
  const { IdentifierText } = await import("../../apps/web/core/components/issues/issue-detail/identifier-text.tsx");
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

  await React.act(async () => {
    menuTrigger.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
  });
  await React.act(async () => {
    await new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve)));
  });
  assert.ok([...document.querySelectorAll("button")].some((button) => button.textContent.includes("Create child")));
  await environment.unmount();
});

void test("enabled identifier keeps its real copy tooltip and copies the identifier", async () => {
  const environment = await createReactDomTestEnvironment();
  const { IdentifierText } = await import("../../apps/web/core/components/issues/issue-detail/identifier-text.tsx");
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

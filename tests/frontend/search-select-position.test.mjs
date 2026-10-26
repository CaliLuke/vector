import { setPositioningGeometry } from "./helpers/positioning.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const stubs = {
  "@plane/propel/icons": `export const ChevronDownIcon = () => null;
    export const CheckIcon = () => null; export const SearchIcon = () => null;`,
  "@plane/propel/tooltip": "export const Tooltip = ({ children }) => children;",
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "../utils" && context.parentURL?.includes("/packages/ui/src/dropdowns/"))
      return nextResolve(new URL("../../packages/ui/src/utils/classname.tsx", import.meta.url).href, context);
    if (specifier === "@plane/hooks")
      return nextResolve(
        new URL("../../packages/hooks/src/use-outside-click-detector.tsx", import.meta.url).href,
        context
      );
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});

const variants = {
  label: { label: "State" },
  content: { customButton: React.createElement("span", null, "State") },
  child: { customButton: React.createElement("button", null, "State"), customButtonAsChild: true },
};
for (const [name, buttonProps] of Object.entries(variants)) {
  void test(`search select positions its portal at the ${name} trigger`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const { CustomSearchSelect } = await import("../../packages/ui/src/dropdowns/custom-search-select.tsx");
    await environment.render(
      React.createElement(CustomSearchSelect, {
        ...buttonProps,
        value: ["backlog"],
        multiple: true,
        onChange: () => {},
        options: [{ value: "backlog", query: "backlog", content: "Backlog" }],
      })
    );
    const trigger = environment.container.querySelector("button");
    setPositioningGeometry(trigger);
    await act(async () => trigger.click());
    const menu = document.querySelector('[role="listbox"]');
    assert.ok(menu, "the options are open");
    assert.equal(menu.getAttribute("data-popper-placement"), "bottom-start");
    assert.equal(menu.style.transform, "translate(400px, 132px)");
  });
}

await test("search-select options remain accessible inside a modal's focus boundary", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { CustomSearchSelect } = await import("../../packages/ui/src/dropdowns/custom-search-select.tsx");
  const { Dialog } = webRequire("@headlessui/react");
  let closed = 0;
  await env.render(
    React.createElement(
      Dialog,
      {
        open: true,
        onClose: () => {
          closed++;
        },
        "aria-label": "Update item",
      },
      React.createElement(
        Dialog.Panel,
        null,
        React.createElement("input", { "aria-label": "Title", defaultValue: "Existing item" }),
        React.createElement(CustomSearchSelect, {
          label: "Type",
          value: "task",
          onChange: () => {},
          options: [{ value: "task", query: "task", content: "Task" }],
        })
      )
    )
  );
  const trigger = [...document.querySelectorAll("button")].find((button) => button.textContent === "Type");
  assert.ok(trigger);
  await act(async () => document.querySelector('input[aria-label="Title"]').focus());
  await act(async () => {
    trigger.dispatchEvent(
      new window.MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0, detail: 1 })
    );
    trigger.dispatchEvent(new window.MouseEvent("mouseup", { bubbles: true, cancelable: true, button: 0, detail: 1 }));
    trigger.dispatchEvent(new window.MouseEvent("click", { bubbles: true, cancelable: true, button: 0, detail: 1 }));
  });
  const input = document.querySelector('input[placeholder="Search"]');
  assert.ok(input);
  assert.equal(input.closest('[aria-hidden="true"], [inert]'), null);
  assert.equal(document.activeElement, input);
  await act(async () =>
    input.dispatchEvent(new window.KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
  );
  assert.equal(document.querySelector('[role="listbox"]'), null);
  assert.equal(closed, 0);
  assert.equal(document.activeElement, trigger);
});

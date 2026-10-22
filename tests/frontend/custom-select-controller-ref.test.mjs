import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { React, act, createReactDomTestEnvironment, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const reactModules = Object.fromEntries(
  ["react", "react/jsx-runtime", "react/jsx-dev-runtime"].map((name) => [
    name,
    pathToFileURL(webRequire.resolve(name)).href,
  ])
);
const customSelectUrl = new URL("../../packages/ui/src/dropdowns/custom-select.tsx", import.meta.url).href;
const outsideClickUrl = new URL("../../packages/hooks/src/use-outside-click-detector.tsx", import.meta.url).href;
const substitutes = {
  "@plane/propel/icons": "export const CheckIcon = () => null; export const ChevronDownIcon = () => null;",
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
  "@plane/ui": `export { CustomSelect } from ${JSON.stringify(customSelectUrl)};`,
  "@plane/utils": "export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');",
};

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/hooks") return { shortCircuit: true, url: outsideClickUrl };
    if (specifier === "../utils" && context.parentURL?.endsWith("/dropdowns/custom-select.tsx")) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(substitutes["@plane/utils"])}`,
      };
    }
    if (Object.hasOwn(substitutes, specifier)) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(substitutes[specifier])}` };
    }
    if (Object.hasOwn(reactModules, specifier)) {
      return { shortCircuit: true, url: reactModules[specifier] };
    }
    return nextResolve(specifier, context);
  },
});

test("CustomSelect forwards the react-hook-form Controller ref to its trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  const { CustomSelect } = await import("../../packages/ui/src/dropdowns/custom-select.tsx");
  const { Controller, useForm } = webRequire("react-hook-form");
  const fieldRef = React.createRef();

  function Harness() {
    const { control } = useForm({ defaultValues: { role: "member" } });
    return React.createElement(Controller, {
      name: "role",
      control,
      render: ({ field }) =>
        React.createElement(
          CustomSelect,
          {
            ...field,
            ref: (node) => {
              field.ref(node);
              fieldRef.current = node;
            },
            customButton: React.createElement("span", null, field.value),
          },
          React.createElement(CustomSelect.Option, { value: "member" }, "Member"),
          React.createElement(CustomSelect.Option, { value: "admin" }, "Admin")
        ),
    });
  }

  await environment.render(React.createElement(Harness));

  const trigger = environment.container.querySelector("button");
  assert.equal(fieldRef.current, trigger, "Controller ref should point to the focusable select trigger");
  assert.deepEqual(errors, [], "passing a Controller ref should not warn");
  trigger.focus();
  await act(() => trigger.click());
  assert.ok(document.querySelector('[role="listbox"]'), "click should open the real select options");
  await act(() =>
    trigger.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
  );
  assert.equal(document.querySelector('[role="listbox"]'), null, "Escape should close the options");
});

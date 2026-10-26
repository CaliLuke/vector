import { registerDropdownSourceModules } from "./helpers/dropdown-source-modules.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";

const customSelectUrl = new URL("../../packages/ui/src/dropdowns/custom-select.tsx", import.meta.url).href;
const substitutes = {
  "@plane/propel/icons": "export const CheckIcon = () => null; export const ChevronDownIcon = () => null;",
  "react-popper": "export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });",
  "@plane/ui": `export { CustomSelect } from ${JSON.stringify(customSelectUrl)};`,
  "@plane/utils/common":
    "export const cn = (...values) => values.filter(value => typeof value === 'string').join(' ');",
};

registerDropdownSourceModules("custom-select.tsx", substitutes);

test("CustomSelect forwards the TanStack Form field ref to its trigger", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  const { CustomSelect } = await import("../../packages/ui/src/dropdowns/custom-select.tsx");
  const { useForm } = webRequire("@tanstack/react-form");
  const fieldRef = React.createRef();

  function Harness() {
    const form = useForm({ defaultValues: { role: "member" } });
    return React.createElement(form.Field, { name: "role" }, (field) =>
      React.createElement(
        CustomSelect,
        {
          value: field.state.value,
          onChange: field.handleChange,
          onBlur: field.handleBlur,
          ref: (node) => {
            fieldRef.current = node;
          },
          customButton: React.createElement("span", null, field.state.value),
        },
        React.createElement(CustomSelect.Option, { value: "member" }, "Member"),
        React.createElement(CustomSelect.Option, { value: "admin" }, "Admin")
      )
    );
  }

  await environment.render(React.createElement(Harness));

  const trigger = environment.container.querySelector("button");
  assert.equal(fieldRef.current, trigger, "Field ref should point to the focusable select trigger");
  assert.deepEqual(errors, [], "passing a Field ref should not warn");
  trigger.focus();
  await act(() => trigger.click());
  assert.ok(document.querySelector('[role="listbox"]'), "click should open the real select options");
  await act(() =>
    trigger.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
  );
  assert.equal(document.querySelector('[role="listbox"]'), null, "Escape should close the options");
});

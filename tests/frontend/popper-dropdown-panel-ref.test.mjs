import assert from "node:assert/strict";
import { test } from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { createModuleStub } from "./helpers/module-stubs.mjs";
import { clickWithPointer } from "./helpers/pointer.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

registerWebTestModules(
  {
    "@plane/hooks": "export const useOutsideClickDetector = () => {};",
    "@plane/propel/icons":
      "export const CheckIcon = () => null; export const ChevronDownIcon = () => null; export const SearchIcon = () => null;",
    "@plane/utils/common": `export const cn = (...values) => values.flatMap((value) => typeof value === "string" ? [value] : Object.entries(value || {}).filter(([, enabled]) => enabled).map(([key]) => key)).join(" ");`,
  },
  (specifier, context, resolveStub) => {
    if (specifier.endsWith("utils") && context.parentURL?.includes("/packages/ui/src/")) {
      return createModuleStub(
        `export const cn = (...values) => values.flatMap((value) => typeof value === "string" ? [value] : Object.entries(value || {}).filter(([, enabled]) => enabled).map(([key]) => key)).join(" ");`
      );
    }
    return resolveStub(specifier, context);
  }
);

async function openAndAssertPopperPanel(t, Component, value, selectedValue) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const changes = [];

  await environment.render(
    React.createElement(Component, {
      value,
      onChange(nextValue) {
        changes.push(nextValue);
      },
      options: [{ value: "one", data: { one: "one" } }],
      keyExtractor: (option) => option.value,
      disableSearch: true,
      buttonContent: () => "Choose",
    })
  );

  const trigger = environment.container.querySelector("button");
  assert.ok(trigger);
  trigger.getBoundingClientRect = () => ({
    x: 180,
    y: 90,
    left: 180,
    top: 90,
    right: 300,
    bottom: 122,
    width: 120,
    height: 32,
    toJSON() {},
  });
  await act(async () => trigger.click());

  const panel = document.querySelector('[role="listbox"]');
  assert.ok(panel, "opening the dropdown renders its options panel");
  assert.match(panel.getAttribute("data-popper-placement") ?? "", /^(top|bottom)-start$/);
  assert.equal(panel.style.position, "absolute", "Popper positions the public options element");
  assert.notEqual(panel.style.transform, "", "Popper applies coordinates to the public options element");

  if (selectedValue !== undefined) {
    const option = panel.querySelector('[role="option"]');
    assert.ok(option);
    await clickWithPointer(option);
    assert.deepEqual(changes, [selectedValue]);
  }
}

await test("single-select options element receives the real Popper ref and positioning", async (t) => {
  const { Dropdown } = await import("../../packages/ui/src/dropdown/single-select.tsx");
  await openAndAssertPopperPanel(t, Dropdown, "one", "one");
});

await test("multi-select options element receives the real Popper ref and positioning", async (t) => {
  const { MultiSelectDropdown } = await import("../../packages/ui/src/dropdown/multi-select.tsx");
  await openAndAssertPopperPanel(t, MultiSelectDropdown, [], ["one"]);
});

await test("custom select options element receives the real Popper ref and positioning", async (t) => {
  const { CustomSelect } = await import("../../packages/ui/src/dropdowns/custom-select.tsx");
  await openAndAssertPopperPanel(
    t,
    (props) =>
      React.createElement(
        CustomSelect,
        { ...props, value: "one", customButton: React.createElement("span", null, "Choose") },
        React.createElement(CustomSelect.Option, { value: "one" }, "One")
      ),
    "one"
  );
});

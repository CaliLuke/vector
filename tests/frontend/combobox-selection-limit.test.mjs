import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { clickWithPointer } from "./helpers/pointer.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();

async function openAndChoose(label) {
  if (!document.querySelector('[role="option"]')) {
    await clickWithPointer(document.querySelector("button"));
  }
  const option = [...document.querySelectorAll('[role="option"]')].find((candidate) => candidate.textContent === label);
  assert.ok(option, `option ${label} should be visible`);
  await clickWithPointer(option);
}

for (const controlled of [false, true]) {
  void test(`selection limits prevent extra values and allow deselection in ${controlled ? "controlled" : "uncontrolled"} comboboxes`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const { Combobox } = await import("../../packages/propel/src/combobox/combobox.tsx");
    const changes = [];
    function Fixture() {
      const [value, setValue] = React.useState(["one"]);
      return React.createElement(
        Combobox,
        {
          multiSelect: true,
          maxSelections: 1,
          ...(controlled ? { value } : { defaultValue: ["one"] }),
          onValueChange: (next) => {
            changes.push(next);
            setValue(next);
          },
        },
        React.createElement(Combobox.Button, null, "Choose"),
        React.createElement(
          Combobox.Options,
          null,
          React.createElement(Combobox.Option, { value: "one" }, "One"),
          React.createElement(Combobox.Option, { value: "two" }, "Two")
        )
      );
    }
    await environment.render(React.createElement(Fixture));
    await openAndChoose("Two");
    assert.deepEqual(changes, []);
    await openAndChoose("One");
    await openAndChoose("Two");
    assert.deepEqual(changes, [[], ["two"]]);
  });
}

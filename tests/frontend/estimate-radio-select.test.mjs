import assert from "node:assert/strict";
import test from "node:test";
import { format } from "node:util";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

test("estimate radio choices use native changes once and respect disabled choices", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { RadioInput } = await import("../../apps/web/core/components/estimates/radio-select.tsx");
  const changes = [];
  const warnings = [];
  const originalError = console.error;
  console.error = (...args) => warnings.push(format(...args));
  t.after(() => {
    console.error = originalError;
  });
  function Choices() {
    const [selected, setSelected] = React.useState("points");
    return React.createElement(RadioInput, {
      name: "estimate",
      label: "Estimate system",
      selected,
      options: [
        { label: "Points", value: "points" },
        { label: "Categories", value: "categories" },
        { label: "Time", value: "time", disabled: true },
      ],
      onChange(value) {
        changes.push(value);
        setSelected(value);
      },
    });
  }
  await environment.render(React.createElement(Choices));
  const points = environment.container.querySelector('input[value="points"]');
  const categories = environment.container.querySelector('input[value="categories"]');
  const disabled = environment.container.querySelector('input[value="time"]');
  await act(async () => environment.container.querySelector('label[for="estimate_1"]').click());
  assert.equal(categories.checked, true);
  assert.equal(points.checked, false);
  assert.deepEqual(changes, ["categories"]);
  await act(async () => {
    disabled.click();
    environment.container.querySelector('label[for="estimate_2"]').click();
  });
  assert.equal(disabled.checked, false);
  assert.deepEqual(changes, ["categories"]);
  await act(async () => {
    points.focus();
    points.click();
  });
  assert.equal(document.activeElement, points);
  assert.equal(points.checked, true);
  assert.deepEqual(changes, ["categories", "points"]);
  assert.equal(
    warnings.some((message) => message.includes("checked") && message.includes("onChange")),
    false
  );
});

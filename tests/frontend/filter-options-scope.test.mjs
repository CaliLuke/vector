import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();

await test("a late select catalog cannot restore names from the previous scope", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { MultiSelectFilterValueInput } =
    await import("../../apps/web/core/components/rich-filters/filter-value-input/select/multi.tsx");
  let resolveOld;
  const oldConfig = {
    type: "multi_select",
    getOptions: () =>
      new Promise((resolve) => {
        resolveOld = resolve;
      }),
  };
  const currentConfig = {
    type: "multi_select",
    getOptions: () => [{ id: "new", value: "new", label: "Current type" }],
  };
  const changes = [];
  const draw = (config) =>
    env.render(
      React.createElement(MultiSelectFilterValueInput, {
        config,
        condition: { id: "condition", property: "type_id", operator: "in", value: ["old", "new"] },
        onChange: (value) => changes.push(value),
      })
    );
  await draw(oldConfig);
  await draw(currentConfig);
  assert.match(env.container.textContent, /Current type/);
  await act(async () => resolveOld([{ id: "old", value: "old", label: "Previous private type" }]));
  assert.match(env.container.textContent, /Current type/);
  assert.match(env.container.textContent, /Unavailable \(old\)/);
  assert.doesNotMatch(env.container.textContent, /Previous private type/);
  assert.deepEqual(changes, []);
});

import { registerWebTestModules, reactModuleUrl as reactUrl } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";
import test from "node:test";

import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";

const stubs = {
  "@/components/dropdowns/date-range": `import React from ${JSON.stringify(reactUrl)};
    export const DateRangeDropdown=({value,onSelect})=>React.createElement("button",{
      onClick:()=>onSelect(value)
    },[value.from?.getDate(),value.to?.getDate()].join(" to "));`,
};
registerWebTestModules(stubs);

await test("saved calendar-day ranges display and resubmit the same dates west of UTC", async (t) => {
  const previousTZ = process.env.TZ;
  process.env.TZ = "America/Los_Angeles";
  t.after(() => (previousTZ === undefined ? delete process.env.TZ : (process.env.TZ = previousTZ)));
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { DateRangeFilterValueInput } =
    await import("../../apps/web/core/components/rich-filters/filter-value-input/date/range.tsx");
  const changes = [];
  await env.render(
    React.createElement(DateRangeFilterValueInput, {
      config: { type: "date_range" },
      condition: { id: "c", property: "customproperty_date", operator: "range", value: ["2026-10-25", "2026-10-31"] },
      onChange: (value) => changes.push(value),
    })
  );
  assert.equal(env.container.textContent, "25 to 31");
  await act(async () => env.container.querySelector("button").click());
  assert.deepEqual(changes, [["2026-10-25", "2026-10-31"]]);
});

await test("a saved false selection stays closed and displays its selected label", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { SingleSelectFilterValueInput } =
    await import("../../apps/web/core/components/rich-filters/filter-value-input/select/single.tsx");
  await env.render(
    React.createElement(SingleSelectFilterValueInput, {
      config: { type: "single_select", getOptions: () => [{ id: "false", value: false, label: "False" }] },
      condition: { id: "c", property: "customproperty_boolean", operator: "exact", value: false },
      onChange: () => assert.fail("loading a saved value must not change it"),
    })
  );
  assert.equal(env.container.textContent, "False");
  assert.equal(document.querySelector('[role="listbox"]'), null);
  assert.equal(env.container.querySelector("button").getAttribute("aria-expanded"), "false");
});

await test("selection filters remain controlled while operands are entered and reset", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  const { SingleSelectFilterValueInput } =
    await import("../../apps/web/core/components/rich-filters/filter-value-input/select/single.tsx");
  const config = { type: "single_select", getOptions: () => [{ id: "false", value: false, label: "False" }] };
  const draw = (value) =>
    env.render(
      React.createElement(SingleSelectFilterValueInput, {
        config,
        condition: { id: "c", property: "customproperty_boolean", operator: "exact", value },
        isDisabled: true,
        onChange: () => assert.fail("rerendering must not mutate the predicate"),
      })
    );
  await draw(undefined);
  await draw(false);
  await draw(undefined);
  assert.deepEqual(errors, []);
});

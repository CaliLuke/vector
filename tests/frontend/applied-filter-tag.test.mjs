import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
registerWebTestModules({ "@plane/i18n": "export const useTranslation=()=>({t:key=>key});" });
const child = (key) => React.createElement("span", null, `${key} value`);

await test("applied filter tags preserve values and expose removal only to editable consumers", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { AppliedFilterTag } = await import("../../apps/web/core/components/common/applied-filters/tag.tsx");
  let removals = 0;
  await env.render(
    React.createElement(
      AppliedFilterTag,
      {
        filterKey: "start_date",
        value: ["2026-10-31"],
        onClear: () => removals++,
        valuesClassName: "values-row",
      },
      "October 31"
    )
  );
  assert.match(env.container.textContent, /start date.*October 31/i);
  const remove = env.container.querySelector('button[aria-label="Remove start date filter"]');
  assert.ok(remove);
  assert.equal(remove.type, "button");
  await act(async () => remove.click());
  assert.equal(removals, 1);
  await env.render(
    React.createElement(AppliedFilterTag, { filterKey: "start_date", value: ["2026-10-31"] }, "October 31")
  );
  assert.match(env.container.textContent, /October 31/);
  assert.equal(env.container.querySelector("button"), null);
  await env.render(React.createElement(AppliedFilterTag, { filterKey: "start_date", value: [] }, "October 31"));
  assert.equal(env.container.textContent, "");
  await env.render(React.createElement(AppliedFilterTag, { filterKey: "favorites", value: false }, "Favorite"));
  assert.equal(env.container.textContent, "");
});

await test("filter rows preserve stored order, omit empty and unknown filters, and clear the correct scope", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { AppliedFiltersRow } = await import("../../apps/web/core/components/common/applied-filters/tag.tsx");
  const removed = [];
  let cleared = 0;
  const props = {
    filters: { status: ["active"], ignored: ["unknown"], start_date: [], owned_by: ["owner"] },
    keys: ["owned_by", "start_date", "status"],
    onRemove: (key) => removed.push(key),
    onClearAll: () => cleared++,
  };
  await env.render(React.createElement(AppliedFiltersRow, props, child));
  assert.match(env.container.textContent, /status.*owned by.*common.clear_all/s);
  assert.doesNotMatch(env.container.textContent, /ignored|start date/);
  const buttons = [...env.container.querySelectorAll("button")];
  assert.equal(buttons.length, 3);
  await act(async () => buttons[0].click());
  assert.deepEqual(removed, ["status"]);
  await act(async () => buttons[2].click());
  assert.equal(cleared, 1);
  await env.render(
    React.createElement(AppliedFiltersRow, { ...props, onRemove: undefined, onClearAll: undefined }, child)
  );
  assert.equal(env.container.querySelector("button"), null);
  await env.render(React.createElement(AppliedFiltersRow, { ...props, filters: {} }, child));
  assert.equal(env.container.textContent, "");
});

import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

await test("date filter presets retain their labels and removal respects edit permission", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { AppliedDateFilters } = await import("../../apps/web/core/components/common/applied-filters/date.tsx");
  const { DATE_AFTER_FILTER_OPTIONS, PROJECT_CREATED_AT_FILTER_OPTIONS } =
    await import("../../packages/constants/dist/index.mjs");
  const removed = [];
  const handleRemove = (value) => removed.push(value);

  await env.render(
    React.createElement(AppliedDateFilters, {
      editable: true,
      values: ["1_weeks;after;fromnow"],
      options: DATE_AFTER_FILTER_OPTIONS,
      className: "bg-layer-3",
      handleRemove,
    })
  );
  assert.equal(env.container.textContent, "1 week from now");
  assert.ok(env.container.firstElementChild.classList.contains("bg-layer-3"));
  await act(async () => env.container.querySelector("button").click());
  assert.deepEqual(removed, ["1_weeks;after;fromnow"]);

  await env.render(
    React.createElement(AppliedDateFilters, {
      editable: false,
      values: ["today;custom;custom"],
      options: PROJECT_CREATED_AT_FILTER_OPTIONS,
      handleRemove,
    })
  );
  assert.equal(env.container.textContent, "Today");
  assert.equal(env.container.querySelector("button"), null);

  await env.render(
    React.createElement(AppliedDateFilters, {
      editable: false,
      values: ["1_weeks;before;fromnow"],
      handleRemove,
    })
  );
  assert.equal(env.container.textContent, "1 week ago");
  assert.deepEqual(removed, ["1_weeks;after;fromnow"]);
});

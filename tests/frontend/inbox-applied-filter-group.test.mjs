import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const { InboxAppliedFilterGroup, toggleAppliedFilterValue } =
  await import("../../apps/web/core/components/inbox/inbox-filter/applied-filters/shared.tsx");

await test("applied filter groups keep stale selections clearable and preserve single-value removal", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const calls = [];

  await environment.render(
    React.createElement(InboxAppliedFilterGroup, {
      label: "Labels",
      hasFilters: true,
      options: [],
      onClearAll: () => calls.push("clear"),
    })
  );
  assert.equal(environment.container.textContent, "Labels");
  assert.equal(environment.container.querySelectorAll("button").length, 1);
  await act(() => environment.container.querySelector("button").click());
  assert.deepEqual(calls, ["clear"]);

  await environment.render(
    React.createElement(InboxAppliedFilterGroup, {
      label: "Labels",
      options: [{ key: "label-1", label: "Urgent", onRemove: () => calls.push("remove") }],
    })
  );
  assert.equal(environment.container.textContent, "LabelsUrgent");
  assert.equal(environment.container.querySelectorAll("button").length, 1);
  await act(() => environment.container.querySelector("button").click());
  assert.deepEqual(calls, ["clear", "remove"]);
});

await test("applied filter value toggles return new value arrays", () => {
  const selected = ["one", "two"];
  const removed = toggleAppliedFilterValue(selected, "one");
  const added = toggleAppliedFilterValue(selected, "three");

  assert.deepEqual(removed, ["two"]);
  assert.deepEqual(added, ["one", "two", "three"]);
  assert.deepEqual(selected, ["one", "two"]);
});

await test("shared filter selection search delegates text and clear changes", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { FilterSelectionSearch } =
    await import("../../apps/web/core/components/common/filters/filter-selection-search.tsx");
  const changes = [];

  await environment.render(
    React.createElement(FilterSelectionSearch, { value: "priority", onChange: (value) => changes.push(value) })
  );
  assert.equal(environment.container.querySelector("input")?.placeholder, "Search");
  assert.equal(environment.container.querySelector("input")?.value, "priority");
  await act(() => environment.container.querySelector("button").click());
  assert.deepEqual(changes, [""]);

  await environment.render(
    React.createElement(FilterSelectionSearch, { value: "", onChange: (value) => changes.push(value) })
  );
  assert.equal(environment.container.querySelector("button"), null);
});

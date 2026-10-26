import { filterEditorFixture } from "./helpers/filter-editor-fixture.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import {
  displayFilterConditions,
  filterConditions,
  readFilterExpression,
  updateFilterGroupOperator,
  workItemFiltersAdapter,
} from "../../packages/shared-state/dist/index.js";

registerWebModuleHooks();

function editorFor(raw, overrides = {}) {
  const snapshot = readFilterExpression(workItemFiltersAdapter, raw);
  const conditions = filterConditions(snapshot.expression);
  const displayConditions = displayFilterConditions(snapshot.expression);
  return filterEditorFixture({
    expression: snapshot.expression,
    conditions,
    displayConditions,
    configs: [],
    conversionError: snapshot.error,
    hasActiveFilters: Boolean(snapshot.error) || conditions.length > 0,
    canClear: Boolean(snapshot.error) || conditions.length > 0,
    ...overrides,
  });
}

await test("unsupported saved filters show repair state and invoke only explicit clear", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { FiltersRow } = await import("../../apps/web/core/components/rich-filters/filters-row.tsx");
  const calls = [];
  const editor = editorFor(
    { unsupported__exact: "preserved" },
    {
      onClear: async () => calls.push("clear"),
    }
  );
  await env.render(React.createElement(FiltersRow, { editor }));
  assert.match(env.container.querySelector('[role="alert"]').textContent, /could not be loaded/);
  assert.deepEqual(calls, []);
  const clear = env.container.querySelector("button");
  assert.equal(clear.textContent, "Clear filters");
  await act(async () => clear.click());
  assert.deepEqual(calls, ["clear"]);
});

await test("grouped filter UI shows NOT boundaries and delegates all/any changes", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { getPriorityFilterConfig } = await import("../../packages/utils/dist/index.js");
  const { FiltersRow } = await import("../../apps/web/core/components/rich-filters/filters-row.tsx");
  const updates = [];
  const editor = editorFor(
    { not: { or: [{ priority__in: "high" }, { priority__in: "low" }] } },
    {
      configs: [
        getPriorityFilterConfig("priority")({
          isEnabled: true,
          allowNegative: true,
          allowedOperators: new Set(["in"]),
        }),
      ],
      onUpdateGroupOperator: (groupId, operator) => {
        updates.push(updateFilterGroupOperator(editor.expression, groupId, operator));
      },
    }
  );
  await env.render(React.createElement(FiltersRow, { editor }));
  assert.ok(env.container.querySelector('fieldset[aria-label="Exclude matching filters"]'));
  const match = env.container.querySelector('select[aria-label="Match conditions"]');
  assert.equal(match.value, "or");
  assert.equal(env.container.textContent.includes("is none of"), false, "group exclusion must not invert each leaf");
  await act(async () => {
    match.value = "and";
    match.dispatchEvent(new Event("change", { bubbles: true }));
  });
  assert.deepEqual(workItemFiltersAdapter.toExternal(updates[0]), {
    not: { and: [{ priority__in: "high" }, { priority__in: "low" }] },
  });
  await env.render(React.createElement(FiltersRow, { editor, disabledAllOperations: true }));
  assert.equal(env.container.querySelector('select[aria-label="Match conditions"]').disabled, true);
});

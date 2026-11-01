import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { clickWithPointer } from "./helpers/pointer.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const options = [
  { id: "zero", value: 0, label: "Zero" },
  { id: "false", value: false, label: "False" },
  { id: "empty", value: "", label: "Empty" },
];

async function selectOption(label) {
  const option = [...document.querySelectorAll('[role="option"]')].find((node) => node.textContent === label);
  assert.ok(option, `the ${label} option is available`);
  await clickWithPointer(option);
}

async function mountSelect(t, { multiple = false, value = null, config, isDisabled = false } = {}) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { SingleSelectFilterValueInput } =
    await import("../../apps/web/core/components/rich-filters/filter-value-input/select/single.tsx");
  const { MultiSelectFilterValueInput } =
    await import("../../apps/web/core/components/rich-filters/filter-value-input/select/multi.tsx");
  const Component = multiple ? MultiSelectFilterValueInput : SingleSelectFilterValueInput;
  const changes = [];
  const props = {
    config: config ?? { getOptions: async () => options },
    condition: { id: "priority", property: "priority", operator: "in", value },
    isDisabled,
    onChange: (next) => changes.push(next),
  };
  await environment.render(React.createElement(Component, props));
  return { environment, Component, props, changes, trigger: () => environment.container.querySelector("button") };
}

await test("single-select keeps a falsy scalar when choosing an option", async (t) => {
  const { changes } = await mountSelect(t);
  await selectOption("Zero");
  assert.deepEqual(changes, [0]);
});

await test("single-select toggles the first legacy array value off when selected again", async (t) => {
  const { trigger, changes } = await mountSelect(t, { value: [false] });
  assert.equal(trigger().textContent, "False");
  await act(async () => trigger().click());
  await selectOption("False");
  assert.deepEqual(changes, [null]);
});

await test("single-select accepts an empty legacy array and an absent value", async (t) => {
  const { environment, Component, props, trigger, changes } = await mountSelect(t, { value: [] });
  await act(async () => trigger().click());
  await selectOption("Empty");
  assert.deepEqual(changes, [""]);
  await environment.render(
    React.createElement(Component, { ...props, condition: { ...props.condition, value: undefined } })
  );
  assert.equal(trigger().textContent, "--");
});

await test("multi-select retains existing falsy values when adding another option", async (t) => {
  const { changes, trigger } = await mountSelect(t, { multiple: true, value: [0, false] });
  await act(async () => trigger().click());
  await selectOption("Empty");
  assert.deepEqual(changes, [[0, false, ""]]);
});

await test("an empty multi-select opens and commits its first selection", async (t) => {
  const { changes } = await mountSelect(t, { multiple: true, value: [] });
  await selectOption("False");
  assert.deepEqual(changes, [[false]]);
});

await test("pending and failed option loads keep the selector disabled", async (t) => {
  let rejectOptions;
  const config = {
    getOptions: () =>
      new Promise((_, reject) => {
        rejectOptions = reject;
      }),
  };
  const { trigger, changes } = await mountSelect(t, { config, value: 0 });
  assert.equal(trigger().disabled, true);
  await act(async () => rejectOptions(new Error("catalog unavailable")));
  assert.equal(trigger().disabled, true);
  await act(async () => trigger().click());
  assert.deepEqual(changes, []);
  assert.equal(document.querySelector('[role="listbox"]'), null);
});

await test("a late option response cannot replace a newer configuration", async (t) => {
  let resolveOld;
  const oldConfig = {
    getOptions: () =>
      new Promise((resolve) => {
        resolveOld = resolve;
      }),
  };
  const { environment, Component, props, trigger } = await mountSelect(t, { config: oldConfig, value: 0 });
  const newConfig = { getOptions: async () => [{ id: "zero", value: 0, label: "Current zero" }] };
  await environment.render(React.createElement(Component, { ...props, config: newConfig }));
  assert.equal(trigger().disabled, false);
  assert.equal(trigger().textContent, "Current zero");
  await act(async () => resolveOld([{ id: "zero", value: 0, label: "Stale zero" }]));
  assert.equal(trigger().textContent, "Current zero");
});

await test("an explicitly disabled multi-select prevents changes after options load", async (t) => {
  const { trigger, changes } = await mountSelect(t, { multiple: true, value: [0], isDisabled: true });
  assert.equal(trigger().disabled, true);
  await act(async () => trigger().click());
  assert.deepEqual(changes, []);
  assert.equal(document.querySelector('[role="listbox"]'), null);
});

await test("adding a filter uses its first enabled operator", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { AddFilterDropdown } = await import("../../apps/web/core/components/rich-filters/add-filters/dropdown.tsx");
  const selections = [];
  const config = {
    id: "priority",
    label: "Priority",
    isEnabled: true,
    icon: () => React.createElement("span", null, "!"),
    rightContent: React.createElement("span", null, "Available"),
    supportedOperatorConfigsMap: new Map([
      ["exact", { isOperatorEnabled: false }],
      ["in", { isOperatorEnabled: true }],
    ]),
  };
  await environment.render(
    React.createElement(AddFilterDropdown, {
      customButton: React.createElement("span", null, "Add filter"),
      buttonConfig: { defaultOpen: true },
      editor: { configs: [config], conditions: [] },
      handleFilterSelect: (...selection) => selections.push(selection),
    })
  );
  const option = document.querySelector('[role="option"]');
  assert.match(option.textContent, /Priority/);
  assert.match(option.textContent, /Available/);
  await clickWithPointer(option);
  assert.deepEqual(selections, [["priority", "in", false]]);
});

await test("an already applied nonrepeatable filter cannot be added again", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { AddFilterDropdown } = await import("../../apps/web/core/components/rich-filters/add-filters/dropdown.tsx");
  const selections = [];
  await environment.render(
    React.createElement(AddFilterDropdown, {
      customButton: React.createElement("span", null, "Add filter"),
      editor: {
        configs: [{ id: "priority", label: "Priority", isEnabled: true }],
        conditions: [{ property: "priority" }],
      },
      handleFilterSelect: (...selection) => selections.push(selection),
    })
  );
  await clickWithPointer(environment.container.querySelector("button"));
  const option = document.querySelector('[role="option"]');
  assert.equal(option.textContent, "All filters applied");
  assert.equal(option.getAttribute("aria-disabled"), "true");
  await clickWithPointer(option);
  assert.deepEqual(selections, []);
});

await test("a filter without an enabled operator reports a configuration error without adding a condition", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { AddFilterDropdown } = await import("../../apps/web/core/components/rich-filters/add-filters/dropdown.tsx");
  const { Toast } = await import(webRequire.resolve("@plane/propel/toast"));
  const selections = [];
  await environment.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(Toast, { theme: "light" }),
      React.createElement(AddFilterDropdown, {
        customButton: React.createElement("span", null, "Add filter"),
        buttonConfig: { defaultOpen: true },
        editor: {
          configs: [{ id: "priority", label: "Priority", isEnabled: true, supportedOperatorConfigsMap: new Map() }],
          conditions: [],
        },
        handleFilterSelect: (...selection) => selections.push(selection),
      })
    )
  );
  await selectOption("Priority");
  assert.deepEqual(selections, []);
  assert.match(document.body.textContent, /Filter configuration error/);
  assert.match(document.body.textContent, /cannot be applied/);
});

await test("an uninitialized filter catalog cannot expose editing controls", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { FilterItem } = await import("../../apps/web/core/components/rich-filters/filter-item/root.tsx");
  await environment.render(
    React.createElement(FilterItem, {
      condition: { id: "priority", property: "priority", operator: "in", value: [0] },
      editor: { configs: [], isConfigReady: false },
    })
  );
  assert.equal(environment.container.querySelector("button, input, select"), null);
  assert.ok(environment.container.querySelector(".animate-pulse"));
  assert.doesNotMatch(environment.container.textContent, /Invalid|Unavailable/);
});

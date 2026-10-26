import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { clickWithPointer } from "./helpers/pointer.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

let ComboDropDown;
let ComboOptions;
let ComboOption;
let ComboSearchInput;

function renderTrigger({ open }) {
  return React.createElement("button", { type: "button", "aria-label": "Picker trigger" }, open ? "Open" : "Closed");
}

function LifecyclePicker() {
  const [query, setQuery] = React.useState("initial");
  const [opened, setOpened] = React.useState(0);
  const [closed, setClosed] = React.useState(0);

  return React.createElement(
    React.Fragment,
    null,
    React.createElement(
      ComboDropDown,
      {
        as: "div",
        value: "",
        onChange() {},
        renderByDefault: false,
        onOpen: () => setOpened((count) => count + 1),
        onClose: () => setClosed((count) => count + 1),
        button: renderTrigger,
      },
      ({ open, restoreButtonFocusOnClose }) =>
        open &&
        React.createElement(
          React.Fragment,
          null,
          React.createElement(ComboSearchInput, {
            query,
            onQueryChange: setQuery,
            onEscapeClose: restoreButtonFocusOnClose,
            placeholder: "Search",
          }),
          React.createElement(
            ComboOptions,
            { "aria-label": "Choices" },
            React.createElement(ComboOption, { value: "choice" }, "Choice")
          )
        )
    ),
    React.createElement("output", { "data-open-count": true }, opened),
    React.createElement("output", { "data-close-count": true }, closed)
  );
}

async function renderBasicPicker(t, { disabled = false, onChange = () => {}, value = null } = {}) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  if (!ComboDropDown) {
    const primitives = await import("../../packages/ui/src/dropdowns/combo-box.tsx");
    ComboDropDown = primitives.ComboDropDown;
    ComboOptions = primitives.ComboOptions;
    ComboOption = primitives.ComboOption;
  }
  await environment.render(
    React.createElement(
      ComboDropDown,
      {
        as: "div",
        value,
        onChange,
        disabled,
        button: renderTrigger,
      },
      ({ open }) =>
        open &&
        React.createElement(
          ComboOptions,
          { "aria-label": "Choices" },
          React.createElement(ComboOption, { value: "choice" }, "Choice")
        )
    )
  );
  return { environment, trigger: environment.container.querySelector('[aria-label="Picker trigger"]') };
}

await test("ComboDropDown keeps its native trigger mounted and restores focus only after empty Escape", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  ({ ComboDropDown, ComboOptions, ComboOption, ComboSearchInput } =
    await import("../../packages/ui/src/dropdowns/combo-box.tsx"));

  await environment.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(LifecyclePicker),
      React.createElement("button", { type: "button", "aria-label": "Outside target" }, "Outside")
    )
  );
  const trigger = environment.container.querySelector('[aria-label="Picker trigger"]');
  assert.ok(trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.querySelector('input[placeholder="Search"]'), null);

  await clickWithPointer(trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "true", "the first pointer activation opens the native picker");
  assert.equal(environment.container.querySelector('[data-open-count="true"]').textContent, "1");
  const search = document.querySelector('input[placeholder="Search"]');
  assert.ok(search);

  await React.act(async () => {
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  assert.equal(trigger.getAttribute("aria-expanded"), "true", "non-empty Escape clears search without closing");
  assert.equal(search.value, "");

  await React.act(async () => {
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  });
  assert.equal(trigger.getAttribute("aria-expanded"), "false", "empty Escape uses the native close behavior");
  assert.equal(document.querySelector('input[placeholder="Search"]'), null);
  assert.equal(document.activeElement, trigger, "empty Escape restores focus to the trigger");
  assert.equal(environment.container.querySelector('[data-close-count="true"]').textContent, "1");

  const outside = environment.container.querySelector('[aria-label="Outside target"]');
  await clickWithPointer(trigger);
  await clickWithPointer(outside);
  assert.equal(trigger.getAttribute("aria-expanded"), "false", "outside click closes the picker");
  assert.equal(document.activeElement, outside, "outside click preserves the newly focused target");
});

await test("ComboDropDown supports keyboard opening, pointer toggling, and controlled null selection", async (t) => {
  const { environment, trigger } = await renderBasicPicker(t, { value: null });
  assert.ok(trigger);

  await React.act(async () => {
    trigger.focus();
    trigger.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }));
  });
  assert.equal(trigger.getAttribute("aria-expanded"), "true", "ArrowDown opens the native picker");

  const option = document.querySelector('[role="option"]');
  assert.ok(option);
  await clickWithPointer(option);
  assert.equal(trigger.getAttribute("aria-expanded"), "false");

  await clickWithPointer(trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.equal(option.getAttribute("aria-selected"), "false", "null remains the controlled empty selection");
  await clickWithPointer(trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "false", "a second pointer activation closes the picker");
  await environment.unmount();
});

await test("ComboDropDown does not open when its native trigger is disabled", async (t) => {
  const { trigger } = await renderBasicPicker(t, { disabled: true });
  assert.ok(trigger);
  assert.equal(trigger.disabled, true);
  await clickWithPointer(trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.querySelector('[role="listbox"]'), null);
});

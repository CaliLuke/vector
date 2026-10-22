import assert from "node:assert/strict";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { React, createReactDomTestEnvironment, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
let Collapsible;
let FiltersDropdown;
async function loadComponents() {
  ({ Collapsible } = await import("../../packages/ui/src/collapsible/collapsible.tsx"));
  ({ FiltersDropdown } =
    await import("../../apps/web/core/components/issues/issue-layouts/filters/header/helpers/dropdown.tsx"));
}
function pressKey(element, key) {
  element.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
}
async function settledPanel(environment, text, expected) {
  const deadline = Date.now() + 1000;
  while (environment.container.textContent.includes(text) !== expected && Date.now() < deadline) {
    // Each transition must settle before checking whether another wait is necessary.
    // eslint-disable-next-line no-await-in-loop
    await React.act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
  }
  assert.equal(environment.container.textContent.includes(text), expected);
}
function button(environment, text) {
  return [...environment.container.querySelectorAll("button")].find((element) => element.textContent === text);
}

// Exercise the real Disclosure, Popover, Transition and filter trigger together.
// No component, event handler, or popup is replaced by a fake.
test("collapsible actions open a real filter popup without toggling the panel", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await loadComponents();
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  let selected = 0;
  function Harness() {
    const [open, setOpen] = React.useState(true);
    return React.createElement(
      Collapsible,
      {
        title: "Sub-work items",
        isOpen: open,
        onToggle: () => setOpen((value) => !value),
        titleActionElement: React.createElement(
          FiltersDropdown,
          {
            menuButton: React.createElement("span", null, "Filters"),
          },
          React.createElement("button", { type: "button", onClick: () => selected++ }, "High priority")
        ),
      },
      React.createElement("p", null, "Existing child")
    );
  }
  await environment.render(React.createElement(Harness));
  const trigger = button(environment, "Sub-work items");
  const filter = button(environment, "Filters");
  assert.ok(filter, "the separate title action must render");
  assert.equal(trigger.contains(filter), false);
  assert.equal(environment.container.querySelector("button button"), null);
  assert.equal(trigger.getAttribute("aria-expanded"), "true");

  await React.act(async () => {
    filter.focus();
    pressKey(filter, "Enter");
  });
  const choice = button(environment, "High priority");
  assert.ok(choice, "Enter opens the real filter popover");
  assert.equal(trigger.getAttribute("aria-expanded"), "true", "filter activation must not collapse its parent");
  await React.act(async () => {
    choice.focus();
    choice.click();
  });
  assert.equal(selected, 1);
  await React.act(async () => pressKey(choice, "Escape"));
  assert.equal(button(environment, "High priority"), undefined);
  assert.equal(document.activeElement, filter, "Escape returns focus to the filter trigger");

  // Native button activation is synthesized by the browser, not jsdom.
  // The real filter popup above exercises its own Enter/Escape handlers.
  await React.act(async () => {
    trigger.focus();
    trigger.click();
  });
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(button(environment, "Sub-work items"), trigger, "toggling preserves trigger identity");
  assert.equal(document.activeElement, trigger, "toggling preserves keyboard focus");
  await settledPanel(environment, "Existing child", false);
  await React.act(async () => trigger.click());
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.equal(environment.container.textContent.includes("Existing child"), true);
  assert.deepEqual(errors, []);
});

test("controlled collapsible keeps accessible state and its ref aligned after external changes", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await loadComponents();
  const ref = React.createRef();
  const view = (isOpen) =>
    React.createElement(Collapsible, { title: "Relations", isOpen, buttonRef: ref }, "Related work item");
  await environment.render(view(false));
  const trigger = ref.current;
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  await settledPanel(environment, "Related work item", false);
  await environment.render(view(true));
  assert.equal(ref.current, trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.equal(environment.container.textContent.includes("Related work item"), true);
  await environment.render(view(false));
  assert.equal(ref.current, trigger);
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  await settledPanel(environment, "Related work item", false);
});

test("collapsible preserves nested Disclosure.Panel consumers when controlled externally", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await loadComponents();
  const { Disclosure } = await import(
    new URL("headlessui.esm.js", pathToFileURL(webRequire.resolve("@headlessui/react"))).href
  );
  const view = (isOpen) =>
    React.createElement(
      Collapsible,
      { title: "Workspace members", isOpen },
      React.createElement(Disclosure.Panel, null, "Member row")
    );
  await environment.render(view(false));
  assert.equal(environment.container.textContent.includes("Member row"), false);
  await environment.render(view(true));
  assert.equal(environment.container.textContent.includes("Member row"), true);
  await environment.render(view(false));
  await settledPanel(environment, "Member row", false);
});

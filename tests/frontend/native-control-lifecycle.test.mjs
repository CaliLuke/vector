import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment, waitForReactCondition } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();

void test("disabled control links retain layout and refs without navigation; enabled links preserve modified clicks", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { ControlLink } = await import("../../packages/ui/src/control-link/control-link.tsx");
  const ref = React.createRef();
  let actions = 0;
  const props = { href: "/work-item", onClick: () => actions++, ref, className: "card" };
  await env.render(React.createElement(ControlLink, { ...props, disabled: true }, "Work item"));
  assert.equal(ref.current.tagName, "DIV");
  assert.equal(ref.current.getAttribute("aria-disabled"), "true");
  assert.equal(env.container.querySelector("a"), null);
  await React.act(async () => ref.current.click());
  assert.equal(actions, 0);
  await env.render(React.createElement(ControlLink, props, "Work item"));
  const anchor = ref.current;
  await React.act(async () => {
    for (const modifiers of [{}, { metaKey: true }, { ctrlKey: true }]) {
      const event = new MouseEvent("click", { bubbles: true, cancelable: true, ...modifiers });
      // Observe prevention before stopping jsdom's unsupported navigation.
      let prevented;
      const intercept = (e) => {
        prevented = e.defaultPrevented;
        e.preventDefault();
      };
      document.addEventListener("click", intercept, { once: true });
      anchor.dispatchEvent(event);
      assert.equal(prevented, !modifiers.metaKey && !modifiers.ctrlKey);
    }
  });
  assert.equal(actions, 1);
  await env.render(null);
  assert.equal(ref.current, null);
  await env.render(
    React.createElement(ControlLink, { href: "/disabled", disabled: true, onClick: () => actions++ }, "Plain")
  );
  assert.equal(env.container.innerHTML, "Plain");
});

void test("link cards expose separate native navigation and menu actions without submitting their parent form", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { LinkItemBlock } = await import("../../packages/ui/src/link/block.tsx");
  let selected = 0;
  let submitted = 0;
  const card = { title: "Reference", url: "https://example.com", onClick: () => selected++, menuItems: [] };
  await env.render(
    React.createElement(
      "form",
      {
        onSubmit: (event) => {
          event.preventDefault();
          submitted++;
        },
      },
      React.createElement(LinkItemBlock, card)
    )
  );
  const primary = env.container.querySelector('button[type="button"]');
  assert.ok(primary);
  await React.act(async () => primary.click());
  assert.equal(selected, 1);
  assert.equal(submitted, 0);
  const menu = env.container.querySelector('[aria-label="Link actions"]');
  assert.ok(menu);
  await React.act(async () => menu.click());
  assert.equal(selected, 1, "menu trigger does not activate the card");
  await env.render(React.createElement(LinkItemBlock, { title: card.title, url: card.url }));
  const link = env.container.querySelector("a");
  assert.equal(link.href, card.url + "/");
  assert.equal(link.rel, "noopener noreferrer");
});

void test("custom icon colors retain initial focus and move focus only after the user opens the editor", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { IconRoot } = await import("../../packages/propel/src/emoji-icon-picker/icon/icon-root.tsx");
  const before = document.createElement("button");
  document.body.append(before);
  before.focus();
  await env.render(React.createElement(IconRoot, { defaultColor: "#123456", onChange: () => {}, iconType: "lucide" }));
  assert.equal(document.activeElement, before, "a saved custom color does not steal focus on mount");
  let input = env.container.querySelector('input[aria-label="Custom icon color"]');
  assert.equal(input.value, "123456");
  const toggle = env.container.querySelector('button[aria-label="Custom icon color"]');
  await React.act(async () => toggle.click());
  assert.equal(toggle.getAttribute("aria-expanded"), "false");
  await React.act(async () => toggle.click());
  input = env.container.querySelector('input[aria-label="Custom icon color"]');
  assert.equal(document.activeElement, input);
  await React.act(async () =>
    input.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }))
  );
  assert.equal(document.activeElement, toggle);
  assert.equal(env.container.querySelector('input[aria-label="Custom icon color"]'), null);
});

void test("modal portals own focus, honor dismissal flags and retire their shared container", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { ModalPortal } = await import("../../packages/propel/src/portal/modal-portal.tsx");
  const trigger = document.createElement("button");
  document.body.append(trigger);
  trigger.focus();
  let closes = 0;
  const props = { isOpen: true, onClose: () => closes++, portalId: "native-modal", ariaLabel: "Work items" };
  const content = React.createElement("button", null, "Done");
  await env.render(
    React.createElement(ModalPortal, { ...props, closeOnEscape: false, closeOnOverlayClick: false }, content)
  );
  const dialog = document.querySelector('[role="dialog"]');
  assert.equal(dialog.getAttribute("aria-label"), "Work items");
  await waitForReactCondition(() => dialog.contains(document.activeElement), "dialog receives focus");
  await React.act(async () =>
    document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
  );
  assert.equal(closes, 0);
  await env.render(React.createElement(ModalPortal, props, content));
  await React.act(async () =>
    document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }))
  );
  assert.equal(closes, 1);
  await env.render(React.createElement(ModalPortal, { ...props, isOpen: false }, content));
  await waitForReactCondition(() => document.activeElement === trigger, "closing restores previous focus");
  assert.equal(document.getElementById("native-modal"), null);
});

void test("shared filters use one native trigger and restore focus after Escape", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const { FiltersDropdown } = await import("../../packages/ui/src/dropdowns/filters.tsx");
  await env.render(
    React.createElement(
      FiltersDropdown,
      { title: "Filters", responsive: true, isFiltersApplied: true },
      React.createElement("input", { "aria-label": "Search filters" })
    )
  );
  assert.equal(env.container.querySelectorAll("button").length, 1, "responsive labels share one keyboard stop");
  const trigger = env.container.querySelector("button");
  await React.act(async () => {
    trigger.focus();
    trigger.click();
  });
  const input = document.querySelector('input[aria-label="Search filters"]');
  assert.ok(input);
  await waitForReactCondition(() => document.activeElement === input, "opening focuses filter search");
  await React.act(async () => input.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })));
  await waitForReactCondition(() => document.activeElement === trigger, "closing restores the filter trigger");
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  await env.render(React.createElement(FiltersDropdown, { title: "Filters", disabled: true }, "Disabled contents"));
  await React.act(async () => env.container.querySelector("button").click());
  assert.equal(env.container.querySelector("button").getAttribute("aria-expanded"), "false");
});

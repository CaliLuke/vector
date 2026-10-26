import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { clickWithPointer } from "./helpers/pointer.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const modules = {
  "@plane/propel/popover": new URL("../../packages/propel/src/popover/root.tsx", import.meta.url).href,
  "@plane/propel/calendar": new URL("../../packages/propel/src/calendar/root.tsx", import.meta.url).href,
};
const stubs = {
  "@/hooks/use-current-user-profile":
    "export const useCurrentUserProfile = () => ({ data: { start_of_the_week: 1 } });",
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "./buttons": "export const DropdownButton = ({children}) => children;",
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (modules[specifier]) return nextResolve(modules[specifier], context);
    if (stubs[specifier])
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` };
    return nextResolve(specifier, context);
  },
});

async function setup(t, props = {}) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { DateDropdown } = await import("../../apps/web/core/components/dropdowns/date.tsx");
  let dismissals = 0;
  await environment.render(
    React.createElement(
      "form",
      {
        onKeyDown: (event) => {
          if (event.key === "Escape" && !event.defaultPrevented) dismissals++;
        },
      },
      React.createElement(DateDropdown, {
        buttonVariant: "border-with-text",
        value: new Date(2026, 9, 15),
        onChange: () => {},
        ...props,
      }),
      React.createElement("button", { type: "button" }, "Other field")
    )
  );
  return { environment, trigger: environment.container.querySelector("button"), dismissals: () => dismissals };
}

async function pressEscape() {
  await React.act(async () =>
    document.activeElement.dispatchEvent(
      new KeyboardEvent("keydown", {
        key: "Escape",
        bubbles: true,
        cancelable: true,
      })
    )
  );
}

await test("calendar Escape dismisses only its popup and restores its trigger", async (t) => {
  const { trigger, dismissals } = await setup(t);
  await React.act(async () => trigger.click());
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.ok(document.querySelector('[role="grid"]'));
  await pressEscape();
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.querySelector('[role="grid"]'), null);
  assert.equal(document.activeElement, trigger);
  assert.equal(dismissals(), 0);
  await pressEscape();
  assert.equal(dismissals(), 1);
});

await test("calendar defaultOpen and closeOnSelect preserve selection and closure callbacks", async (t) => {
  let selected;
  let closes = 0;
  const { trigger } = await setup(t, {
    defaultOpen: true,
    onChange: (date) => {
      selected = date;
    },
    onClose: () => closes++,
  });
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  const day = document.querySelector('[data-day="2026-10-20"] button');
  assert.ok(day);
  await React.act(async () => day.click());
  assert.equal(selected?.getDate(), 20);
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(closes, 1);
});

await test("calendar outside click preserves the newly focused field", async (t) => {
  const { environment, trigger } = await setup(t);
  await clickWithPointer(trigger);
  const outside = [...environment.container.querySelectorAll("button")].find(
    (button) => button.textContent === "Other field"
  );
  await clickWithPointer(outside);
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.activeElement, outside);
});

await test("date range Escape consumes one dismissal and retains the controlled range", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { DateRangeDropdown } = await import("../../apps/web/core/components/dropdowns/date-range.tsx");
  let dismissals = 0;
  const value = { from: new Date(2026, 9, 10), to: new Date(2026, 9, 15) };
  await environment.render(
    React.createElement(
      "form",
      {
        onKeyDown: (event) => {
          if (event.key === "Escape" && !event.defaultPrevented) dismissals++;
        },
      },
      React.createElement(DateRangeDropdown, { value, defaultOpen: true, buttonVariant: "border-with-text" })
    )
  );
  const trigger = environment.container.querySelector("button");
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  assert.ok(document.querySelector('[role="grid"]'));
  await pressEscape();
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.activeElement, trigger);
  assert.equal(dismissals, 0);
  await clickWithPointer(trigger);
  assert.ok(document.querySelector('[data-selected="true"]'), "opening again retains the selected dates");
});

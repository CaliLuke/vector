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
  let formClicks = 0;
  let dismissals = 0;
  await environment.render(
    React.createElement(
      "form",
      {
        onClick: () => formClicks++,
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
  return {
    environment,
    trigger: environment.container.querySelector("button"),
    dismissals: () => dismissals,
    formClicks: () => formClicks,
  };
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

const dateRangeScenarios = [
  {
    name: "previous month",
    today: new Date(2026, 10, 15),
    from: new Date(2026, 9, 10),
    to: new Date(2026, 9, 15),
    visibleDay: "2026-10-15",
  },
  {
    name: "previous year",
    today: new Date(2027, 0, 15),
    from: new Date(2026, 11, 10),
    to: new Date(2026, 11, 15),
    visibleDay: "2026-12-15",
  },
  {
    name: "end date only",
    today: new Date(2027, 0, 15),
    from: undefined,
    to: new Date(2026, 11, 15),
    visibleDay: "2026-12-15",
  },
  { name: "empty range", today: new Date(2027, 0, 15), from: undefined, to: undefined, visibleDay: "2027-01-15" },
];
await Promise.all(
  dateRangeScenarios.map((scenario) =>
    test(`date range Escape retains the controlled range: ${scenario.name}`, async (t) => {
      const environment = await createReactDomTestEnvironment();
      t.mock.timers.enable({ apis: ["Date"], now: scenario.today.getTime() });
      t.after(() => environment.unmount());
      const { DateRangeDropdown } = await import("../../apps/web/core/components/dropdowns/date-range.tsx");
      let dismissals = 0;
      const value = { from: scenario.from, to: scenario.to };
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
      const assertRangeVisible = () => {
        assert.ok(
          document.querySelector(`[data-day="${scenario.visibleDay}"]`),
          "opens at the controlled endpoint or current month for an empty range"
        );
        if (value.from && value.to)
          assert.ok(document.querySelector('[data-selected="true"]'), "shows the selected range");
      };
      assertRangeVisible();
      await pressEscape();
      assert.equal(trigger.getAttribute("aria-expanded"), "false");
      assert.equal(document.activeElement, trigger);
      assert.equal(dismissals, 0);
      await clickWithPointer(trigger);
      assertRangeVisible();
    })
  )
);

await test("date constraints disable out-of-range days while an open picker retains its selection contract", async (t) => {
  const changes = [];
  const { trigger } = await setup(t, {
    defaultOpen: true,
    closeOnSelect: false,
    minDate: new Date(2026, 9, 10),
    maxDate: new Date(2026, 9, 20),
    onChange: (date) => changes.push(date),
  });
  const before = document.querySelector('[data-day="2026-10-09"] button');
  const after = document.querySelector('[data-day="2026-10-21"] button');
  assert.equal(before.disabled, true);
  assert.equal(after.disabled, true);
  await React.act(async () => {
    before.click();
    after.click();
  });
  assert.deepEqual(changes, []);
  await React.act(async () => document.querySelector('[data-day="2026-10-20"] button').click());
  assert.equal(changes[0].getDate(), 20);
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
});

for (const mergeDates of [false, true])
  void test(`date range ${mergeDates ? "merged" : "separate"} labels preserve placeholders, clearing and disabled state`, async (t) => {
    const environment = await createReactDomTestEnvironment();
    t.after(() => environment.unmount());
    const { DateRangeDropdown } = await import("../../apps/web/core/components/dropdowns/date-range.tsx");
    const changes = [];
    const props = {
      value: { from: undefined, to: undefined },
      buttonVariant: "border-with-text",
      mergeDates,
      isClearable: true,
      placeholder: { from: "Start", to: "Finish" },
      clearIconClassName: "clear-date",
      onSelect: (value) => changes.push(value),
    };
    await environment.render(React.createElement(DateRangeDropdown, props));
    assert.match(environment.container.textContent, /Start/);
    assert.match(environment.container.textContent, /Finish/);
    await environment.render(React.createElement(DateRangeDropdown, { ...props, renderPlaceholder: false }));
    assert.doesNotMatch(environment.container.textContent, /Start|Finish/);
    const dates = { from: new Date(2026, 9, 10), to: new Date(2026, 9, 15) };
    await environment.render(React.createElement(DateRangeDropdown, { ...props, value: dates }));
    await React.act(async () =>
      environment.container
        .querySelector(".clear-date")
        .dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true }))
    );
    assert.deepEqual(changes, [{ from: undefined, to: undefined }]);
    assert.equal(
      environment.container.querySelector("button").getAttribute("aria-expanded"),
      "false",
      "clearing does not open the picker"
    );
    await environment.render(React.createElement(DateRangeDropdown, { ...props, value: dates, disabled: true }));
    assert.equal(environment.container.querySelector(".clear-date"), null);
    assert.equal(environment.container.querySelector("button").disabled, true);
  });

await test("clearing a date changes only its value and does not activate the enclosing picker", async (t) => {
  const changes = [];
  const { trigger, formClicks } = await setup(t, {
    isClearable: true,
    clearIconClassName: "clear-date",
    onChange: (value) => changes.push(value),
  });
  const clear = trigger.querySelector(".clear-date");
  assert.ok(clear);
  const event = new MouseEvent("click", { bubbles: true, cancelable: true });
  await React.act(async () => clear.dispatchEvent(event));
  assert.equal(event.defaultPrevented, true);
  assert.deepEqual(changes, [null]);
  assert.equal(formClicks(), 0);
  assert.equal(trigger.getAttribute("aria-expanded"), "false");
  assert.equal(document.querySelector('[role="grid"]'), null);
});

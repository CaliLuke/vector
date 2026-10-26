import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";

registerWebModuleHooks();
const nativeForms = observeNativeForms([
  "apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx",
]);
const { snoozeDate } =
  await import("../../apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/time.ts");

await test("12 AM and PM use the correct hour without mutating the selected date", () => {
  for (const [period, expectedHour] of [
    ["AM", 0],
    ["PM", 12],
  ]) {
    const date = new Date(2030, 4, 10, 8, 15, 45, 123);
    const result = snoozeDate({ date, time: "12:30", period });
    assert.equal(result.getHours(), expectedHour, period);
    assert.equal(result.getMinutes(), 30);
    assert.equal(result.getSeconds(), 0);
    assert.equal(result.getMilliseconds(), 0);
    assert.equal(date.getHours(), 8);
    assert.equal(date.getMinutes(), 15);
  }
});

globalThis.snoozeFormReact = React;
const toasts = [];
globalThis.snoozeFormToasts = toasts;
const stubs = {
  "@plane/ui": `const React = globalThis.snoozeFormReact;
    export const EModalPosition = {CENTER: "center"}; export const EModalWidth = {XXL: "xxl"};
    export const ModalCore = ({children}) => React.createElement("div", {role:"dialog"}, children);
    const SelectContext = React.createContext(null);
    export const CustomSelect = ({children, onChange, value, disabled}) => React.createElement(SelectContext.Provider, {value: {onChange, value, disabled}}, children);
    CustomSelect.Option = ({value}) => { const select = React.useContext(SelectContext); return React.createElement("button", {type: "button", disabled: select.disabled, "aria-pressed": select.value === value, onClick: () => select.onChange(value)}, value); };`,
  "@plane/propel/button": `const React = globalThis.snoozeFormReact; export const Button = ({children, loading, disabled, variant, size, ...props}) => React.createElement("button", {...props, disabled: loading || disabled}, children);`,
  "@plane/propel/toast": `export const TOAST_TYPE = {ERROR:"error"}; export const setToast = value => globalThis.snoozeFormToasts.push(value);`,
  "@/components/dropdowns/date": `const React = globalThis.snoozeFormReact; export const DateDropdown = ({onChange, disabled}) => React.createElement("button", {type:"button", disabled, onClick: () => onChange(new Date(2099, 4, 10))}, "Future date");`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

await test("snooze validates the date, preserves the selection on failure, and resets when reopened", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { NotificationSnoozeModal } =
    await import("../../apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx");
  let fail = true;
  const savedDates = [];
  let closed = 0;
  const props = {
    isOpen: true,
    onClose: () => closed++,
    onSubmit: async (date) => {
      savedDates.push(date);
      if (fail) throw new Error("offline");
    },
  };
  const render = (isOpen) => environment.render(React.createElement(NotificationSnoozeModal, { ...props, isOpen }));
  const button = (label) =>
    [...environment.container.querySelectorAll("button")].find((item) => item.textContent === label);
  await render(true);
  await act(async () => button("Submit").click());
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /future/);
  assert.equal(savedDates.length, 0);
  await act(async () => button("Future date").click());
  await act(async () => button("PM").click());
  await act(async () => button("12:00").click());
  await act(async () => button("Submit").click());
  assert.equal(closed, 0);
  assert.equal(
    nativeForms
      .getForms("apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx")
      .at(-1).state.isSubmitSuccessful,
    false
  );
  assert.equal(toasts.at(-1).type, "error");
  assert.equal(button("12:00").getAttribute("aria-pressed"), "true");
  fail = false;
  await act(async () => button("Submit").click());
  assert.equal(closed, 1);
  assert.equal(
    nativeForms
      .getForms("apps/web/core/components/workspace-notifications/sidebar/notification-card/options/snooze/modal.tsx")
      .at(-1).state.isSubmitSuccessful,
    true
  );
  assert.equal(savedDates[1].getHours(), 12);
  await render(false);
  await render(true);
  await act(async () => button("Submit").click());
  assert.equal(savedDates.length, 2);
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /future/);
});

nativeForms.restore();

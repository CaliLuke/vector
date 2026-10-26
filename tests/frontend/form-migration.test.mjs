import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
globalThis.migrationFormReact = React;
const themeSelections = [];
globalThis.migrationThemeSelections = themeSelections;
const formToasts = [];
globalThis.migrationFormToasts = formToasts;
const stubs = {
  "@/hooks/use-current-user-profile": `export const useCurrentUserProfile = () => ({ data: globalThis.migrationProfile.data }); export const useUpdateUserTheme = () => ({ mutateAsync: globalThis.migrationProfile.updateUserTheme });`,
  "@/components/settings/profile/heading": `const React = globalThis.migrationFormReact; export const ProfileSettingsHeading = ({control}) => React.createElement("div", null, control);`,
  "next-themes": `export const useTheme = () => ({setTheme: value => globalThis.migrationThemeSelections.push(value)});`,
  "@plane/i18n": `export const useTranslation = () => ({t: key => key});`,
  "@plane/propel/toast": `export const TOAST_TYPE = {ERROR:"error", SUCCESS:"success"}; export const setToast = value => globalThis.migrationFormToasts.push(value);`,
  "@plane/propel/button": `const React = globalThis.migrationFormReact; export const Button = ({children, loading, disabled, variant, size, ...props}) => React.createElement("button", {...props, disabled: loading || disabled}, children);`,
  "@plane/ui": `const React = globalThis.migrationFormReact;
    export const EModalPosition = {CENTER: "center"}; export const EModalWidth = {XXL: "xxl"};
    export const ModalCore = ({children}) => React.createElement("div", {role:"dialog"}, children);
    export const InputColorPicker = ({hasError, onChange, ...props}) => React.createElement("input", {...props, onChange:event => onChange(event.target.value)});
    export const Input = ({hasError, ...props}) => React.createElement("input", props);`,
  "@plane/propel/calendar": `const React = globalThis.migrationFormReact; export const Calendar = ({selected, onSelect}) => React.createElement("input", {"aria-label":"Calendar date", type:"date", value:selected.toISOString().slice(0,10), onChange:event => onSelect(new Date(event.target.value + "T12:00:00"))});`,
  "./date-filter-select": `const React = globalThis.migrationFormReact; export const DateFilterSelect = ({value, onChange}) => React.createElement("select", {value, onChange: event => onChange(event.target.value)}, ["before","after","range"].map(value => React.createElement("option", {key:value, value}, value)));`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

async function change(input, value) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
const submit = (container) =>
  act(async () =>
    container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );

await test("automation months reject invalid numbers and retain the entered month after a failed save", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { SelectMonthModal } = await import("../../apps/web/core/components/automation/select-month-modal.tsx");
  const saved = [];
  let fail = true,
    closed = 0;
  await environment.render(
    React.createElement(SelectMonthModal, {
      isOpen: true,
      type: "auto-close",
      initialValues: { close_in: 3 },
      handleClose: () => closed++,
      handleChange: async (value) => {
        saved.push(value);
        if (fail) throw new Error("offline");
      },
    })
  );
  const input = environment.container.querySelector("input");
  await change(input, "13");
  await submit(environment.container);
  assert.deepEqual(saved, []);
  await change(input, "6");
  await submit(environment.container);
  assert.equal(closed, 0);
  assert.equal(input.value, "6");
  fail = false;
  await submit(environment.container);
  assert.deepEqual(saved, [{ close_in: 6 }, { close_in: 6 }]);
  assert.equal(closed, 1);
});

await test("link edits survive a refetch and failed save, then reopen with the latest server value", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { LinkModal } = await import("../../apps/web/core/components/common/link-modal.tsx");
  const saved = [];
  let closed = 0,
    fail = true;
  const props = {
    isOpen: true,
    link: { id: "link", title: "Original", url: "https://example.com" },
    onClose: () => closed++,
    onSave: async (value) => {
      saved.push(value);
      if (fail) throw new Error("offline");
    },
  };
  await environment.render(React.createElement(LinkModal, props));
  await change(environment.container.querySelector('[name="title"]'), "Draft");
  const refreshed = { ...props, link: { ...props.link, title: "Server" } };
  await environment.render(React.createElement(LinkModal, refreshed));
  await submit(environment.container);
  assert.equal(closed, 0);
  assert.equal(environment.container.querySelector('[name="title"]').value, "Draft");
  fail = false;
  await submit(environment.container);
  assert.equal(saved[1].title, "Draft");
  await environment.render(React.createElement(LinkModal, { ...refreshed, isOpen: false }));
  await environment.render(React.createElement(LinkModal, refreshed));
  assert.equal(environment.container.querySelector('[name="title"]').value, "Server");
});

await test("date ranges cannot apply an end date before the start date", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { DateFilterModal } = await import("../../apps/web/core/components/core/filters/date-filter-modal.tsx");
  const selections = [];
  await environment.render(
    React.createElement(DateFilterModal, {
      isOpen: true,
      title: "Created",
      handleClose() {},
      onSelect: (value) => selections.push(value),
    })
  );
  const dates = environment.container.querySelectorAll('input[type="date"]');
  await change(dates[0], "2030-05-10");
  await change(dates[1], "2030-05-01");
  await submit(environment.container);
  assert.deepEqual(selections, []);
  await change(dates[1], "2030-05-20");
  await submit(environment.container);
  assert.deepEqual(selections, [["2030-05-10;after", "2030-05-20;before"]]);
});

await test("invitation arrays remove the selected row, exclude optional blanks, and retain recipients on failure", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { useInvitationForm } = await import("../../apps/web/core/hooks/use-workspace-invitation.tsx");
  let fail = true;
  const saved = [];
  const failures = [];
  function Harness() {
    const form = useInvitationForm({
      initialRows: 3,
      allowEmptyRows: true,
      onSubmit: async (value) => {
        saved.push(value);
        if (fail) throw new Error("offline");
      },
    });
    return React.createElement(
      "form",
      {
        onSubmit: (event) => {
          event.preventDefault();
          void form.handleSubmit().catch((error) => failures.push(error));
        },
      },
      React.createElement(form.Field, { name: "emails", mode: "array" }, (field) =>
        field.state.value.map((entry, index) =>
          React.createElement(form.Field, { key: entry.id, name: `emails[${index}].email` }, (email) =>
            React.createElement("input", {
              value: email.state.value,
              onChange: (event) => email.handleChange(event.target.value),
            })
          )
        )
      ),
      React.createElement(
        "button",
        { type: "button", onClick: () => form.removeFieldValue("emails", 0) },
        "Remove first"
      )
    );
  }
  await environment.render(React.createElement(Harness));
  const inputs = environment.container.querySelectorAll("input");
  await change(inputs[0], "remove@example.com");
  await change(inputs[1], "keep@example.com");
  await act(async () => environment.container.querySelector("button").click());
  await submit(environment.container);
  assert.equal(environment.container.querySelector("input").value, "keep@example.com");
  assert.equal(failures.length, 1);
  assert.match(failures[0].message, /offline/);
  fail = false;
  await submit(environment.container);
  assert.deepEqual(saved[1], { emails: [{ email: "keep@example.com", role: 15 }] });
  assert.equal(environment.container.querySelector("input").value, "");
  assert.equal(failures.length, 1);
});

await test("theme imports reject malformed values instead of treating them as a saved configuration", async () => {
  const { parseThemeConfig } = await import("../../apps/web/core/components/core/theme/theme-config.ts");
  assert.throws(() => parseThemeConfig("null"), /configuration object/);
  assert.throws(() => parseThemeConfig('{"primary":"red","background":"#fff"}'), /brand color/);
  assert.throws(() => parseThemeConfig('{"primary":"#abc","background":"#fff","darkPalette":"true"}'), /boolean/);
  assert.deepEqual(parseThemeConfig('{"primary":"#abc","background":"#fff"}'), {
    theme: "custom",
    primary: "#abc",
    background: "#fff",
    darkPalette: false,
  });
});

await test("a failed theme import retains the palette without reporting success and retries without a reload", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const saved = [];
  let fail = true;
  globalThis.migrationProfile = {
    data: { theme: { theme: "custom", primary: "#123456", background: "#ffffff", darkPalette: false } },
    updateUserTheme: async (value) => {
      saved.push(value);
      if (fail) throw new Error("offline");
      globalThis.migrationProfile.data.theme = value;
    },
  };
  const { CustomThemeSelector } = await import("../../apps/web/core/components/core/theme/custom-theme-selector.tsx");
  await environment.render(React.createElement(CustomThemeSelector));
  const file = environment.container.querySelector('input[type="file"]');
  Object.defineProperty(file, "files", {
    value: [{ text: async () => JSON.stringify({ primary: "#abcdef", background: "#333333", darkPalette: true }) }],
  });
  await act(async () => file.dispatchEvent(new Event("change", { bubbles: true })));
  assert.equal(formToasts.at(-1).type, "error");
  assert.deepEqual(themeSelections, []);
  assert.equal(environment.container.querySelector('input[name="primary"]').value, "#abcdef");
  fail = false;
  await submit(environment.container);
  assert.equal(formToasts.at(-1).type, "success");
  assert.deepEqual(themeSelections, ["custom"]);
  assert.deepEqual(saved[1], { theme: "custom", primary: "#abcdef", background: "#333333", darkPalette: true });
});

await test("confirmation requires both exact values, preserves failed attempts, and locks dismissal during a request", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { ConfirmActionModal } = await import("../../apps/web/core/components/common/confirm-action-modal.tsx");
  let finish,
    calls = 0,
    closed = 0,
    confirmed = 0;
  const props = {
    isOpen: true,
    title: "Delete project",
    name: "Project",
    confirmation: "delete my project",
    actionLabel: "Delete",
    pendingLabel: "Deleting",
    errorMessage: "Unable to delete",
    onClose: () => closed++,
    onConfirmed: () => confirmed++,
    onConfirm: () => {
      calls++;
      return new Promise((resolve, reject) => {
        finish = { resolve, reject };
      });
    },
  };
  await environment.render(React.createElement(ConfirmActionModal, props));
  await change(environment.container.querySelector('[name="name"]'), "Project");
  await submit(environment.container);
  assert.equal(calls, 0);
  await change(environment.container.querySelector('[name="confirmation"]'), "delete my project");
  await submit(environment.container);
  assert.equal(calls, 1);
  assert.ok([...environment.container.querySelectorAll("button")].every((button) => button.disabled));
  await act(async () => finish.reject(new Error("offline")));
  assert.equal(closed, 0);
  assert.equal(confirmed, 0);
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Unable to delete/);
  assert.equal(environment.container.querySelector('[name="name"]').value, "Project");
  await submit(environment.container);
  await act(async () => finish.resolve());
  assert.equal(closed, 1);
  assert.equal(confirmed, 1);
  await environment.render(React.createElement(ConfirmActionModal, { ...props, isOpen: false }));
  await environment.render(React.createElement(ConfirmActionModal, props));
  assert.equal(environment.container.querySelector('[name="name"]').value, "");
});

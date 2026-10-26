import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createModuleStub } from "./helpers/module-stubs.mjs";
import { changeFormControl, submitForm } from "./helpers/native-forms.mjs";

registerWebModuleHooks();
globalThis.apiTokenFormReact = React;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (context.parentURL?.endsWith("/form.tsx") && specifier === "@/components/dropdowns/date")
      return createModuleStub(`const React = globalThis.apiTokenFormReact;
        export const DateDropdown = () => React.createElement("button", { type: "button" }, "Date");`);
    if (specifier === "@plane/i18n")
      return createModuleStub(`export const useTranslation = () => ({ t: (key) => key });`);
    if (specifier === "@plane/propel/button")
      return createModuleStub(`const React = globalThis.apiTokenFormReact;
        export const Button = ({ loading, ...props }) => React.createElement("button", props, props.children);`);
    if (specifier === "@plane/ui")
      return createModuleStub(`const React = globalThis.apiTokenFormReact;
        export const Input = ({ hasError, ...props }) => React.createElement("input", props);
        export const TextArea = (props) => React.createElement("textarea", props);
        export const ToggleSwitch = ({ value, onChange, ...props }) => React.createElement("button", {
          ...props, type: "button", role: "switch", "aria-checked": value, onClick: onChange
        });
        const Select = ({ customButton, value, onChange, children, ...props }) => React.createElement("select", {
          ...props, value, onChange: (event) => onChange(event.target.value)
        }, React.createElement("option", { value: "" }, "Set expiration date"), children);
        Select.Option = ({ value, children }) => React.createElement("option", { value }, children);
        export const CustomSelect = Select;`);
    if (specifier === "@plane/utils/common")
      return createModuleStub(`export const cn = (...values) => values.filter(Boolean).join(" ");`);
    if (specifier === "@plane/utils/datetime")
      return createModuleStub(`export const renderFormattedDate = () => "date";
        export const renderFormattedTime = () => "time";`);
    return nextResolve(specifier, context);
  },
});

await test("API token form validates expiration and preserves its draft after a failed request", async (t) => {
  const env = await createReactDomTestEnvironment();
  const submissions = [];
  let shouldFail = true;
  t.after(() => env.unmount());
  const { CreateApiTokenForm } = await import("../../apps/web/core/components/api-token/modal/form.tsx");
  await env.render(
    React.createElement(CreateApiTokenForm, {
      handleClose: () => {},
      onSubmit: async (data) => {
        submissions.push(data);
        if (shouldFail) throw new Error("offline");
        shouldFail = false;
      },
    })
  );

  await changeFormControl(env.container.querySelector('input[name="label"]'), "Personal token");
  await submitForm(env.container);
  await act(async () => new Promise((resolve) => setImmediate(resolve)));
  assert.equal(submissions.length, 0);
  assert.match(env.container.querySelector('[role="alert"]').textContent, /future expiration/);

  await act(async () => env.container.querySelector('[role="switch"]').click());
  await submitForm(env.container);
  await act(async () => new Promise((resolve) => setImmediate(resolve)));
  assert.equal(submissions.length, 1);
  assert.equal(submissions[0].label, "Personal token");
  assert.equal(submissions[0].expired_at, null);
  assert.equal(env.container.querySelector('input[name="label"]').value, "Personal token");
  assert.equal(env.container.querySelector('[role="alert"]').textContent, "offline");

  shouldFail = false;
  await submitForm(env.container);
  await act(async () => new Promise((resolve) => setImmediate(resolve)));
  assert.equal(submissions.length, 2);
  assert.equal(submissions[1].label, "Personal token");
});

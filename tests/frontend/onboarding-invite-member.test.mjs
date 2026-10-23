import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createReactDomTestEnvironment, webRequire } from "./helpers/react-dom.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const packageStubs = {
  "@plane/constants": `export const ROLE = { 15: "Admin", 20: "Member" };
    export const ROLE_DETAILS = {
      15: { i18n_title: "role.admin", i18n_description: "role.admin_description" },
      20: { i18n_title: "role.member", i18n_description: "role.member_description" },
    };`,
  "@plane/i18n": `export const useTranslation = () => ({ t: key => key });`,
  "@plane/propel/icons": `export const CheckIcon = () => null; export const ChevronDownIcon = () => null;`,
  "@plane/ui": `export const Input = globalThis.React.forwardRef((props, ref) => {
    const inputProps = { ...props };
    delete inputProps.hasError;
    return globalThis.React.createElement("input", { ...inputProps, ref });
  });`,
  "mobx-react": `export const observer = component => component;`,
  "react-popper": `export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });`,
};
globalThis.React = React;

registerHooks({
  resolve(specifier, context, nextResolve) {
    return resolvePackageStub(specifier, packageStubs, context, nextResolve);
  },
});

const { useFieldArray, useForm } = webRequire("react-hook-form");

function setInputValue(input, value) {
  Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
  input.dispatchEvent(new Event("input", { bubbles: true }));
}

await test("shared onboarding invite row edits email, selects a role, and removes a row", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { InviteMemberInput } = await import("../../apps/web/core/components/onboarding/invite-member-input.tsx");

  function Harness() {
    const [isInvitationDisabled, setIsInvitationDisabled] = React.useState(true);
    const { control, getValues, setValue, watch } = useForm({
      defaultValues: {
        emails: [
          { email: "", role: 15, role_active: false },
          { email: "", role: 15, role_active: false },
        ],
      },
    });
    const { fields, remove } = useFieldArray({ control, name: "emails" });

    return React.createElement(
      React.Fragment,
      null,
      React.createElement(
        "output",
        { "data-testid": "invitation-state" },
        isInvitationDisabled ? "disabled" : "enabled"
      ),
      React.createElement("output", { "data-testid": "first-role-active" }, String(watch("emails.0.role_active"))),
      fields.map((field, index) =>
        React.createElement(InviteMemberInput, {
          key: field.id,
          control,
          emailColumnClassName: "col-span-6",
          errors: {},
          fields,
          getValues,
          index,
          isInvitationDisabled,
          remove,
          setIsInvitationDisabled,
          setValue,
          watch,
        })
      )
    );
  }

  await environment.render(React.createElement(Harness));
  const emailInputs = () => environment.container.querySelectorAll('input[type="text"]');
  assert.equal(emailInputs().length, 2);

  await React.act(async () => setInputValue(emailInputs()[0], "person@example.com"));
  assert.equal(environment.container.querySelector('[data-testid="invitation-state"]').textContent, "enabled");
  assert.equal(environment.container.querySelector('[data-testid="first-role-active"]').textContent, "true");

  await React.act(async () => environment.container.querySelector(".group button").click());
  const memberOption = [...document.querySelectorAll('[role="option"]')].find((option) =>
    option.textContent.includes("role.member")
  );
  assert.ok(memberOption, "the real Headless UI listbox should expose the member role");
  await React.act(async () => memberOption.click());
  assert.equal(environment.container.querySelector(".group button").textContent.trim(), "Member");

  const rowRemoveButtons = environment.container.querySelectorAll(".group button");
  await React.act(async () => rowRemoveButtons[rowRemoveButtons.length - 1].click());
  assert.equal(emailInputs().length, 1, "removing a row should retain the other invitation");
});

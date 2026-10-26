import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const packageStubs = {
  "@plane/i18n": `export const useTranslation = () => ({ t: key => key });`,
  "@plane/propel/icons": `export const CheckIcon = () => null; export const ChevronDownIcon = () => null;`,
  "@plane/ui": `export const Input = globalThis.React.forwardRef((props, ref) => {
    const inputProps = { ...props };
    delete inputProps.hasError;
    return globalThis.React.createElement("input", { ...inputProps, ref });
  });`,
  "react-popper": `export const usePopper = () => ({ styles: { popper: {} }, attributes: { popper: {} } });`,
};
globalThis.React = React;

registerHooks({
  resolve(specifier, context, nextResolve) {
    return resolvePackageStub(specifier, packageStubs, context, nextResolve);
  },
});

function setInputValue(input, value) {
  Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
  input.dispatchEvent(new Event("input", { bubbles: true }));
}

await test("shared onboarding invite row edits email, selects a role, and removes a row", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { InviteMemberFields } = await import("../../apps/web/core/components/onboarding/invite-member-input.tsx");

  const { useInvitationForm } = await import("../../apps/web/core/hooks/use-workspace-invitation.tsx");
  function Harness() {
    const form = useInvitationForm({ initialRows: 2, allowEmptyRows: true, onSubmit: async () => {} });
    return React.createElement(
      React.Fragment,
      null,
      React.createElement(
        form.Subscribe,
        { selector: (state) => state.canSubmit && state.values.emails.some((entry) => entry.email.trim()) },
        (ready) => React.createElement("output", { "data-testid": "invitation-state" }, ready ? "enabled" : "disabled")
      ),
      React.createElement(InviteMemberFields, { form, emailColumnClassName: "col-span-6" })
    );
  }

  await environment.render(React.createElement(Harness));
  const emailInputs = () => environment.container.querySelectorAll('input[type="email"]');
  assert.equal(emailInputs().length, 2);

  await React.act(async () => setInputValue(emailInputs()[0], "person@example.com"));
  assert.equal(environment.container.querySelector('[data-testid="invitation-state"]').textContent, "enabled");

  await React.act(async () => environment.container.querySelector(".group button").click());
  const adminOption = [...document.querySelectorAll('[role="option"]')].find((option) =>
    option.textContent.includes("role_details.admin.title")
  );
  assert.ok(adminOption, "the real Headless UI listbox should expose the admin role");
  await React.act(async () => adminOption.click());
  assert.equal(environment.container.querySelector(".group button").textContent.trim(), "Admin");

  const rowRemoveButtons = environment.container.querySelectorAll(".group button");
  await React.act(async () => rowRemoveButtons[rowRemoveButtons.length - 1].click());
  assert.equal(emailInputs().length, 1, "removing a row should retain the other invitation");
});

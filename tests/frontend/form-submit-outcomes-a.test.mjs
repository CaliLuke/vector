import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();

const emailSource = "packages/ui/src/auth-form/email-entry-form.tsx";
const observer = observeNativeForms([emailSource]);
const nativeEmailForm = () => observer.getForms(emailSource).at(-1);

const settle = () => act(async () => new Promise((resolve) => setTimeout(resolve, 20)));

await test("email continuation failure reports the error and leaves native submission unsuccessful", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  let fail = true;
  const { EmailEntryForm } = await import("../../packages/ui/src/auth-form/email-entry-form.tsx");
  await environment.render(
    React.createElement(EmailEntryForm, {
      defaultEmail: "reader@example.com",
      onSubmit: async () => {
        if (fail) throw new Error("offline");
      },
    })
  );
  await act(async () => environment.container.querySelector('[type="submit"]').click());
  await settle();
  assert.equal(nativeEmailForm()?.state.isSubmitSuccessful, false);
  assert.equal(environment.container.querySelector('[type="submit"]').disabled, false);
  assert.match(environment.container.querySelector('[role="alert"]')?.textContent ?? "", /Could not continue/);
  assert.equal(environment.container.querySelector("input").value, "reader@example.com");

  fail = false;
  await act(async () => environment.container.querySelector('[type="submit"]').click());
  await settle();
  assert.equal(nativeEmailForm()?.state.isSubmitSuccessful, true);
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
});

import { resolvePackageStub } from "./helpers/module-stubs.mjs";
const emailChangeSource = "apps/web/core/components/core/modals/change-email-modal.tsx";
const changeEmailObserver = observeNativeForms([emailChangeSource]);
globalThis.changeEmailOutcomeAReact = React;
const changeEmailState = { checks: 0, generated: 0, existing: true, checkFailure: true };
globalThis.changeEmailOutcomeA = changeEmailState;
const changeEmailStubs = {
  "@/hooks/use-current-user": `export const useSignOut = () => ({mutateAsync: async () => {}});`,
  "@/services/auth.service": `export const authService = { requestCSRFToken: async () => ({ csrf_token: "csrf" }) };`,
  "@/services/user.service": `export default { checkEmail: async () => { globalThis.changeEmailOutcomeA.checks++; if (globalThis.changeEmailOutcomeA.checkFailure) throw new Error("offline"); return { existing: globalThis.changeEmailOutcomeA.existing }; }, generateEmailCode: async () => { globalThis.changeEmailOutcomeA.generated++; }, verifyEmailCode: async () => {} };`,
  "@plane/i18n": `export const useTranslation = () => ({ t: key => key });`,
  "@plane/propel/toast": `export const TOAST_TYPE = {SUCCESS:"success",ERROR:"error"}; export const setToast = () => {};`,
  "@plane/propel/button": `const React = globalThis.changeEmailOutcomeAReact; export const Button = props => React.createElement("button", props, props.children);`,
  "@plane/ui": `const React = globalThis.changeEmailOutcomeAReact; export const EModalPosition={CENTER:"center"}, EModalWidth={XXL:"xxl"}; export const ModalCore=({children})=>React.createElement("div", null, children); export const Input=props=>React.createElement("input", props);`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, changeEmailStubs, context, nextResolve),
});

await test("change-email availability validation retains the draft and allows retry", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { ChangeEmailModal } = await import("../../apps/web/core/components/core/modals/change-email-modal.tsx");
  await environment.render(React.createElement(ChangeEmailModal, { isOpen: true, onClose: () => {} }));
  const input = environment.container.querySelector('[name="email"]');
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, "new@example.com");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () => environment.container.querySelector('[type="submit"]').click());
  await settle();
  assert.equal(changeEmailState.checks, 1);
  assert.equal(changeEmailState.generated, 0);
  assert.equal(changeEmailObserver.getForms(emailChangeSource).at(-1)?.state.isSubmitSuccessful, false);
  assert.match(environment.container.querySelector('[role="alert"]')?.textContent ?? "", /offline/);
  assert.equal(environment.container.querySelector('[type="submit"]').disabled, false);
  assert.equal(environment.container.querySelector('[name="email"]').value, "new@example.com");
  changeEmailState.checkFailure = false;
  await act(async () => environment.container.querySelector('[type="submit"]').click());
  await settle();
  assert.equal(changeEmailState.checks, 2);
  assert.equal(changeEmailState.generated, 0);
  assert.equal(changeEmailObserver.getForms(emailChangeSource).at(-1)?.state.isSubmitSuccessful, false);
  assert.match(environment.container.querySelector('[role="alert"]')?.textContent ?? "", /exists/);
  assert.equal(environment.container.querySelector('[type="submit"]').disabled, false);
  changeEmailState.existing = false;
  await act(async () => environment.container.querySelector('[type="submit"]').click());
  await settle();
  assert.equal(changeEmailState.checks, 3);
  assert.equal(changeEmailState.generated, 1);
  assert.ok(environment.container.querySelector('[name="code"]'));
  assert.equal(changeEmailObserver.getForms(emailChangeSource).at(-1)?.state.isSubmitSuccessful, true);
});

observer.restore();
changeEmailObserver.restore();

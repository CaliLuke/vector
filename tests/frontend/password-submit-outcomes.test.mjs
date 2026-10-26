import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { setupWebNativeFormTest } from "./helpers/web-native-form-test.mjs";

const hookSource = "apps/web/core/components/account/auth-forms/common/new-password-fields.tsx";
const formObserver = setupWebNativeFormTest({
  react: React,
  sourceFiles: [hookSource],
  globalName: "passwordOutcomeReact",
});
const state = { resetCalls: 0, setCalls: 0, authErrors: 0, navigations: 0, fail: true, failNavigation: false };
globalThis.passwordOutcomeState = state;

const stubs = {
  "@/helpers/authentication.helper": `export const showAuthenticationError=()=>{globalThis.passwordOutcomeState.authErrors++};export const EErrorAlertType={BANNER_ALERT:"banner"};export const authErrorHandler=()=>undefined;`,
  "@/services/auth.service": `export const authService={submitForm:async()=>{globalThis.passwordOutcomeState.resetCalls++;if(globalThis.passwordOutcomeState.fail)throw new Error("offline")},requestCSRFToken:async()=>({csrf_token:"csrf"})};`,
  "@/hooks/use-current-user": `export const useCurrentUser=()=>({data:{email:"reader@example.com"}});export const usePasswordMutation=()=>({mutateAsync:async()=>{globalThis.passwordOutcomeState.setCalls++;if(globalThis.passwordOutcomeState.fail)throw new Error("offline")},reset:()=>{globalThis.passwordOutcomeState.resetCalls++}});`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/button": `const React=globalThis.passwordOutcomeReact;export const Button=props=>React.createElement("button",props,props.children);`,
  "@plane/ui": `const React=globalThis.passwordOutcomeReact;const Field=({label,...props})=>React.createElement("label",null,label,React.createElement("input",props));export const Input=props=>React.createElement("input",props);export const AuthPasswordInput=Field;export const AuthConfirmPasswordInput=({label,...props})=>React.createElement("label",null,label,React.createElement("input",props));`,
  "@plane/utils/auth":
    "export const isPasswordConfirmationValid=(password,confirmation)=>password.length>=8&&password===confirmation;",
  "@tanstack/react-router": `export const useLocation=()=>({searchStr:"?uidb64=encoded&token=token&email=reader%40example.com"});export const useRouter=()=>({navigate:async()=>{globalThis.passwordOutcomeState.navigations++;if(globalThis.passwordOutcomeState.failNavigation){globalThis.passwordOutcomeState.failNavigation=false;throw new Error("navigation failed")}}});`,
  "./common/container": `const React=globalThis.passwordOutcomeReact;export const FormContainer=({children})=>React.createElement("main",null,children);`,
  "./common/header": `const React=globalThis.passwordOutcomeReact;export const AuthFormHeader=({title})=>React.createElement("h1",null,title);`,
  "./auth-banner": `const React=globalThis.passwordOutcomeReact;export const AuthBanner=()=>React.createElement("aside",null,"Authentication notice");`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

const formApi = () => formObserver.getForms(hookSource).at(-1);
async function submitAndWaitForForm(environment) {
  const form = formApi();
  assert.ok(form, "the custom hook returned a real native FormApi");
  await act(async () => {
    await new Promise((resolve) => {
      let sawSubmitting = form.state.isSubmitting;
      let subscription;
      subscription = form.store.subscribe(() => {
        if (form.state.isSubmitting) sawSubmitting = true;
        else if (sawSubmitting) {
          subscription.unsubscribe();
          resolve();
        }
      });
      environment.container.querySelector('[type="submit"]').click();
    });
  });
}

await test("reset password failure remains a native failure and retries the preserved draft", async (t) => {
  state.resetCalls = 0;
  state.authErrors = 0;
  state.fail = true;
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { ResetPasswordForm } = await import("../../apps/web/core/components/account/auth-forms/reset-password.tsx");
  await environment.render(React.createElement(ResetPasswordForm));
  await changeFormControl(environment.container.querySelector('[name="password"]'), "correct-horse");
  await changeFormControl(environment.container.querySelector('[name="confirm_password"]'), "correct-horse");
  await submitAndWaitForForm(environment);
  assert.equal(state.resetCalls, 1);
  assert.equal(state.authErrors, 1);
  assert.equal(formApi()?.state.isSubmitSuccessful, false);
  assert.equal(environment.container.querySelector('[name="password"]').value, "correct-horse");
  assert.equal(environment.container.querySelector('[type="submit"]').disabled, false);
  state.fail = false;
  await submitAndWaitForForm(environment);
  assert.equal(state.resetCalls, 2);
  assert.equal(formApi()?.state.isSubmitSuccessful, true);
});

await test("set password failure remains a native failure and retries the preserved draft", async (t) => {
  state.setCalls = 0;
  state.authErrors = 0;
  state.navigations = 0;
  state.fail = true;
  state.failNavigation = false;
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { SetPasswordForm } = await import("../../apps/web/core/components/account/auth-forms/set-password.tsx");
  await environment.render(React.createElement(SetPasswordForm));
  await changeFormControl(environment.container.querySelector('[name="password"]'), "correct-horse");
  await changeFormControl(environment.container.querySelector('[name="confirm_password"]'), "correct-horse");
  await submitAndWaitForForm(environment);
  assert.equal(state.setCalls, 1);
  assert.equal(state.authErrors, 1);
  assert.equal(formApi()?.state.isSubmitSuccessful, false);
  assert.equal(environment.container.querySelector('[name="password"]').value, "correct-horse");
  assert.equal(environment.container.querySelector('[type="submit"]').disabled, false);
  state.fail = false;
  await submitAndWaitForForm(environment);
  assert.equal(state.setCalls, 2);
  assert.equal(state.navigations, 1);
  assert.equal(formApi()?.state.isSubmitSuccessful, true);
});

await test("set password navigation retry does not repeat the saved credential", async (t) => {
  state.setCalls = 0;
  state.authErrors = 0;
  state.navigations = 0;
  state.fail = false;
  state.failNavigation = true;
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { SetPasswordForm } = await import("../../apps/web/core/components/account/auth-forms/set-password.tsx");
  await environment.render(React.createElement(SetPasswordForm));
  await changeFormControl(environment.container.querySelector('[name="password"]'), "correct-horse");
  await changeFormControl(environment.container.querySelector('[name="confirm_password"]'), "correct-horse");
  await submitAndWaitForForm(environment);
  assert.equal(state.setCalls, 1);
  assert.equal(state.navigations, 1);
  assert.equal(formApi()?.state.isSubmitSuccessful, false);
  assert.equal(environment.container.querySelector('[name="password"]').readOnly, true);
  assert.equal(environment.container.querySelector('[type="submit"]').disabled, false);
  await submitAndWaitForForm(environment);
  assert.equal(state.setCalls, 1);
  assert.equal(state.navigations, 2);
  assert.equal(formApi()?.state.isSubmitSuccessful, true);
});

formObserver.restore();

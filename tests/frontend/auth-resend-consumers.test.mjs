import assert from "node:assert/strict";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { changeFormControl } from "./helpers/native-forms.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

globalThis.authResendConsumerReact = React;
registerWebTestModules({
  "@plane/constants": "export const API_BASE_URL='http://localhost';",
  "@plane/i18n":
    "export const useTranslation=()=>({t:(key,values)=>values?.seconds?`Resend in ${values.seconds}s`:key});",
  "@plane/hooks": `export {useResendCodeForm} from ${JSON.stringify(new URL("../../packages/hooks/src/use-resend-code-form.ts", import.meta.url).href)};`,
  "@plane/ui": `export {Input} from ${JSON.stringify(new URL("../../packages/ui/src/form-fields/input.tsx", import.meta.url).href)};
    export {AuthCodeResend} from ${JSON.stringify(new URL("../../packages/ui/src/auth-form/code-resend.tsx", import.meta.url).href)};
    export const Spinner=()=>null;`,
  "@plane/propel/button": `const R=globalThis.authResendConsumerReact;export const Button=({variant,size,children,...props})=>R.createElement('button',props,children);`,
  "@plane/propel/toast":
    "export const TOAST_TYPE={ERROR:'error'};export const setToast=()=>globalThis.authResendConsumer.failures++;",
  "@/types/auth": "export const EAuthModes={SIGN_IN:'SIGN_IN',SIGN_UP:'SIGN_UP'};",
  "@/helpers/authentication.helper": `export const EAuthModes={SIGN_IN:'SIGN_IN',SIGN_UP:'SIGN_UP'};export const showAuthenticationError=()=>globalThis.authResendConsumer.failures++;`,
  "@/services/auth.service":
    "export const authService={submitForm:()=>{throw Error('Unexpected authentication submit');}};",
  "@plane/services": "export class AuthService {submitForm(){throw Error('Unexpected authentication submit');}}",
});

async function testConsumer(kind) {
  await test(`${kind} code form retains failed codes, locks pending resend and clears only after delivery`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const state = (globalThis.authResendConsumer = { failures: 0, request: Promise.withResolvers(), sent: [] });
    const { AuthUniqueCodeForm } =
      kind === "web"
        ? await import("../../apps/web/core/components/account/auth-forms/unique-code.tsx")
        : await import("../../apps/space/components/account/auth-forms/unique-code.tsx");
    t.mock.timers.enable({ apis: ["setTimeout"] });
    await env.render(
      React.createElement(AuthUniqueCodeForm, {
        mode: "SIGN_IN",
        email: "person@example.com",
        nextPath: "/destination",
        isExistingEmail: true,
        handleEmailClear: () => {},
        generateEmailUniqueCode: (email) => {
          state.sent.push(email);
          return state.request.promise;
        },
      })
    );
    const code = env.container.querySelector('input[name="code"]');
    await changeFormControl(code, "123456");
    const resend = () =>
      Array.from(env.container.querySelectorAll("button")).find((b) => /resend/i.test(b.textContent));
    const continueButton = env.container.querySelector('button[type="submit"]');
    assert.equal(continueButton.disabled, false);
    const resendButton = resend();
    await act(async () => resendButton.click());
    assert.deepEqual(state.sent, ["person@example.com"]);
    assert.equal(resendButton.disabled, true);
    assert.equal(continueButton.disabled, true);
    await act(async () => resendButton.click());
    assert.equal(state.sent.length, 1);
    await act(async () => state.request.reject(Error("offline")));
    assert.equal(state.failures, 1);
    assert.equal(code.value, "123456");
    assert.equal(resendButton.disabled, false);
    assert.equal(continueButton.disabled, false);
    state.request = Promise.withResolvers();
    await act(async () => resendButton.click());
    await act(async () => state.request.resolve());
    assert.equal(code.value, "");
    assert.equal(continueButton.disabled, true);
    assert.equal(resendButton.disabled, true);
    assert.match(resendButton.textContent, /5/);
    assert.equal(env.container.querySelector('input[name="email"]').value, "person@example.com");
    assert.equal(env.container.querySelector('input[name="next_path"]').value, "/destination");
  });
}

await testConsumer("web");
await testConsumer("space");

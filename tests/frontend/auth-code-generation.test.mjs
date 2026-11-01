import assert from "node:assert/strict";
import test from "node:test";
import { EAuthModes, EAuthSteps } from "../../packages/constants/dist/index.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

const state = {};
globalThis.vectorAuthCodeGeneration = state;
registerWebTestModules({
  "@tanstack/react-router": `export const useRouter = () => ({navigate:({href})=>{globalThis.vectorAuthCodeGeneration.navigations.push(href);return Promise.resolve();}});
    export const useLocation = ({ select }) => select({ searchStr: "" });`,
  "@tanstack/react-query": "export const useQuery = () => ({ data: { is_smtp_configured: true } });",
  "@/queries/instance": "export const instanceConfigOptions = {};",
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "@plane/ui":
    "export const EmailEntryForm = props => { globalThis.vectorAuthCodeGeneration.form = props; return null; };",
  "./password":
    "export const AuthPasswordForm = props => { globalThis.vectorAuthCodeGeneration.form = props; return null; };",
  "./unique-code":
    "export const AuthUniqueCodeForm = props => { globalThis.vectorAuthCodeGeneration.form = props; return null; };",
  "@/services/auth.service": `export const authService = {
    emailCheck: async () => globalThis.vectorAuthCodeGeneration.emailResponse,
    generateUniqueCode: data => {
      globalThis.vectorAuthCodeGeneration.sent.push(data);
      return globalThis.vectorAuthCodeGeneration.request.promise;
    }
  };`,
  "@/helpers/authentication.helper": `export const authErrorHandler = () => undefined;
    export const showAuthenticationError = () => { globalThis.vectorAuthCodeGeneration.failures++; };`,
});
const { AuthFormRoot } = await import("../../apps/web/core/components/account/auth-forms/form-root.tsx");

async function renderForm(t, authStep, existing = true, mode = EAuthModes.SIGN_IN) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  Object.assign(state, {
    request: Promise.withResolvers(),
    sent: [],
    failures: 0,
    navigations: [],
    emailResponse: { existing, status: "MAGIC_CODE" },
  });
  const props = {
    authStep,
    authMode: mode,
    currentAuthMode: mode,
    email: "person@example.com",
    setEmail: (email) => {
      state.email = email;
    },
    setAuthMode: (nextMode) => {
      state.mode = nextMode;
    },
    setAuthStep: (step) => {
      state.step = step;
    },
    setErrorInfo: (error) => {
      state.error = error;
    },
  };
  await env.render(React.createElement(AuthFormRoot, props));
  return props;
}

for (const existing of [true, false]) {
  void test(`email verification owns a failed code request for an ${existing ? "existing" : "unknown"} email`, async (t) => {
    const props = await renderForm(t, EAuthSteps.EMAIL, existing);
    let submission;
    let settled = false;
    await act(async () => {
      submission = state.form.onSubmit({ email: props.email }).finally(() => {
        settled = true;
      });
      await Promise.resolve();
    });
    assert.equal(settled, false, "submission must wait for the code request");
    assert.deepEqual(state.sent, [{ email: props.email }]);
    assert.equal(state.step, EAuthSteps.UNIQUE_CODE);
    if (!existing) assert.equal(state.mode, EAuthModes.SIGN_UP);
    await act(async () => {
      state.request.reject(new Error("offline"));
      await submission;
    });
    assert.equal(state.failures, 1);
  });
}

await test("switching from password to code reports a failed request without an unhandled rejection", async (t) => {
  await renderForm(t, EAuthSteps.PASSWORD);
  await act(async () => {
    state.form.handleAuthStep(EAuthSteps.UNIQUE_CODE);
    state.request.reject(new Error("offline"));
  });
  assert.equal(state.step, EAuthSteps.UNIQUE_CODE);
  assert.equal(state.failures, 1);
});

await test("the resend caller still receives rejection so it can retain its code and cooldown", async (t) => {
  const props = await renderForm(t, EAuthSteps.UNIQUE_CODE);
  const result = assert.rejects(state.form.generateEmailUniqueCode(props.email), /offline/);
  state.request.reject(new Error("offline"));
  await result;
  assert.equal(state.failures, 0, "resend owns its generic failure feedback");
});

for (const mode of [EAuthModes.SIGN_IN, EAuthModes.SIGN_UP]) {
  void test(`changing the email resets the ${mode} flow and returns to its entry route`, async (t) => {
    await renderForm(t, EAuthSteps.PASSWORD, true, mode);
    state.email = "previous@example.test";
    state.error = { message: "Old error" };
    await act(async () => state.form.handleEmailClear());
    assert.equal(state.email, "");
    assert.equal(state.error, undefined);
    assert.equal(state.mode, mode);
    assert.equal(state.step, EAuthSteps.EMAIL);
    assert.deepEqual(state.navigations, [mode === EAuthModes.SIGN_IN ? "/" : "/sign-up"]);
  });
}

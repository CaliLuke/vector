import assert from "node:assert/strict";
import { registerHooks } from "./helpers/module-hooks.mjs";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

const state = {};
globalThis.spaceAuthCodeFixture = state;
registerSpaceModuleHooks();
const stubs = {
  "react-router":
    "export const useSearchParams = () => [globalThis.spaceAuthCodeFixture.params]; export const Link = () => null;",
  "@tanstack/react-query": `export const useQuery = () => ({ data: { config: {
    is_smtp_configured: true, is_magic_login_enabled: true, is_email_password_enabled: true
  } } });`,
  "@/queries/session": "export const instanceInfoOptions = {};",
  "@/hooks/oauth": "export const useOAuthConfig = () => ({ isOAuthEnabled: false, oAuthOptions: [] });",
  "@plane/ui": `export const AuthMessageBanner = ({bannerData}) => bannerData?.message ?? null;
export const OAuthOptions = () => null;
    export const EmailEntryForm = props => { globalThis.spaceAuthCodeFixture.form = props; return null; };`,
  "./password":
    "export const AuthPasswordForm = props => { globalThis.spaceAuthCodeFixture.form = props; return null; };",
  "./unique-code":
    "export const AuthUniqueCodeForm = props => { globalThis.spaceAuthCodeFixture.form = props; return null; };",
  "./auth-header": "export const AuthHeader = () => null;",
  "../terms-and-conditions": "export const TermsAndConditions = () => null;",
  "@plane/services": `export class SitesAuthService {
    async emailCheck() { return { existing: true, status: 'MAGIC_CODE' }; }
    generateUniqueCode() { return globalThis.spaceAuthCodeFixture.request.promise; }
  }`,
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { AuthRoot } = await import("../../apps/space/components/account/auth-forms/auth-root.tsx");
const { EAuthSteps } = await import("../../apps/space/types/auth.ts");

async function fixture(t, errorCode = "") {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  state.params = new URLSearchParams({ email: "reader@example.com", error_code: errorCode });
  state.request = Promise.withResolvers();
  await env.render(React.createElement(AuthRoot));
  return env;
}

await test("public email verification waits for code delivery and displays unknown failures", async (t) => {
  const env = await fixture(t);
  let submission;
  let finished = false;
  await act(async () => {
    submission = state.form.onSubmit({ email: "reader@example.com" }).finally(() => {
      finished = true;
    });
    await Promise.resolve();
  });
  assert.equal(finished, false);
  await act(async () => {
    state.request.reject(new Error("offline"));
    await submission;
  });
  assert.match(env.container.textContent, /Please try again/);
});

await test("public password-to-code switching owns a failed delivery and permits resend", async (t) => {
  const env = await fixture(t, "5065");
  await act(async () => {
    state.form.handleAuthStep(EAuthSteps.UNIQUE_CODE);
    state.request.reject(new Error("offline"));
  });
  assert.match(env.container.textContent, /Please try again/);
  state.request = Promise.withResolvers();
  const resend = assert.rejects(state.form.generateEmailUniqueCode("reader@example.com"), /offline/);
  state.request.reject(new Error("offline"));
  await resend;
});

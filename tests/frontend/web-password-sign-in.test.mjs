import assert from "node:assert/strict";
import test from "node:test";
import { EAuthModes } from "../../packages/constants/dist/index.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

const requests = [];
globalThis.vectorPasswordSignIn = { requests };
registerWebTestModules({
  "@/services/auth.service": `export const authService = {
    submitPasswordSignIn: form => {
      globalThis.vectorPasswordSignIn.requests.push(form);
      return globalThis.vectorPasswordSignIn.pending.promise;
    }
  };`,
  "@tanstack/react-router": "export const Link = () => null;",
  "./forgot-password-popover": "export const ForgotPasswordPopover = () => null;",
});
const { AuthPasswordForm } = await import("../../apps/web/core/components/account/auth-forms/password.tsx");

void test("password sign-in keeps the form in place and renders the returned rejection", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const attempt = Promise.withResolvers();
  globalThis.vectorPasswordSignIn.pending = attempt;

  await environment.render(
    React.createElement(AuthPasswordForm, {
      email: "reader@example.test",
      isSMTPConfigured: false,
      mode: EAuthModes.SIGN_IN,
      handleAuthStep() {},
      handleEmailClear() {},
      nextPath: "/settings/profile/api-tokens/",
    })
  );
  const form = environment.container.querySelector("form");
  const input = form.querySelector('input[name="password"]');
  const initialLocation = window.location.href;
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, "fixture-password");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await act(async () => {
    form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });
  assert.deepEqual(requests, [form]);
  assert.equal(environment.container.querySelector('button[type="submit"]').disabled, true);
  await act(async () => attempt.resolve(new URL("/?error_code=5065", initialLocation)));

  assert.match(environment.container.querySelector('[role="alert"]').textContent, /authentication failed/i);
  assert.equal(environment.container.querySelector("form"), form);
  assert.equal(window.location.href, initialLocation);
  assert.equal(environment.container.querySelector('button[type="submit"]').disabled, false);
});

void test("the password sign-in transport submits CSRF and returns the redirect without navigating", async (t) => {
  const { AuthService } = await import("../../packages/services/dist/index.js");
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const service = new AuthService("http://localhost/");
  t.mock.method(service, "requestCSRFToken", async () => ({ csrf_token: "fixture-csrf" }));
  t.mock.method(globalThis, "fetch", async (action, options) => {
    assert.equal(action.href, "http://localhost/auth/sign-in/");
    assert.equal(options.method, "POST");
    assert.equal(options.credentials, "same-origin");
    assert.equal(options.mode, "same-origin");
    assert.equal(options.body.get("csrfmiddlewaretoken"), "fixture-csrf");
    assert.equal(options.body.get("password"), "fixture-password");
    return { ok: true, redirected: true, url: "http://localhost/?next_path=%2Fvector" };
  });
  const originalFormData = globalThis.FormData;
  globalThis.FormData = window.FormData;
  t.after(() => {
    globalThis.FormData = originalFormData;
  });
  environment.container.innerHTML = `
    <form action="/auth/sign-in/">
      <input name="csrfmiddlewaretoken">
      <input name="password" value="fixture-password">
    </form>
  `;
  const destination = await service.submitPasswordSignIn(environment.container.querySelector("form"));
  assert.equal(destination.searchParams.get("next_path"), "/vector");
  assert.equal(window.location.href, "http://localhost/");
});

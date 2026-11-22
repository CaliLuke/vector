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
  t.mock.method(service, "get", async (path, options) => {
    assert.equal(path, "/auth/get-csrf-token/");
    assert.equal(options.validateStatus, null);
    return { data: { csrf_token: "fixture-csrf" } };
  });
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

void test("cross-origin password sign-in uses the original form submission with fresh CSRF", async (t) => {
  const { AuthService } = await import("../../packages/services/dist/index.js");
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const service = new AuthService("http://localhost/");
  t.mock.method(service, "requestCSRFToken", async () => ({ csrf_token: "fixture-csrf" }));
  const fetch = t.mock.method(globalThis, "fetch", async () => assert.fail("used fetch for a cross-origin form"));
  environment.container.innerHTML = `
    <form action="https://auth.example.test/sign-in/" method="post">
      <input name="csrfmiddlewaretoken">
    </form>
  `;
  const form = environment.container.querySelector("form");
  const submit = t.mock.method(form, "submit", () => {
    assert.equal(form.elements.namedItem("csrfmiddlewaretoken").value, "fixture-csrf");
    assert.equal(form.action, "https://auth.example.test/sign-in/");
  });

  assert.equal(await service.submitPasswordSignIn(form), undefined);
  assert.equal(submit.mock.callCount(), 1);
  assert.equal(fetch.mock.callCount(), 0);
});

void test("password sign-in requires a connected form and a fresh CSRF field", async (t) => {
  const { AuthService } = await import("../../packages/services/dist/index.js");
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const service = new AuthService("http://localhost/");
  let csrf = { csrf_token: "fixture-csrf" };
  t.mock.method(service, "requestCSRFToken", async () => csrf);
  const fetch = t.mock.method(globalThis, "fetch", async () => assert.fail("posted an invalid form"));
  environment.container.innerHTML = '<form action="/auth/sign-in/"></form>';
  const form = environment.container.querySelector("form");

  await assert.rejects(service.submitPasswordSignIn(null), /Authentication form is unavailable/);
  await assert.rejects(service.submitPasswordSignIn(form), /CSRF field not found/);
  csrf = {};
  await assert.rejects(service.submitPasswordSignIn(form), /CSRF token not found/);
  csrf = { csrf_token: "fixture-csrf" };
  form.remove();
  await assert.rejects(service.submitPasswordSignIn(form), /Authentication form was closed/);
  assert.equal(fetch.mock.callCount(), 0);
});

void test("password sign-in rejects non-redirect responses and ignores results after the form closes", async (t) => {
  const { AuthService } = await import("../../packages/services/dist/index.js");
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const service = new AuthService("http://localhost/");
  t.mock.method(service, "requestCSRFToken", async () => ({ csrf_token: "fixture-csrf" }));
  const originalFormData = globalThis.FormData;
  globalThis.FormData = window.FormData;
  t.after(() => {
    globalThis.FormData = originalFormData;
  });
  environment.container.innerHTML = '<form action="/auth/sign-in/"><input name="csrfmiddlewaretoken"></form>';
  const form = environment.container.querySelector("form");
  const started = Promise.withResolvers();
  const pending = Promise.withResolvers();
  let fetchResponse = { ok: true, redirected: false };
  const fetch = t.mock.method(globalThis, "fetch", async () => fetchResponse);

  await assert.rejects(service.submitPasswordSignIn(form), /Unexpected sign-in response/);
  fetchResponse = { ok: false, redirected: true };
  await assert.rejects(service.submitPasswordSignIn(form), /Unexpected sign-in response/);

  fetch.mock.mockImplementation(async () => {
    started.resolve();
    return pending.promise;
  });
  const result = service.submitPasswordSignIn(form);
  await started.promise;
  form.remove();
  pending.resolve({ ok: true, redirected: true, url: "http://localhost/?next_path=%2Fvector" });
  assert.equal(await result, undefined);
  assert.equal(window.location.href, "http://localhost/");
});

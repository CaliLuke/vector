import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider, notifyManager } from "@tanstack/react-query";
import { submitForm } from "./helpers/native-forms.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerSpaceModuleHooks();
const require = createRequire(new URL("../../apps/space/package.json", import.meta.url));
const { AuthService } = require("@plane/services");
const toasts = [];
globalThis.spaceAuthToasts = toasts;
registerHooks({
  resolve: (specifier, context, nextResolve) =>
    resolvePackageStub(
      specifier,
      {
        "@plane/propel/toast": `export const TOAST_TYPE = { ERROR: "error" }; export const setToast = value => globalThis.spaceAuthToasts.push(value);`,
      },
      context,
      nextResolve
    ),
});
const { EAuthModes } = await import("../../apps/space/types/auth.ts");
await test("public password sign-in submits only after a fresh CSRF token and recovers from token failure", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const { AuthPasswordForm } = await import("../../apps/space/components/account/auth-forms/password.tsx");
  t.after(() => environment.unmount());
  let fail = true;
  const submitted = [];
  t.mock.method(AuthService.prototype, "requestCSRFToken", async () => {
    if (fail) throw new Error("offline");
    return { csrf_token: "fresh-token" };
  });
  t.mock.method(HTMLFormElement.prototype, "submit", function () {
    submitted.push(this.querySelector('input[name="csrfmiddlewaretoken"]').value);
  });
  await environment.render(
    React.createElement(AuthPasswordForm, {
      email: "reader@example.com",
      mode: EAuthModes.SIGN_IN,
      isSMTPConfigured: false,
      handleAuthStep() {},
      handleEmailClear() {},
    })
  );
  const form = environment.container.querySelector("form");
  const submit = () => act(async () => form.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
  const input = environment.container.querySelector('input[name="password"]');
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, "reader-password");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await submit();
  assert.deepEqual(submitted, []);
  assert.equal(toasts.at(-1).type, "error");
  fail = false;
  await submit();
  assert.deepEqual(submitted, ["fresh-token"]);
});

await test("public sign-out preserves session data after CSRF failure and clears it only after posting", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const { MemoryRouter } = require("react-router");
  const { UserAvatar } = await import("../../apps/space/components/issues/navbar/user-avatar.tsx");
  const { currentUserOptions } = await import("../../apps/space/queries/session.ts");
  notifyManager.setScheduler(queueMicrotask);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const user = { id: "reader", first_name: "Test", last_name: "Reader", email: "reader@example.com" };
  client.setQueryData(currentUserOptions.queryKey, user);
  client.setQueryData(["private-draft"], "retained");
  t.after(async () => {
    await environment.unmount();
    client.clear();
  });
  let fail = true;
  const posted = [];
  t.mock.method(AuthService.prototype, "requestCSRFToken", async () => {
    if (fail) throw new Error("offline");
    return { csrf_token: "fresh-logout-token" };
  });
  t.mock.method(HTMLFormElement.prototype, "submit", function () {
    posted.push({
      token: this.querySelector('[name="csrfmiddlewaretoken"]').value,
      path: this.querySelector('[name="next_path"]').value,
    });
  });
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(
        MemoryRouter,
        { initialEntries: ["/public-project?board=kanban"] },
        React.createElement(UserAvatar)
      )
    )
  );
  await act(async () => environment.container.querySelector("button").click());
  await submitForm(environment.container);
  assert.deepEqual(posted, []);
  assert.equal(client.getQueryData(["private-draft"]), "retained");
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Could not sign out/);
  fail = false;
  await submitForm(environment.container);
  assert.equal(posted[0].token, "fresh-logout-token");
  assert.match(posted[0].path, /^\/public-project\?/);
  assert.equal(client.getQueryData(["private-draft"]), undefined);
});

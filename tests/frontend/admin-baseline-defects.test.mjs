import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerAdminFormHooks, changeInput, formToasts } from "./helpers/admin-forms.mjs";

registerAdminFormHooks();
const require = createRequire(new URL("../../apps/admin/package.json", import.meta.url));
const { AuthService, InstanceWorkspaceService } = require("@plane/services");

async function mountAuthForm(t, name, path) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { MemoryRouter } = createRequire(new URL("../../apps/admin/package.json", import.meta.url))("react-router");
  const { InstanceSignInForm } = await import("../../apps/admin/app/(all)/(home)/sign-in-form.tsx");
  const { InstanceSetupForm } = await import("../../apps/admin/components/instance/setup-form.tsx");
  await environment.render(
    React.createElement(
      MemoryRouter,
      { initialEntries: [path] },
      React.createElement(name === "setup" ? InstanceSetupForm : InstanceSignInForm)
    )
  );
  return environment;
}
const submit = (container) =>
  act(async () => {
    container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
  });

await test("admin sign-in retains credentials after a failed CSRF request and submits with a fresh token on retry", async (t) => {
  let fail = true;
  const submissions = [];
  t.mock.method(AuthService.prototype, "requestCSRFToken", async () => {
    if (fail) throw new Error("offline");
    return { csrf_token: "fresh-token" };
  });
  const { container } = await mountAuthForm(t, "sign-in", "/?email=admin@example.com");
  t.mock.method(HTMLFormElement.prototype, "submit", function () {
    submissions.push(Object.fromEntries(new window.FormData(this)));
  });
  await changeInput(container.querySelector('[name="password"]'), "test-password");
  await submit(container);
  assert.deepEqual(submissions, []);
  assert.equal(formToasts.at(-1).type, "error");
  assert.equal(container.querySelector('[name="password"]').value, "test-password");
  fail = false;
  await submit(container);
  assert.deepEqual(submissions, [
    { email: "admin@example.com", password: "test-password", csrfmiddlewaretoken: "fresh-token" },
  ]);
});

await test("setup preserves a disabled telemetry choice and rejects mismatched passwords before posting", async (t) => {
  const tokenRequest = t.mock.method(AuthService.prototype, "requestCSRFToken", async () => ({
    csrf_token: "fresh-token",
  }));
  const { container } = await mountAuthForm(
    t,
    "setup",
    "/?first_name=Luca&email=admin@example.com&is_telemetry_enabled=False"
  );
  const submissions = [];
  t.mock.method(HTMLFormElement.prototype, "submit", function () {
    submissions.push(Object.fromEntries(new window.FormData(this)));
  });
  assert.equal(container.querySelector('[name="is_telemetry_enabled"]').value, "False");
  assert.equal(container.querySelector('[type="checkbox"]').checked, false);
  await changeInput(container.querySelector('[name="password"]'), "Strong!Password123");
  await changeInput(container.querySelector('[name="confirm_password"]'), "mismatch");
  await submit(container);
  assert.equal(tokenRequest.mock.callCount(), 0);
  await changeInput(container.querySelector('[name="confirm_password"]'), "Strong!Password123");
  await submit(container);
  assert.equal(submissions.length, 1);
  assert.equal(submissions[0].is_telemetry_enabled, "False");
  assert.equal(submissions[0].csrfmiddlewaretoken, "fresh-token");
});

await test("workspace loading settles into success with an empty result", async (t) => {
  t.mock.method(InstanceWorkspaceService.prototype, "list", async () => ({
    results: [],
    next_cursor: null,
    next_page_results: false,
  }));
  const { QueryClient, InfiniteQueryObserver } = webRequire("@tanstack/react-query");
  const { instanceWorkspacesOptions } = await import("../../apps/admin/queries/workspaces.ts");
  const client = new QueryClient();
  t.after(() => client.clear());
  const observer = new InfiniteQueryObserver(client, instanceWorkspacesOptions);
  const result = await observer.refetch();
  assert.equal(result.status, "success");
  assert.equal(result.isFetching, false);
  assert.deepEqual(result.data.pages[0].results, []);
});

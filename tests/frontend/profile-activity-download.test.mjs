import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const state = { route: {}, accountId: "account", generation: 0, toasts: [] };
globalThis.activityDownload = state;
const stubs = {
  "@tanstack/react-router": "export const useParams = () => globalThis.activityDownload.route;",
  "@tanstack/react-query": "export const useQueryClient = () => ({});",
  "@/hooks/use-current-user": "export const useCurrentUserId = () => globalThis.activityDownload.accountId;",
  "@/hooks/use-identity-generation":
    "export const useIdentityGeneration = () => globalThis.activityDownload.generation;",
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "@plane/propel/toast":
    "export const TOAST_TYPE = { ERROR: 'error' }; export const setToast = toast => globalThis.activityDownload.toasts.push(toast);",
  "@/services/user.service":
    "export class UserService { downloadProfileActivity(...args) { return globalThis.activityDownload.download(...args); } }",
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { DownloadActivityButton } = await import("../../apps/web/core/components/profile/activity/download-button.tsx");

async function setup(t) {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  state.route = { workspaceSlug: "workspace", userId: "profile" };
  state.accountId = "account";
  state.generation = 0;
  state.toasts = [];
  const requests = [];
  state.download = (...args) => {
    const request = { ...Promise.withResolvers(), args };
    requests.push(request);
    return request.promise;
  };
  const blobs = [];
  const revoked = [];
  window.URL.createObjectURL = (blob) => {
    blobs.push(blob);
    return "blob:activity";
  };
  window.URL.revokeObjectURL = (url) => revoked.push(url);
  const clicks = t.mock.method(window.HTMLAnchorElement.prototype, "click", () => {});
  const render = () => env.render(React.createElement(DownloadActivityButton));
  await render();
  return {
    env,
    requests,
    blobs,
    revoked,
    clicks,
    render,
    click: () => act(async () => env.container.querySelector("button").click()),
  };
}

void test("activity download reports failure, permits retry, and releases the successful download", async (t) => {
  const fixture = await setup(t);
  await fixture.click();
  await fixture.click();
  assert.equal(fixture.requests.length, 1);
  await act(async () => fixture.requests[0].reject(new Error("private transport details")));
  assert.equal(state.toasts[0].title, "Could not download activity");
  assert.doesNotMatch(JSON.stringify(state.toasts), /private transport/);
  await fixture.click();
  assert.equal(fixture.requests.length, 2);
  assert.deepEqual(fixture.requests[1].args.slice(0, 2), ["workspace", "profile"]);
  await act(async () => fixture.requests[1].resolve("name,count\nAlice,2"));
  assert.equal(fixture.blobs[0].type, "text/csv");
  assert.equal(fixture.clicks.mock.callCount(), 1);
  assert.deepEqual(fixture.revoked, ["blob:activity"]);
  assert.equal(document.querySelector("a[download]"), null);
  assert.equal(fixture.env.container.querySelector("button").disabled, false);
});

void test("failed browser download still releases its URL and temporary link", async (t) => {
  const fixture = await setup(t);
  fixture.clicks.mock.mockImplementation(() => {
    throw new Error("browser failure");
  });
  await fixture.click();
  await act(async () => fixture.requests[0].resolve("csv"));
  assert.deepEqual(fixture.revoked, ["blob:activity"]);
  assert.equal(document.querySelector("a[download]"), null);
  assert.equal(state.toasts.length, 1);
  assert.equal(fixture.env.container.querySelector("button").disabled, false);
});

void test("replaced profiles and sessions cannot publish a late download or failure", async (t) => {
  const fixture = await setup(t);
  await fixture.click();
  state.route = { ...state.route, userId: "replacement" };
  await fixture.render();
  assert.equal(fixture.env.container.querySelector("button").disabled, false);
  await act(async () => fixture.requests[0].resolve("old profile"));
  assert.equal(fixture.blobs.length, 0);
  await fixture.click();
  state.generation++;
  await fixture.render();
  await act(async () => fixture.requests[1].reject(new Error("old session")));
  assert.deepEqual(state.toasts, []);
  await fixture.click();
  await fixture.env.unmount();
  await act(async () => fixture.requests[2].resolve("unmounted"));
  assert.equal(fixture.blobs.length, 0);
});

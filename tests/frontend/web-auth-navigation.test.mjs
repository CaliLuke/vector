import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createRouterFixture, routerApi } from "./helpers/router.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

const state = { workspaceFetches: 0 };
globalThis.vectorAuthNavigationTest = state;
const stubs = {
  "@/components/common/logo-spinner": `export const LogoSpinner = () => null;`,
  "@/helpers/authentication.helper": `export const EPageTypes = {PUBLIC:"PUBLIC",NON_AUTHENTICATED:"NON_AUTHENTICATED",SET_PASSWORD:"SET_PASSWORD",ONBOARDING:"ONBOARDING",AUTHENTICATED:"AUTHENTICATED"};`,
};
registerWebModuleHooks();
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { AuthenticationWrapper } = await import("../../apps/web/core/lib/wrappers/authentication-wrapper.tsx");
const { QueryClient, QueryClientProvider } = await import("@tanstack/react-query");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { workspaceKeys } = await import("../../apps/web/core/queries/workspaces.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");

function authClient(t, environment, user, profile, seedIdentity = true) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  if (seedIdentity) client.setQueryData(identityKeys.currentUser(), user ?? null);
  if (user) {
    client.setQueryData(identityKeys.profile(user.id), profile);
    client.setQueryData(identityKeys.settings(user.id), { workspace: { last_workspace_slug: "acme" } });
    client.setQueryData(workspaceKeys.list(user.id), [{ id: "workspace", slug: "acme" }]);
  }
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    await environment.unmount();
  });
  return client;
}

async function renderAuth(t, { path, pageType, user, profile, seedIdentity = true }) {
  const environment = await createReactDomTestEnvironment();
  const client = authClient(t, environment, user, profile, seedIdentity);
  const fixture = createRouterFixture(path);
  const initialPath = new URL(path, "http://localhost").pathname;
  function Harness() {
    const pathname = routerApi.useLocation({ select: (location) => location.pathname });
    if (pathname !== initialPath) return React.createElement("output", null, pathname);
    return React.createElement(
      AuthenticationWrapper,
      { pageType },
      React.createElement("div", { "data-protected": true }, "Protected content")
    );
  }
  await fixture.render(
    environment,
    React.createElement(
      React.StrictMode,
      null,
      React.createElement(QueryClientProvider, { client }, React.createElement(Harness))
    )
  );
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 25));
  });
  return { environment, router: fixture.router, client };
}

function waitForQueryState(client, predicate) {
  if (predicate()) return Promise.resolve();
  return new Promise((resolve) => {
    const unsubscribe = client.getQueryCache().subscribe(() => {
      if (predicate()) {
        unsubscribe();
        resolve();
      }
    });
  });
}

function waitForDom(predicate) {
  if (predicate()) return Promise.resolve();
  return new Promise((resolve) => {
    const observer = new MutationObserver(() => {
      if (predicate()) {
        observer.disconnect();
        resolve();
      }
    });
    observer.observe(document.body, { childList: true, subtree: true, characterData: true });
  });
}

await test("an unauthenticated deep link reaches sign-in without rendering protected content", async (t) => {
  const { environment, router } = await renderAuth(t, {
    path: "/acme/projects/p1/issues?view=board#selected",
    pageType: "AUTHENTICATED",
  });
  assert.equal(router.state.location.pathname, "/");
  assert.equal(router.state.location.search.next_path, "/acme/projects/p1/issues?view=board#selected");
  assert.equal(environment.container.querySelector("[data-protected]"), null);
});

await test("an authenticated sign-in redirect preserves its requested destination", async (t) => {
  const { router } = await renderAuth(t, {
    path: "/?next_path=%2Facme%2Fprojects%2Fp1%2Fissues%3Fview%3Dboard%23selected",
    pageType: "NON_AUTHENTICATED",
    user: { id: "user" },
    profile: { id: "profile", is_onboarded: true },
  });
  assert.equal(router.state.location.href, "/acme/projects/p1/issues?view=board#selected");
});

await test("a protocol-relative login destination cannot leave the application", async (t) => {
  const { router } = await renderAuth(t, {
    path: "/?next_path=%2F%2Fexample.org",
    pageType: "NON_AUTHENTICATED",
    user: { id: "user" },
    profile: { id: "profile", is_onboarded: true },
  });
  assert.equal(router.state.location.pathname, "/acme");
});

for (const [pageType, path] of [
  ["ONBOARDING", "/onboarding"],
  ["SET_PASSWORD", "/accounts/set-password"],
]) {
  void test(`an unauthenticated ${pageType} page requires sign-in`, async (t) => {
    const { router } = await renderAuth(t, { path, pageType });
    assert.equal(router.state.location.pathname, "/");
    assert.equal(router.state.location.search.next_path, path);
  });
  void test(`a completed profile leaves ${pageType}`, async (t) => {
    const { router } = await renderAuth(t, {
      path,
      pageType,
      user: { id: "user", is_password_autoset: false },
      profile: { id: "profile", is_onboarded: true },
    });
    assert.equal(router.state.location.pathname, "/acme");
  });
}

await test("an automatically assigned password keeps the password setup page accessible", async (t) => {
  const { environment, router } = await renderAuth(t, {
    path: "/accounts/set-password",
    pageType: "SET_PASSWORD",
    user: { id: "user", is_password_autoset: true },
    profile: { id: "profile", is_onboarded: true },
  });
  assert.equal(router.state.location.pathname, "/accounts/set-password");
  assert.ok(environment.container.querySelector("[data-protected]"), environment.container.textContent);
});

await test("an incomplete profile reaches onboarding before protected content", async (t) => {
  const { environment, router } = await renderAuth(t, {
    path: "/acme",
    pageType: "AUTHENTICATED",
    user: { id: "user" },
    profile: { id: "profile", is_onboarded: false },
  });
  assert.equal(router.state.location.pathname, "/onboarding");
  assert.equal(environment.container.querySelector("[data-protected]"), null);
});

await test("a delayed sign-in route retains the full protected destination", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const client = authClient(t, environment);
  const { createMemoryHistory, createRootRoute, createRoute, createRouter, Outlet, RouterProvider } = routerApi;
  const root = createRootRoute({ component: Outlet });
  const signIn = createRoute({
    getParentRoute: () => root,
    path: "/",
    loader: async () => new Promise((resolve) => setTimeout(resolve, 30)),
    component: () => React.createElement(AuthenticationWrapper, { pageType: "NON_AUTHENTICATED" }, "Sign in"),
  });
  const protectedRoute = createRoute({
    getParentRoute: () => root,
    path: "/acme/projects/$projectId/issues",
    component: () => React.createElement(AuthenticationWrapper, null, "Protected content"),
  });
  const destination = "/acme/projects/p1/issues?view=board#selected";
  const router = createRouter({
    routeTree: root.addChildren([signIn, protectedRoute]),
    history: createMemoryHistory({ initialEntries: [destination] }),
    isServer: false,
    defaultPendingMinMs: 0,
  });
  await act(async () => {
    await router.load();
  });
  await environment.render(
    React.createElement(QueryClientProvider, { client }, React.createElement(RouterProvider, { router }))
  );
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 80));
  });
  assert.equal(router.state.location.pathname, "/");
  assert.equal(router.state.location.search.next_path, destination);
  assert.equal(environment.container.textContent, "Sign in");
});

await test("authenticated routes wait for the profile before deciding onboarding access", async (t) => {
  const { default: userService } = await import("../../apps/web/core/services/user.service.ts");
  const pending = Promise.withResolvers();
  t.mock.method(userService, "getCurrentUserProfile", () => pending.promise);
  const { environment, router } = await renderAuth(t, {
    path: "/acme/projects/p1/issues",
    pageType: "AUTHENTICATED",
    user: { id: "user" },
  });
  assert.equal(router.state.location.pathname, "/acme/projects/p1/issues");
  assert.equal(environment.container.querySelector("[data-protected]"), null);
  await act(async () => pending.resolve({ id: "profile", user: "user", is_onboarded: true }));
  await act(async () => new Promise((resolve) => setTimeout(resolve, 25)));
  assert.ok(environment.container.querySelector("[data-protected]"), environment.container.textContent);
});

async function retryFailedAccountQuery(t, failedResource) {
  const { default: userService } = await import("../../apps/web/core/services/user.service.ts");
  const requests = { user: 0, profile: 0 };
  t.mock.method(userService, "currentUser", async () => {
    requests.user++;
    if (failedResource === "user" && requests.user === 1) throw new Error("offline");
    return { id: "user", is_password_autoset: true };
  });
  t.mock.method(userService, "getCurrentUserProfile", async () => {
    requests.profile++;
    if (failedResource === "profile" && requests.profile === 1) throw new Error("profile offline");
    return { id: "profile", user: "user", is_onboarded: true };
  });
  t.mock.method(userService, "currentUserSettings", async () => ({ id: "user", workspace: {} }));
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => {
    state.workspaceFetches++;
    return [{ id: "workspace", slug: "acme" }];
  });
  state.workspaceFetches = 0;
  const { environment, client } = await renderAuth(t, {
    path: "/acme/projects/p1/issues",
    pageType: "AUTHENTICATED",
    seedIdentity: false,
  });
  const failedKey = failedResource === "user" ? identityKeys.currentUser() : identityKeys.profile("user");

  await act(async () => {
    await waitForQueryState(client, () => client.getQueryState(failedKey)?.status === "error");
  });
  assert.equal(environment.container.querySelector("[data-protected]"), null);
  assert.match(environment.container.querySelector('[role="alert"]')?.textContent ?? "", /Unable to load your account/);

  const accountLoaded = waitForQueryState(
    client,
    () =>
      client.getQueryState(identityKeys.currentUser())?.status === "success" &&
      client.getQueryState(identityKeys.profile("user"))?.status === "success" &&
      client.getQueryState(identityKeys.settings("user"))?.status === "success" &&
      client.getQueryState(workspaceKeys.list("user"))?.status === "success"
  );
  await act(async () => {
    environment.container.querySelector("button")?.click();
    await accountLoaded;
    await waitForDom(() => Boolean(environment.container.querySelector("[data-protected]")));
  });
  assert.ok(environment.container.querySelector("[data-protected]"), environment.container.textContent);
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
  return { requests, workspaceFetches: state.workspaceFetches };
}

await test("a failed current-user bootstrap hides protected content and retry completes the account load", async (t) => {
  const { requests, workspaceFetches } = await retryFailedAccountQuery(t, "user");
  assert.equal(requests.user, 2);
  assert.equal(requests.profile, 1);
  assert.equal(workspaceFetches, 1);
});

await test("retrying a failed profile does not refetch the healthy current-user query", async (t) => {
  const { requests, workspaceFetches } = await retryFailedAccountQuery(t, "profile");
  assert.equal(requests.user, 1);
  assert.equal(requests.profile, 2);
  assert.equal(workspaceFetches, 1);
});

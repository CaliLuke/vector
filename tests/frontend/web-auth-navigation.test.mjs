import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createRouterFixture, routerApi } from "./helpers/router.mjs";
import { React, act, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const state = { user: undefined, profile: undefined };
globalThis.vectorAuthNavigationTest = state;
const stubs = {
  "@/hooks/store/user": `export const useUser = () => ({ data: globalThis.vectorAuthNavigationTest.user, isLoading: false, fetchCurrentUser: async () => globalThis.vectorAuthNavigationTest.user });
    export const useUserProfile = () => ({data: globalThis.vectorAuthNavigationTest.profile});
    export const useUserSettings = () => ({data: {workspace: {last_workspace_slug: "acme"}}});`,
  "@/hooks/store/use-workspace": `export const useWorkspace = () => ({loader: false, workspaces: {acme: {slug: "acme"}}});`,
  "@/components/common/logo-spinner": `export const LogoSpinner = () => null;`,
  "@/helpers/authentication.helper": `export const EPageTypes = {PUBLIC:"PUBLIC",NON_AUTHENTICATED:"NON_AUTHENTICATED",SET_PASSWORD:"SET_PASSWORD",ONBOARDING:"ONBOARDING",AUTHENTICATED:"AUTHENTICATED"};`,
};
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (Object.hasOwn(stubs, specifier))
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(stubs[specifier])}` };
    return nextResolve(specifier, context);
  },
});
const { AuthenticationWrapper } = await import("../../apps/web/core/lib/wrappers/authentication-wrapper.tsx");
const { SWRConfig } = await import("swr");

async function renderAuth(t, { path, pageType, user, profile }) {
  state.user = user;
  state.profile = profile;
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
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
      React.createElement(SWRConfig, { value: { provider: () => new Map() } }, React.createElement(Harness))
    )
  );
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  return { environment, router: fixture.router };
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
  assert.ok(environment.container.querySelector("[data-protected]"));
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
  state.user = undefined;
  state.profile = undefined;
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
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
    React.createElement(
      SWRConfig,
      { value: { provider: () => new Map() } },
      React.createElement(RouterProvider, { router })
    )
  );
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 80));
  });
  assert.equal(router.state.location.pathname, "/");
  assert.equal(router.state.location.search.next_path, destination);
  assert.equal(environment.container.textContent, "Sign in");
});

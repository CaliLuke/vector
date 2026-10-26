import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { createRouterFixture, routerApi } from "./helpers/router.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.workspaceSettingsTransitionReact = React;
globalThis.workspaceSettingsTransitionRole = undefined;
const stubs = {
  "@/components/work-item-types/preview-provider": `const React=globalThis.workspaceSettingsTransitionReact;export const WorkItemTypesPreviewProvider=({children})=>children;`,
  "@/components/auth-screens/not-authorized-view": `const React=globalThis.workspaceSettingsTransitionReact;export const NotAuthorizedView=()=>React.createElement("div",{role:"alert","data-denied":true},"Not authorized");`,
  "@/components/settings/mobile/nav": `export const SettingsMobileNav=()=>null;`,
  "@/components/settings/workspace/sidebar": `const React=globalThis.workspaceSettingsTransitionReact;export const WorkspaceSettingsSidebarRoot=()=>React.createElement("div",{"data-settings-visible":true},"Settings");`,
  "@/hooks/use-permissions": `export const useWorkspacePermissions=()=>({workspaceRole:globalThis.workspaceSettingsTransitionRole,isPending:false,isError:false});`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) =>
    Object.hasOwn(stubs, specifier)
      ? resolvePackageStub(specifier, stubs, context, nextResolve)
      : nextResolve(specifier, context),
});

const { Route } = await import("../../apps/web/app/(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx");

const layoutFile = "(all)/[workspaceSlug]/(settings)/settings/(workspace)/layout.tsx";
const homeFile = "(all)/[workspaceSlug]/(projects)/page.tsx";

async function fixture(t, { path, role, deferHome = false }) {
  const environment = await createReactDomTestEnvironment();
  const home = Promise.withResolvers();
  const homeStarted = Promise.withResolvers();
  t.after(async () => {
    home.resolve();
    await environment.render(null);
    await environment.unmount();
  });
  globalThis.workspaceSettingsTransitionRole = role;

  const modules = {
    [layoutFile]: {
      update: (options) => routerApi.createRoute({ ...options, component: Route.options.component }),
    },
  };
  if (deferHome) {
    modules[homeFile] = {
      update: (options) =>
        routerApi.createRoute({
          ...options,
          loader: async () => {
            homeStarted.resolve();
            await home.promise;
          },
          component: () => React.createElement("main", null, "Home"),
        }),
    };
  }
  const { router } = createRouterFixture(path, { modules });
  await router.load();
  await environment.render(React.createElement(routerApi.RouterProvider, { router }));
  return { environment, router, home, homeStarted };
}

await test("leaving workspace settings does not authorize against the pending destination URL", async (t) => {
  const state = await fixture(t, {
    path: "/acme/settings/",
    role: 20,
    deferHome: true,
  });
  assert.ok(state.environment.container.querySelector("[data-settings-visible]"));
  assert.equal(state.environment.container.querySelector("[data-denied]"), null);

  let navigation = Promise.resolve();
  await act(async () => {
    navigation = state.router.navigate({ to: "/acme/" });
    await state.homeStarted.promise;
  });
  assert.equal(state.router.state.location.pathname, "/acme/");
  assert.ok(state.environment.container.querySelector("[data-settings-visible]"));
  assert.equal(state.environment.container.querySelector("[data-denied]"), null);

  await act(async () => state.home.resolve());
  await navigation;
  assert.equal(state.environment.container.querySelector("main")?.textContent, "Home");
  assert.equal(state.environment.container.querySelector("[data-denied]"), null);
});

await test("workspace settings still deny a member on an admin-only matched route", async (t) => {
  const state = await fixture(t, {
    path: "/acme/settings/webhooks/",
    role: 15,
  });
  assert.ok(state.environment.container.querySelector("[data-denied]"), state.environment.container.innerHTML);
  assert.equal(state.environment.container.querySelector("main"), null);
});

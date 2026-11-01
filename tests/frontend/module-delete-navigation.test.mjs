import * as ContractEnums from "../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { createRouterFixture } from "./helpers/router.mjs";
import { createPermissionQueryClient } from "./helpers/permission-query-fixture.mjs";
import { settleQuery } from "./helpers/query-router.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
registerWebModuleHooks();
globalThis.moduleDeleteReact = React;
globalThis.moduleActionToasts = [];
const h = "const React=globalThis.moduleDeleteReact;";
const stubs = {
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/propel/toast":
    "export const TOAST_TYPE={SUCCESS:'success',ERROR:'error'};export const setToast=value=>globalThis.moduleActionToasts.push(value);",
  "@/components/core/page-title": "export const PageHead=()=>null;",
  "@/components/common/empty-state": h + "export const EmptyState=({title})=>React.createElement('output',null,title);",
  "@/components/modules": `export {DeleteModuleModal} from ${JSON.stringify(new URL("../../apps/web/core/components/modules/delete-module-modal.tsx", import.meta.url).href)};export const ModuleAnalyticsSidebar=()=>null;export const ArchiveModuleModal=()=>null;export const CreateUpdateModuleModal=()=>null;`,
  "@/components/common/quick-actions-helper": `export {normalizeQuickActions, useQuickActionDialogs} from ${JSON.stringify(new URL("../../apps/web/core/components/common/quick-actions-helper.tsx", import.meta.url).href)};export const useModuleMenuItems=({handleDelete,handleRestore,moduleDetails,isEditingAllowed})=>!isEditingAllowed?[]:moduleDetails.archived_at?[{key:'restore-module',title:'Restore module',action:handleRestore}]:[{key:'delete-module',title:'Delete module',action:handleDelete}];`,
  "@/components/issues/issue-layouts/roots/module-layout-root":
    h +
    `
    import {useModuleDetails} from '@/hooks/use-modules';
    import {ModuleQuickActions} from '@/components/modules/quick-actions';
    export function ModuleLayoutRoot(){
      const {module}=useModuleDetails('acme','p1','m1');const ref=React.useRef(null);
      const [visible,setVisible]=React.useState(true);globalThis.hideModuleDeleteRow=()=>setVisible(false);
      return module&&visible?React.createElement('div',{ref,'data-testid':'module-row'},React.createElement(ModuleQuickActions,{module,moduleId:'m1',projectId:'p1',workspaceSlug:'acme',parentRef:ref})):null;
    }`,
};
registerHooks({
  resolve(specifier, context, next) {
    if (specifier.endsWith("?url")) return { shortCircuit: true, url: 'data:text/javascript,export default "asset";' };
    return resolvePackageStub(specifier, stubs, context, next);
  },
});
const routeFile = "(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/modules/(detail)/[moduleId]/page.tsx";
const { ModuleService } = await import("../../apps/web/core/services/module.service.ts");
const { ModuleArchiveService } = await import("../../apps/web/core/services/module_archive.service.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { UserService } = await import("../../apps/web/core/services/user.service.ts");
const { CycleModulePreferencesProvider } = await import("../../apps/web/core/providers/cycle-module-preferences.tsx");
const { moduleDetailOptions, projectModuleOptions } = await import("../../apps/web/core/queries/cycle-modules.ts");
const { estimateStatisticsRevision } = await import("../../apps/web/core/queries/parent-statistics.ts");
const { estimateKeys } = await import("../../apps/web/core/queries/estimates.ts");
const { projectKeys } = await import("../../apps/web/core/queries/projects.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const scope = {
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  workspaceRole: ContractEnums.EUserPermissions.MEMBER,
  membershipRevision: "revision-1",
  projectId: "p1",
  projectMembershipRole: ContractEnums.EUserPermissions.ADMIN,
  projectRole: ContractEnums.EUserPermissions.ADMIN,
};
const moduleRow = { id: "m1", project_id: "p1", workspace_id: "w1", name: "Module", archived_at: null };
const detail = moduleDetailOptions(scope, "m1", false, estimateStatisticsRevision(null, "p1", []));
async function fixture(t, { initial = moduleRow, read } = {}) {
  globalThis.moduleActionToasts = [];
  const environment = await createReactDomTestEnvironment();
  const { Route } = await import("../../apps/web/app/" + routeFile);
  const { client } = await createPermissionQueryClient(t, scope);
  client.setDefaultOptions({
    queries: { retry: false, gcTime: Infinity, staleTime: Infinity },
    mutations: { retry: false, gcTime: Infinity },
  });
  client.setQueryData(projectKeys.detail({ ...scope, projectRoles: [["p1", ContractEnums.EUserPermissions.ADMIN]] }), {
    id: "p1",
    workspace: "w1",
    estimate: null,
    name: "Project",
  });
  client.setQueryData(
    estimateKeys.list({ ...scope, kind: "project", membershipRole: ContractEnums.EUserPermissions.MEMBER }),
    []
  );
  client.setQueryData(projectModuleOptions(scope).queryKey, [moduleRow]);
  if (initial?.archived_at) client.setQueryData(projectModuleOptions(scope, true).queryKey, [initial]);
  if (initial !== undefined) client.setQueryData(detail.queryKey, initial);
  t.mock.method(ModuleService.prototype, "getModules", async () => [moduleRow]);
  t.mock.method(ModuleArchiveService.prototype, "getArchivedModules", async () => []);
  t.mock.method(ModuleService.prototype, "getModuleDetails", read ?? (async () => moduleRow));
  t.mock.method(UserService.prototype, "currentUser", async () => ({ id: "alice" }));
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => ({
    role: ContractEnums.EUserPermissions.MEMBER,
    is_active: true,
    updated_at: "revision-1",
  }));
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => ({ p1: 20 }));
  const component = Route.options.component;
  const router = createRouterFixture("/acme/projects/p1/modules/m1", {
    modules: { [routeFile]: { update: (options) => Route.update({ ...options, component }) } },
  });
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    await environment.unmount();
  });
  const render = async () => {
    await act(async () => router.router.load());
    await environment.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(CycleModulePreferencesProvider, null, React.createElement(router.Provider))
      )
    );
    await settleQuery();
  };
  return { environment, client, router: router.router, render };
}
function control(text) {
  return [...document.querySelectorAll("button")].find((button) => button.textContent.trim() === text);
}
await test("failed module restoration retains its route and archive row until an accepted retry", async (t) => {
  const requests = [];
  t.mock.method(ModuleArchiveService.prototype, "restoreModule", async (...args) => {
    requests.push(args);
    if (requests.length === 1) throw new Error("restore offline");
  });
  const view = await fixture(t, { initial: { ...moduleRow, archived_at: "2026-10-30T00:00:00Z" } });
  await view.render();
  const restore = async () => {
    await act(async () => view.environment.container.querySelector("[data-main-menu] button").click());
    assert.ok(control("Restore module"));
    await act(async () => control("Restore module").click());
    await settleQuery();
  };
  await restore();
  assert.equal(view.router.state.location.pathname, "/acme/projects/p1/modules/m1");
  assert.equal(view.client.getQueryData(projectModuleOptions(scope, true).queryKey)[0].id, "m1");
  assert.deepEqual(
    globalThis.moduleActionToasts.map((toast) => toast.type),
    ["error"]
  );
  await restore();
  assert.deepEqual(requests, [
    ["acme", "p1", "m1"],
    ["acme", "p1", "m1"],
  ]);
  assert.equal(view.router.state.location.pathname, "/acme/projects/p1/archives/modules");
  assert.deepEqual(view.client.getQueryData(projectModuleOptions(scope, true).queryKey), []);
  assert.deepEqual(
    globalThis.moduleActionToasts.map((toast) => toast.type),
    ["error", "success"]
  );
});

await test("accepted native deletion leaves a null receipt and stable detail owner navigates after row/modal unmount", async (t) => {
  const pending = Promise.withResolvers();
  let calls = 0;
  t.mock.method(ModuleService.prototype, "deleteModule", async (workspace, project, id) => {
    calls++;
    assert.deepEqual([workspace, project, id], ["acme", "p1", "m1"]);
    await pending.promise;
  });
  const view = await fixture(t);
  await view.render();
  const trigger = view.environment.container.querySelector("[data-main-menu] button");
  assert.ok(trigger);
  await act(async () => trigger.click());
  await act(async () => control("Delete module").click());
  assert.ok(control("Delete"), "the actual module delete dialog is open");
  await act(async () => control("Delete").click());
  assert.equal(calls, 1);
  assert.ok(
    view.environment.container.querySelector('[data-testid="module-row"]'),
    "pending transport keeps the canonical row"
  );
  await act(async () => globalThis.hideModuleDeleteRow());
  assert.equal(
    view.environment.container.querySelector('[data-testid="module-row"]'),
    null,
    "a list projection can unmount the row while transport remains pending"
  );
  assert.equal(control("Delete"), undefined, "the pending dialog has unmounted before the native write settles");
  assert.equal(view.router.state.location.pathname, "/acme/projects/p1/modules/m1");
  await act(async () => pending.resolve());
  await settleQuery();
  assert.equal(
    view.client.getQueryData(detail.queryKey),
    null,
    "accepted mutation publishes the exact native removal receipt"
  );
  assert.equal(view.router.state.location.pathname, "/acme/projects/p1/modules");
  assert.equal(view.environment.container.querySelector('[data-testid="module-row"]'), null);
  assert.equal(control("Delete"), undefined, "the deleted row dialog has unmounted");
});
await test("missing, failed and permission-denied native details never imply accepted removal navigation", async (t) => {
  const view = await fixture(t);
  view.client.removeQueries({ queryKey: detail.queryKey, exact: true });
  const pending = Promise.withResolvers();
  t.mock.method(ModuleService.prototype, "getModuleDetails", () => pending.promise);
  await view.render();
  assert.equal(view.router.state.location.pathname, "/acme/projects/p1/modules/m1");
  await act(async () => pending.reject(Object.assign(new Error("missing"), { response: { status: 404 } })));
  await settleQuery();
  assert.ok(view.environment.container.querySelector('[role="alert"]'));
  assert.equal(view.router.state.location.pathname, "/acme/projects/p1/modules/m1");
  await act(async () =>
    view.client
      .getQueryCache()
      .find({
        queryKey: permissionKeys.workspaceProjectRoles(
          "alice",
          "acme",
          ContractEnums.EUserPermissions.MEMBER,
          "revision-1"
        ),
        exact: true,
      })
      .setState({ status: "error", error: { response: { status: 403 } } })
  );
  await settleQuery();
  assert.ok(view.environment.container.querySelector('[role="alert"]'));
  assert.equal(view.router.state.location.pathname, "/acme/projects/p1/modules/m1");
});

import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import path from "node:path";
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const requireFromWeb = createRequire(path.resolve("apps/web/package.json"));
const reactQueryModuleUrl = pathToFileURL(requireFromWeb.resolve("@tanstack/react-query")).href;
const { createReactDomTestEnvironment } = await fromRoot("tests/frontend/helpers/react-dom.mjs");
const { React, act, webRequire } = await fromRoot("tests/frontend/helpers/react.mjs");
const { routerApi } = await fromRoot("tests/frontend/helpers/router.mjs");
const { registerWebModuleHooks } = await fromRoot("tests/frontend/helpers/web-module-hooks.mjs");
const { resolvePackageStub } = await fromRoot("tests/frontend/helpers/module-stubs.mjs");
const { seedPermissionQueries } = await fromRoot("tests/frontend/helpers/permission-query-fixture.mjs");

const state = { toast: undefined, commands: [], cycleCalls: 0, moduleCalls: 0 };
globalThis.vectorPowerKCommandTest = state;
const stubs = {
  "@/services/user.service":
    "export class UserService { currentUser() { throw new Error('unexpected identity request'); } } export default new UserService();",
  "@/services/workspace.service":
    "export class WorkspaceService { workspaceMemberMe() { throw new Error('unexpected membership request'); } getWorkspaceUserProjectsRole() { throw new Error('unexpected roles request'); } }",
  "@/hooks/use-command-palette": `export const useCommandPalette = () => ({toggleDeleteIssueModal: () => {}});`,
  "@/hooks/use-projects": `export const useWorkspaceProjectDetails = () => ({projects: [{id: "project-id", identifier: "PROJ", estimate: "estimate-id", cycle_view: true, module_view: true}]});`,
  "@/hooks/use-work-item-detail": `
    const item = {id: "issue-id", project_id: "project-id", sequence_id: 1, name: "Test", state_id: "state-id", priority: "medium", estimate_point: null, assignee_ids: [], label_ids: [], cycle_id: null, module_ids: [], is_epic: false};
    export const useWorkItemByIdentifier = (workspaceSlug, _projectIdentifier, _sequenceId, projectId) => ({issue: item, scope: {workspaceSlug, projectId, issueId: item.id}});
  `,
  "@/hooks/use-issue-list-mutation-owner": `export const useIssueListMutationOwner=()=>async(workspaceSlug,projectId)=>({workspaceSlug,projectId});`,
  "@/queries/issue-list-mutations": `export const issueListMutation=async(_client,_owner,mutation)=>{if(mutation.kind.startsWith("cycle"))globalThis.vectorPowerKCommandTest.cycleCalls++;if(mutation.kind==="module-change")globalThis.vectorPowerKCommandTest.moduleCalls++;throw new Error("mutation rejected")};export const patchIssueMutation=async()=>{throw new Error("patch rejected")};`,
  "@/queries/issue-details": `export const captureIssueDetailReconciliation=()=>({refresh:async()=>{},reconcile:async()=>{}});`,
  "@plane/i18n": `export const useTranslation = () => ({t: (key) => key});`,
  "@plane/propel/toast": `export const TOAST_TYPE = {ERROR: "error", SUCCESS: "success"}; export const setToast = (toast) => { globalThis.vectorPowerKCommandTest.toast = toast; };`,
};
registerWebModuleHooks();
registerHooks({
  resolve: (specifier, context, nextResolve) => {
    if (specifier === "@tanstack/react-query") return { shortCircuit: true, url: reactQueryModuleUrl };
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});
const { usePowerKWorkItemContextBasedCommands } = await fromRoot(
  "apps/web/core/components/power-k/ui/pages/context-based/work-item/commands.ts"
);
const { QueryClient, QueryClientProvider } = webRequire("@tanstack/react-query");

async function runRejectedCommand(t, commandId, selectedValue, callCounter) {
  state.toast = undefined;
  state.commands = [];
  state[callCounter] = 0;
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const { currentUserOptions } = await fromRoot("apps/web/core/queries/identity.ts");
  client.setQueryData(currentUserOptions().queryKey, { id: "alice" });
  await seedPermissionQueries(client, {
    userId: "alice",
    workspaceSlug: "acme",
    projectRoles: { "project-id": 20 },
  });
  const { createMemoryHistory, createRootRoute, createRoute, createRouter, RouterProvider } = routerApi;
  function Harness() {
    const commands = usePowerKWorkItemContextBasedCommands();
    React.useEffect(() => {
      state.commands = commands;
    }, [commands]);
    return null;
  }
  const root = createRootRoute();
  const route = createRoute({
    getParentRoute: () => root,
    path: "/$workspaceSlug/projects/$projectId/issues/$workItem",
    component: Harness,
  });
  const router = createRouter({
    routeTree: root.addChildren([route]),
    history: createMemoryHistory({ initialEntries: ["/acme/projects/project-id/issues/PROJ-1"] }),
    isServer: false,
    defaultPendingMinMs: 0,
  });
  t.after(async () => {
    await environment.render(null);
    await client.cancelQueries();
    client.clear();
    await environment.unmount();
  });
  await act(async () => router.load());
  await environment.render(
    React.createElement(QueryClientProvider, { client }, React.createElement(RouterProvider, { router }))
  );
  const command = state.commands.find((item) => item.id === commandId);
  assert.ok(command, `Expected command ${commandId}`);
  await act(async () => command.onSelect(selectedValue, {}));
  assert.equal(state[callCounter], 1);
  assert.equal(state.toast?.type, "error");
  assert.match(state.toast?.message ?? "", /could not be updated/);
}

await test("cycle selection reports a rejected update", async (t) => {
  await runRejectedCommand(t, "add_work_item_to_cycle", { id: "cycle-id" }, "cycleCalls");
});

await test("module selection reports a rejected update", async (t) => {
  await runRejectedCommand(t, "add_work_item_to_modules", { id: "module-id" }, "moduleCalls");
});

import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedWorkspaceQuery } from "./helpers/workspace-query-fixture.mjs";

registerWebModuleHooks();
globalThis.workspaceAccessReact = React;
globalThis.workspaceAccessReads = [];
const stubs = {
  "@/app/assets/workspace/workspace-not-available.png?url": `export default "workspace-not-available.png";`,
  "@/components/common/logo-spinner": `export const LogoSpinner=()=>globalThis.workspaceAccessReact.createElement("span",{role:"status"},"Loading");`,
  "@/hooks/use-platform-os": `export const usePlatformOS=()=>({isMobile:false});`,
  "@/hooks/store/use-project": `export const useProject=()=>({fetchPartialProjects:async()=>{globalThis.workspaceAccessReads.push("projects");return[]}});`,
  "@/hooks/use-project-states":
    "export const useWorkspaceStates=()=>({states:[],workspaceStates:[],getProjectStates:()=>[],getStateById:()=>undefined,projection:undefined,accessPending:false,accessError:false,accessDenied:false,retryAccess:async()=>{},refetch:async()=>({})}); export const useProjectStates=()=>({states:[],projectStates:[],getStateById:()=>undefined,getProjectStateIds:()=>[],getProjectDefaultStateId:()=>undefined,getProjectIntakeStateIds:()=>[],getIntakeStateById:()=>undefined,accessPending:false,accessError:false,accessDenied:false,retryAccess:async()=>{},refetch:async()=>({})});",
  "@plane/propel/button": `const React=globalThis.workspaceAccessReact;export const Button=({variant,children,...props})=>React.createElement("button",props,children);export const getButtonStyling=()=>"";`,
  "@plane/propel/icons": `export const PlaneLogo=()=>null;`,
  "@plane/propel/tooltip": `export const Tooltip=({children})=>children;`,
  "@plane/propel/toast": `export const TOAST_TYPE={ERROR:"error"};export const setToast=()=>{};`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

const { WorkspaceAuthWrapper } = await import("../../apps/web/core/layouts/auth-layout/workspace-wrapper.tsx");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { workspaceKeys, workspacesOptions } = await import("../../apps/web/core/queries/workspaces.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");
const activeMembership = { is_active: true, role: EUserWorkspaceRoles.MEMBER, updated_at: "revision-1" };
function mockAccess(t) {
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => activeMembership);
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => ({}));
}
const content = React.createElement(WorkspaceAuthWrapper, null, React.createElement("main", null, "Workspace content"));

async function expectAccess(t, assertState) {
  // Membership and roles load in sequence. Wait for the visible result while
  // keeping their asynchronous React updates inside the act scope.
  await act(async () => t.waitFor(assertState));
}

async function assertInvalidMembershipIsDenied(t, membership) {
  const fixture = await createQueryRouterFixture(t, "/acme/");
  await seedWorkspaceQuery(fixture.client, { id: "workspace", slug: "acme" });
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => membership);
  const roles = t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => ({}));
  globalThis.workspaceAccessReads = [];

  await fixture.render(content);
  await expectAccess(t, () => assert.match(fixture.environment.container.textContent, /Not Authorized/));
  assert.equal(roles.mock.callCount(), 0);
  assert.deepEqual(globalThis.workspaceAccessReads, []);
}

await test("an initial catalogue failure offers retry and recovers into the workspace", async (t) => {
  mockAccess(t);
  const fixture = await createQueryRouterFixture(t, "/acme/");
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "member" });
  let unavailable = true;
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => {
    if (unavailable) throw new Error("offline");
    return [{ id: "workspace", slug: "acme" }];
  });
  await fixture.render(content);
  const { container } = fixture.environment;
  await expectAccess(t, () =>
    assert.match(container.querySelector('[role="alert"]')?.textContent ?? "", /Unable to load your workspaces/)
  );
  assert.equal(container.querySelector("main"), null);
  unavailable = false;
  await act(async () => container.querySelector("button").click());
  await expectAccess(t, () => assert.equal(container.querySelector("main")?.textContent, "Workspace content"));
  assert.equal(container.querySelector('[role="alert"]'), null);
});

await test("a refreshed catalogue revokes workspace rendering even with cached permission data", async (t) => {
  mockAccess(t);
  const roles = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", () => roles.promise);
  const fixture = await createQueryRouterFixture(t, "/acme/");
  await seedWorkspaceQuery(fixture.client, { id: "workspace", slug: "acme" });
  await fixture.render(content);
  const { container } = fixture.environment;
  assert.equal(container.querySelector("main"), null);
  assert.match(container.querySelector('[role="status"]').textContent, /Loading/);
  await act(async () => roles.resolve({}));
  await expectAccess(t, () => assert.equal(container.querySelector("main")?.textContent, "Workspace content"));
  t.mock.method(WorkspaceService.prototype, "userWorkspaces", async () => []);
  await act(async () => fixture.client.fetchQuery({ ...workspacesOptions("workspace-query-test-user"), staleTime: 0 }));
  await expectAccess(t, () => assert.match(container.textContent, /Workspace not found/));
  assert.deepEqual(fixture.client.getQueryData(workspaceKeys.list("workspace-query-test-user")), []);
  assert.equal(container.querySelector("main"), null);
  assert.match(container.textContent, /Workspace not found/);
});

await test("pending membership waits, then inactive membership denies access without reading project roles", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/");
  await seedWorkspaceQuery(fixture.client, { id: "workspace", slug: "acme" });
  let resolveMembership;
  const pending = new Promise((resolve) => {
    resolveMembership = resolve;
  });
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", () => pending);
  const roles = t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => ({}));
  await fixture.render(content);
  const { container } = fixture.environment;
  assert.match(container.querySelector('[role="status"]').textContent, /Loading/);
  assert.doesNotMatch(container.textContent, /Not Authorized/);
  await act(async () => resolveMembership({ is_active: false, role: null }));
  await expectAccess(t, () => assert.match(container.textContent, /Not Authorized/));
  assert.equal(container.querySelector("main"), null);
  assert.match(container.textContent, /Not Authorized/);
  assert.equal(roles.mock.callCount(), 0);
});

await test("active membership without a valid role denies access without loading workspace entities", async (t) => {
  await assertInvalidMembershipIsDenied(t, { is_active: true, role: null, updated_at: "revision-1" });
});

await test("active membership with an unknown role denies access without loading workspace entities", async (t) => {
  await assertInvalidMembershipIsDenied(t, { is_active: true, role: 999, updated_at: "revision-1" });
});

await test("failed membership read offers retry instead of an authorization denial", async (t) => {
  const fixture = await createQueryRouterFixture(t, "/acme/");
  await seedWorkspaceQuery(fixture.client, { id: "workspace", slug: "acme" });
  let offline = true;
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => {
    if (offline) throw new Error("offline");
    return activeMembership;
  });
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => ({}));
  await fixture.render(content);
  const { container } = fixture.environment;
  await expectAccess(t, () =>
    assert.match(container.querySelector('[role="alert"]')?.textContent ?? "", /Unable to load your workspace access/)
  );
  assert.doesNotMatch(container.textContent, /Not Authorized/);
  offline = false;
  await act(async () => container.querySelector("button").click());
  await expectAccess(t, () => assert.equal(container.querySelector("main")?.textContent, "Workspace content"));
});

await test("a failed role refresh hides cached workspace content and retries only that read", async (t) => {
  mockAccess(t);
  const fixture = await createQueryRouterFixture(t, "/acme/");
  await seedWorkspaceQuery(fixture.client, { id: "workspace", slug: "acme" });
  await fixture.render(content);
  const { container } = fixture.environment;
  await expectAccess(t, () => assert.equal(container.querySelector("main")?.textContent, "Workspace content"));
  let offline = true;
  const roles = t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", async () => {
    if (offline) throw new Error("offline");
    return {};
  });
  const key = permissionKeys.workspaceProjectRoles(
    "workspace-query-test-user",
    "acme",
    activeMembership.role,
    activeMembership.updated_at
  );
  await act(async () => fixture.client.invalidateQueries({ queryKey: key, exact: true }));
  await expectAccess(t, () =>
    assert.match(container.querySelector('[role="alert"]')?.textContent ?? "", /Unable to load your workspace access/)
  );
  assert.equal(container.querySelector("main"), null);
  assert.match(container.querySelector('[role="alert"]').textContent, /Unable to load your workspace access/);
  offline = false;
  await act(async () => container.querySelector("button").click());
  await expectAccess(t, () => assert.equal(container.querySelector("main")?.textContent, "Workspace content"));
  assert.equal(roles.mock.callCount(), 2);
});

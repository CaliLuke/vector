import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";
import test from "node:test";

import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";

globalThis.analyticsStateFixture = {
  userId: "account-a",
  generation: 1,
  role: 15,
  active: true,
  revision: "rev-a",
  projectRoles: { "project-a": 15, "project-b": 15 },
};

const fixtureExpression = `globalThis.analyticsStateFixture`;
registerWebTestModules({
  "@/hooks/use-current-user": `export const useCurrentUser=()=>({data:{id:${fixtureExpression}.userId},isSuccess:true,isError:false,refetch:async()=>({})})`,
  "@/hooks/use-identity-generation": `export const useIdentityGeneration=()=>${fixtureExpression}.generation`,
  "@/hooks/use-permissions": `
        export const useWorkspacePermissions=(slug)=>{
          const f=${fixtureExpression};
          const data={is_active:f.active,role:f.role,updated_at:f.revision};
          return {workspaceRole:data.is_active?f.role:undefined,membershipQuery:{isSuccess:true,isError:false,data,refetch:async()=>({})}};
        };
        export const useWorkspaceProjectRoles=(slug)=>{
          const f=${fixtureExpression};
          return {roles:f.active?f.projectRoles:undefined,rolesQuery:{isSuccess:f.active,isError:false,refetch:async()=>({})}};
        };
      `,
});

const { AnalyticsStateProvider, useAnalyticsState } =
  await import("../../apps/web/core/providers/analytics-state-provider.tsx");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { QueryClient, QueryClientProvider } = await import("@tanstack/react-query");

function StateDisplay() {
  const state = useAnalyticsState();
  return React.createElement(
    "section",
    null,
    React.createElement("output", { "data-testid": "projects" }, state.selectedProjects.join(",")),
    React.createElement("output", { "data-testid": "cycle" }, state.selectedCycle),
    React.createElement("output", { "data-testid": "module" }, state.selectedModule),
    React.createElement("output", { "data-testid": "peek" }, String(state.isPeekView)),
    React.createElement("output", { "data-testid": "epic" }, String(state.isEpic)),
    React.createElement("output", { "data-testid": "filters" }, JSON.stringify(state.requestFilters)),
    React.createElement("output", { "data-testid": "ready" }, String(state.scope.isReady)),
    React.createElement(
      "output",
      { "data-testid": "workspace-access" },
      String(state.scope.hasWorkspaceAnalyticsAccess)
    ),
    React.createElement("output", { "data-testid": "project-access" }, String(state.scope.hasProjectAnalyticsAccess)),
    React.createElement("output", { "data-testid": "chart-access" }, String(state.scope.hasProjectChartAccess)),
    React.createElement("button", { onClick: () => state.updateSelectedProjects(["draft-project"]) }, "Change projects")
  );
}

function wrap(client, workspaceSlug, props = {}) {
  return React.createElement(
    QueryClientProvider,
    { client },
    React.createElement(AnalyticsStateProvider, { workspaceSlug, ...props }, React.createElement(StateDisplay))
  );
}

function primePermissions(client, workspaceSlug, projectIds = []) {
  const fixture = globalThis.analyticsStateFixture;
  const membership = { is_active: fixture.active, role: fixture.role, updated_at: fixture.revision };
  client.setQueryData(["identity", "current-user"], { id: fixture.userId });
  client.setQueryData(permissionKeys.workspaceMembership(fixture.userId, workspaceSlug), membership);
  client.setQueryData(
    permissionKeys.workspaceProjectRoles(fixture.userId, workspaceSlug, fixture.role, fixture.revision),
    Object.fromEntries(projectIds.map((id) => [id, fixture.projectRoles[id]]))
  );
}

function output(environment, name) {
  return environment.container.querySelector(`[data-testid="${name}"]`)?.textContent;
}

void test("dashboard drafts reset on workspace, account, and session changes", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = new QueryClient();
  primePermissions(client, "alpha");
  await environment.render(wrap(client, "alpha"));
  assert.equal(output(environment, "projects"), "");
  await React.act(async () => environment.container.querySelector("button").click());
  assert.equal(output(environment, "projects"), "draft-project");

  primePermissions(client, "beta");
  await environment.render(wrap(client, "beta"));
  assert.equal(output(environment, "projects"), "");

  globalThis.analyticsStateFixture.userId = "account-b";
  primePermissions(client, "beta");
  await environment.render(wrap(client, "beta"));
  assert.equal(output(environment, "projects"), "");

  globalThis.analyticsStateFixture.generation += 1;
  await environment.render(wrap(client, "beta"));
  assert.equal(output(environment, "projects"), "");
  client.clear();
});

void test("each modal starts from explicit selections and close/reopen discards its draft", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = new QueryClient();
  globalThis.analyticsStateFixture.userId = "account-a";
  globalThis.analyticsStateFixture.projectRoles = { "project-a": 15, "project-b": 15 };
  primePermissions(client, "alpha", ["project-a", "project-b"]);
  const props = {
    projectId: "project-a",
    cycleId: "cycle-a",
    moduleId: "module-a",
    isPeekView: true,
    isEpic: true,
    isActive: true,
    instanceKey: "modal:open:project-a:cycle-a:module-a:true",
  };
  await environment.render(wrap(client, "alpha", props));
  assert.equal(output(environment, "projects"), "project-a");
  assert.equal(output(environment, "cycle"), "cycle-a");
  assert.equal(output(environment, "module"), "module-a");
  assert.equal(output(environment, "peek"), "true");
  assert.equal(output(environment, "ready"), "true");
  assert.equal(output(environment, "project-access"), "true");
  assert.equal(output(environment, "chart-access"), "true");
  assert.equal(output(environment, "epic"), "true");
  assert.equal(
    output(environment, "filters"),
    JSON.stringify({ project_ids: "project-a", cycle_id: "cycle-a", module_id: "module-a", epic: true })
  );

  await React.act(async () => environment.container.querySelector("button").click());
  assert.equal(output(environment, "projects"), "draft-project");
  await environment.render(wrap(client, "alpha", { ...props, isActive: false, instanceKey: "modal:closed" }));
  await environment.render(wrap(client, "alpha", { ...props, instanceKey: "modal:reopened" }));
  assert.equal(output(environment, "projects"), "project-a");

  await environment.render(
    wrap(client, "alpha", {
      ...props,
      projectId: "project-b",
      cycleId: "cycle-b",
      moduleId: "module-b",
      instanceKey: "modal:open:project-b:cycle-b:module-b:true",
    })
  );
  assert.equal(output(environment, "projects"), "project-b");
  assert.equal(output(environment, "cycle"), "cycle-b");
  assert.equal(output(environment, "module"), "module-b");
  client.clear();
});

void test("completed project-role lookup with no project membership is denial, not endless loading", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = new QueryClient();
  globalThis.analyticsStateFixture.projectRoles = {};
  primePermissions(client, "alpha", []);
  await environment.render(
    wrap(client, "alpha", {
      projectId: "unjoined-project",
      isPeekView: true,
      isActive: true,
      instanceKey: "unjoined-project",
    })
  );
  assert.equal(output(environment, "peek"), "true");
  assert.equal(output(environment, "ready"), "true");
  assert.equal(output(environment, "project-access"), "false");
  assert.equal(output(environment, "chart-access"), "false");
  assert.equal(environment.container.querySelector('[role="status"]'), null);
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
  client.clear();
  globalThis.analyticsStateFixture.projectRoles = { "project-a": 15, "project-b": 15 };
});

void test("project guest keeps chart access while member-only analytics access is denied", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = new QueryClient();
  globalThis.analyticsStateFixture.role = 5;
  globalThis.analyticsStateFixture.revision = "guest-revision";
  globalThis.analyticsStateFixture.projectRoles = { "guest-project": 5 };
  primePermissions(client, "alpha", ["guest-project"]);
  await environment.render(
    wrap(client, "alpha", {
      projectId: "guest-project",
      isPeekView: true,
      isActive: true,
      instanceKey: "guest-project-modal",
    })
  );
  assert.equal(output(environment, "ready"), "true");
  assert.equal(output(environment, "project-access"), "false");
  assert.equal(output(environment, "chart-access"), "true");
  client.clear();
  globalThis.analyticsStateFixture.role = 15;
  globalThis.analyticsStateFixture.revision = "rev-a";
  globalThis.analyticsStateFixture.projectRoles = { "project-a": 15, "project-b": 15 };
});

void test("inactive but successfully loaded workspace membership resolves to denied", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = new QueryClient();
  globalThis.analyticsStateFixture.active = false;
  primePermissions(client, "alpha");
  await environment.render(wrap(client, "alpha"));
  assert.equal(output(environment, "ready"), "true");
  assert.equal(output(environment, "workspace-access"), "false");
  assert.equal(output(environment, "projects"), "");
  assert.equal(environment.container.querySelector('[role="status"]'), null);
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
  client.clear();
  globalThis.analyticsStateFixture.active = true;
});

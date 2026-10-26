import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { webRequire } from "./helpers/react.mjs";

registerWebModuleHooks();

const { QueryClient, MutationObserver } = webRequire("@tanstack/react-query");
const { EUserProjectRoles, EUserWorkspaceRoles } = webRequire("@plane/types");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { projectDetailOptions, projectKeys, projectMutationOptions } =
  await import("../../apps/web/core/queries/projects.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");

const permission = {
  accountId: "alice",
  workspaceSlug: "acme",
  workspaceRole: EUserWorkspaceRoles.MEMBER,
  membershipRevision: "revision-1",
  projectRoles: [["project-1", EUserProjectRoles.ADMIN]],
  projectId: "project-1",
  projectMembershipRole: EUserProjectRoles.ADMIN,
  projectRole: EUserProjectRoles.ADMIN,
};

async function createClient(t) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } },
  });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: permission.accountId });
  await seedPermissionQueries(client, {
    userId: permission.accountId,
    workspaceSlug: permission.workspaceSlug,
    workspaceRole: permission.workspaceRole,
    revision: permission.membershipRevision,
    projectRoles: { "project-1": EUserProjectRoles.ADMIN },
  });
  return client;
}

function scope() {
  return { ...permission, sessionGeneration: 0 };
}

function project(overrides = {}) {
  return { id: "project-1", workspace: "workspace-1", identifier: "VEC", name: "Vector", ...overrides };
}

await test("a detail read rejects a response after its captured role query is replaced", async (t) => {
  const client = await createClient(t);
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(ProjectService.prototype, "getProject", (_workspaceSlug, _projectId, signal) => {
    assert.ok(signal instanceof AbortSignal);
    started.resolve();
    return response.promise;
  });

  const pending = client.fetchQuery({ ...projectDetailOptions(client, scope()), staleTime: 0 });
  await started.promise;
  const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
  const rolesKey = permissionKeys.workspaceProjectRoles(
    permission.accountId,
    permission.workspaceSlug,
    permission.workspaceRole,
    permission.membershipRevision
  );
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, { "project-1": EUserProjectRoles.ADMIN });
  response.resolve(project());

  await assert.rejects(pending, /access changed while loading/);
  assert.equal(client.getQueryData(projectKeys.detail(scope())), undefined);
});

await test("an accepted update does not replace a newer detail query owner", async (t) => {
  const client = await createClient(t);
  const detailKey = projectKeys.detail(scope());
  const original = project({ name: "Original" });
  client.setQueryData(detailKey, original);
  const partialKey = projectKeys.workspace(scope(), "partial");
  const fullKey = projectKeys.workspace(scope(), "full");
  client.setQueryData(partialKey, [original]);
  client.setQueryData(fullKey, [original]);

  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(ProjectService.prototype, "updateProject", (_workspaceSlug, _projectId, data) => {
    started.resolve();
    return response.promise.then(() => project(data));
  });
  const mutation = new MutationObserver(client, projectMutationOptions(client, scope()));
  const pending = mutation.mutate({ type: "update", projectId: "project-1", data: { name: "Accepted" } });
  await started.promise;

  const replacement = project({ name: "New owner" });
  client.removeQueries({ queryKey: detailKey, exact: true });
  client.setQueryData(detailKey, replacement);
  response.resolve();
  await pending;

  assert.deepEqual(client.getQueryData(detailKey), replacement);
  assert.deepEqual(client.getQueryData(partialKey), [original]);
  assert.deepEqual(client.getQueryData(fullKey), [original]);
});

await test("project-list drafts reset on workspace changes while each workspace keeps its sort order", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const { ProjectListFiltersProvider, useProjectListFilters } =
    await import("../../apps/web/core/hooks/use-project-list-filters.tsx");
  t.after(async () => environment.unmount());

  function Probe() {
    const { displayFilters, searchQuery, updateDisplayFilters, updateSearchQuery } = useProjectListFilters();
    return React.createElement(
      "div",
      null,
      React.createElement("output", { "data-testid": "project-filters" }, JSON.stringify(displayFilters)),
      React.createElement("output", { "data-testid": "project-search" }, searchQuery),
      React.createElement("button", { onClick: () => updateDisplayFilters({ my_projects: true, order_by: "name" }) }),
      React.createElement("button", { onClick: () => updateSearchQuery("private search") })
    );
  }

  const render = (workspaceSlug) =>
    environment.render(React.createElement(ProjectListFiltersProvider, { workspaceSlug }, React.createElement(Probe)));
  await render("alpha");
  const buttons = environment.container.querySelectorAll("button");
  await act(async () => {
    buttons[0].dispatchEvent(new MouseEvent("click", { bubbles: true }));
    buttons[1].dispatchEvent(new MouseEvent("click", { bubbles: true }));
  });
  assert.deepEqual(JSON.parse(environment.container.querySelector('[data-testid="project-filters"]').textContent), {
    order_by: "name",
    my_projects: true,
  });
  assert.equal(environment.container.querySelector('[data-testid="project-search"]').textContent, "private search");

  await render("beta");
  assert.equal(environment.container.querySelector('[data-testid="project-search"]').textContent, "");
  await render("alpha");
  assert.deepEqual(JSON.parse(environment.container.querySelector('[data-testid="project-filters"]').textContent), {
    order_by: "name",
  });
  assert.equal(environment.container.querySelector('[data-testid="project-search"]').textContent, "");
});

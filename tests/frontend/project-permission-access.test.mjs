import { projectWrapperMetadataStubs } from "./helpers/project-wrapper-metadata-stubs.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";

import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture, settleQuery } from "./helpers/query-router.mjs";

import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

globalThis.projectPermissionAccessHarness = {
  fetchProjectDetails: async () => ({ id: "project-1" }),
};
const stubs = {
  "@/app/assets/auth/project-not-authorized.svg?url": `export default "project-not-authorized.svg";`,
  "@/app/assets/auth/unauthorized.svg?url": `export default "unauthorized.svg";`,
  "@/app/assets/auth/workspace-not-authorized.svg?url": `export default "workspace-not-authorized.svg";`,
  ...projectWrapperMetadataStubs,
};
registerWebTestModules(stubs);

const { ProjectAuthWrapper } = await import("../../apps/web/core/layouts/auth-layout/project-wrapper.tsx");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { UserService } = await import("../../apps/web/core/services/user.service.ts");
const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { projectKeys } = await import("../../apps/web/core/queries/projects.ts");
const { EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");

const activeMember = { is_active: true, role: EUserWorkspaceRoles.MEMBER, updated_at: "revision-1" };
const activeAdmin = { is_active: true, role: EUserWorkspaceRoles.ADMIN, updated_at: "revision-1" };
const content = React.createElement(
  ProjectAuthWrapper,
  { workspaceSlug: "acme", projectId: "project-1" },
  React.createElement("main", null, "Project content")
);

function statusError(status) {
  return Object.assign(new Error(`HTTP ${status}`), { status });
}

function setHarness(overrides = {}) {
  globalThis.projectPermissionAccessHarness = {
    fetchProjectDetails: async () => ({ id: "project-1" }),
    ...overrides,
  };
}

function mockPermissionServices(t, { membership = async () => activeMember, roles = async () => ({}) } = {}) {
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", (_slug, _signal) => membership());
  t.mock.method(WorkspaceService.prototype, "getWorkspaceUserProjectsRole", (_slug, _signal) => roles());
  t.mock.method(UserService.prototype, "joinProject", async () => ({ message: "accepted" }));
}

function mockProjectDetails(t) {
  t.mock.method(ProjectService.prototype, "getProject", (workspaceSlug, projectId) =>
    globalThis.projectPermissionAccessHarness.fetchProjectDetails(workspaceSlug, projectId)
  );
}

test.beforeEach((t) => {
  setHarness();
  mockProjectDetails(t);
  t.mock.method(ProjectService.prototype, "getProjectUserProperties", async () => ({
    preferences: {
      pages: { block_display: true },
      navigation: { default_tab: "work_items", hide_in_more_menu: [] },
    },
    rich_filters: {},
    display_filters: {},
    display_properties: {},
    sort_order: 1,
  }));
});

function mockJoinableProject(t, status = 409) {
  let detailReads = 0;
  const join = t.mock.method(UserService.prototype, "joinProject", async () => ({ message: "accepted" }));
  setHarness({
    fetchProjectDetails: async () => {
      detailReads += 1;
      if (detailReads === 1) throw statusError(status);
      return { id: "project-1" };
    },
  });
  return join;
}

async function renderProject(t, { cachedProjectRole } = {}) {
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/issues/");
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "member-1" });
  if (cachedProjectRole) {
    await seedPermissionQueries(fixture.client, {
      userId: "member-1",
      workspaceSlug: "acme",
      workspaceRole: activeMember.role,
      projectRoles: { "project-1": cachedProjectRole },
    });
  }
  await fixture.render(content);
  return fixture;
}

function renderProjectScope(workspaceSlug, projectId) {
  return React.createElement(
    ProjectAuthWrapper,
    { workspaceSlug, projectId },
    React.createElement("main", null, `${workspaceSlug}/${projectId}`)
  );
}

async function clickButton(fixture) {
  await act(async () => fixture.environment.container.querySelector("button")?.click());
  await settleQuery();
}

await test("409 project access can be joined and resolves from refreshed native permissions", async (t) => {
  let roleReads = 0;
  mockPermissionServices(t, {
    roles: async () => (++roleReads === 1 ? {} : { "project-1": 15 }),
  });
  const join = mockJoinableProject(t, 409);

  const fixture = await renderProject(t);
  assert.equal(fixture.environment.container.querySelector("main"), null);
  await clickButton(fixture);

  assert.equal(join.mock.callCount(), 1);
  assert.equal(fixture.environment.container.querySelector("main")?.textContent, "Project content");
});

await test("403 permits project joining only for workspace admins", async (t) => {
  let roleReads = 0;
  mockPermissionServices(t, {
    membership: async () => activeAdmin,
    roles: async () => (++roleReads === 1 ? {} : { "project-1": 15 }),
  });
  const join = mockJoinableProject(t, 403);

  const fixture = await renderProject(t);
  assert.ok(fixture.environment.container.querySelector("button"));
  await clickButton(fixture);
  assert.equal(join.mock.callCount(), 1);
  assert.equal(fixture.environment.container.querySelector("main")?.textContent, "Project content");
});

await test("access probe never exposes project details when current roles omit the project", async (t) => {
  mockPermissionServices(t);
  const getProject = t.mock.method(ProjectService.prototype, "getProject", async () => ({
    id: "project-1",
    name: "Private project details",
  }));

  const fixture = await renderProject(t);
  await settleQuery();

  assert.equal(getProject.mock.callCount(), 1);
  assert.equal(fixture.environment.container.querySelector("main"), null);
  assert.doesNotMatch(fixture.environment.container.textContent, /Private project details/);
  assert.equal(
    fixture.client.getQueryData(
      projectKeys.accessProbe({
        accountId: "member-1",
        sessionGeneration: 0,
        workspaceSlug: "acme",
        workspaceRole: activeMember.role,
        membershipRevision: "revision-1",
        projectRoles: [],
        projectId: "project-1",
      })
    ),
    true
  );
});

await test("403 for a non-admin remains restricted without issuing a join", async (t) => {
  let joinWrites = 0;
  mockPermissionServices(t);
  t.mock.method(UserService.prototype, "joinProject", async () => {
    joinWrites += 1;
    return { message: "accepted" };
  });
  setHarness({ fetchProjectDetails: async () => Promise.reject(statusError(403)) });

  const fixture = await renderProject(t);
  assert.equal(fixture.environment.container.querySelector("button"), null);
  assert.equal(fixture.environment.container.querySelector("main"), null);
  assert.equal(joinWrites, 0);
});

await test("404 stays unavailable even when the project role is cached", async (t) => {
  mockPermissionServices(t);
  setHarness({ fetchProjectDetails: async () => Promise.reject(statusError(404)) });

  const fixture = await renderProject(t, { cachedProjectRole: 15 });
  assert.equal(fixture.environment.container.querySelector("main"), null);
  assert.equal(fixture.environment.container.querySelector("button"), null);
  assert.match(fixture.environment.container.textContent, /invalid_project/);
});

await test("project detail 5xx hides content and Retry reloads the read", async (t) => {
  mockPermissionServices(t, { roles: async () => ({ "project-1": 15 }) });
  let detailReads = 0;
  setHarness({
    fetchProjectDetails: async () => {
      detailReads += 1;
      if (detailReads === 1) throw statusError(503);
      return { id: "project-1" };
    },
  });

  const fixture = await renderProject(t);
  assert.equal(fixture.environment.container.querySelector("main"), null);
  assert.match(fixture.environment.container.textContent, /Unable to load project access/);
  await clickButton(fixture);
  assert.equal(detailReads, 2);
  assert.equal(fixture.environment.container.querySelector("main")?.textContent, "Project content");
});

await test("accepted join with a failed role refresh retries reads without repeating the join", async (t) => {
  let roleReads = 0;
  mockPermissionServices(t, {
    roles: async () => {
      roleReads += 1;
      if (roleReads === 1) return {};
      if (roleReads === 2) throw statusError(503);
      return { "project-1": 15 };
    },
  });
  const join = mockJoinableProject(t, 409);

  const fixture = await renderProject(t);
  await clickButton(fixture);
  assert.equal(join.mock.callCount(), 1);
  assert.equal(fixture.environment.container.querySelector("main"), null);
  assert.match(fixture.environment.container.textContent, /Project joined/);

  await clickButton(fixture);
  assert.equal(join.mock.callCount(), 1);
  assert.equal(fixture.environment.container.querySelector("main")?.textContent, "Project content");
});

await test("a late join does not carry joined feedback or a refetch into a new project scope", async (t) => {
  const joinStarted = Promise.withResolvers();
  const joinResponse = Promise.withResolvers();
  mockPermissionServices(t, { roles: async () => ({}) });
  const join = t.mock.method(UserService.prototype, "joinProject", () => {
    joinStarted.resolve();
    return joinResponse.promise;
  });
  const detailReads = [];
  setHarness({
    fetchProjectDetails: async (slug, projectId) => {
      detailReads.push([slug, projectId]);
      throw statusError(409);
    },
  });

  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/issues/");
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "member-1" });
  await fixture.render(renderProjectScope("acme", "project-1"));
  await clickButton(fixture);
  await joinStarted.promise;

  await fixture.render(renderProjectScope("beta", "project-2"));
  assert.ok(fixture.environment.container.querySelector("button"), "the new project remains independently joinable");

  await act(async () => joinResponse.resolve({ message: "accepted" }));
  await settleQuery();

  assert.deepEqual(join.mock.calls[0].arguments, ["acme", ["project-1"]]);
  assert.deepEqual(detailReads, [
    ["acme", "project-1"],
    ["beta", "project-2"],
  ]);
  assert.ok(fixture.environment.container.querySelector("button"), "the new project remains joinable");
  assert.doesNotMatch(fixture.environment.container.textContent, /Project joined/);
  assert.equal(fixture.environment.container.querySelector("main"), null);
});

await test("membership query errors hide project content and retry the membership read", async (t) => {
  let membershipReads = 0;
  mockPermissionServices(t, {
    membership: async () => {
      membershipReads += 1;
      if (membershipReads === 1) throw statusError(503);
      return activeMember;
    },
    roles: async () => ({ "project-1": 15 }),
  });

  const fixture = await renderProject(t);
  assert.equal(fixture.environment.container.querySelector("main"), null);
  assert.match(fixture.environment.container.textContent, /Unable to load project access/);
  await clickButton(fixture);
  assert.equal(membershipReads, 2);
  assert.equal(fixture.environment.container.querySelector("main")?.textContent, "Project content");
});

await test("project role query errors hide content and retry the role read", async (t) => {
  let roleReads = 0;
  mockPermissionServices(t, {
    roles: async () => {
      roleReads += 1;
      if (roleReads === 1) throw statusError(503);
      return { "project-1": 15 };
    },
  });

  const fixture = await renderProject(t);
  assert.equal(fixture.environment.container.querySelector("main"), null);
  assert.match(fixture.environment.container.textContent, /Unable to load project access/);
  await clickButton(fixture);
  assert.equal(roleReads, 2);
  assert.equal(fixture.environment.container.querySelector("main")?.textContent, "Project content");
});

await test("project settings gate follows native admin access and hides content after a failed refresh", async (t) => {
  const { ProjectSettingsAccess } = await import("../../apps/web/core/components/settings/project/access.tsx");
  const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
  mockPermissionServices(t, { roles: async () => Promise.reject(statusError(503)) });
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-1/issues/");
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "member-1" });
  await seedPermissionQueries(fixture.client, {
    userId: "member-1",
    workspaceSlug: "acme",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { "project-1": 15 },
  });
  await fixture.render(
    React.createElement(
      ProjectSettingsAccess,
      { workspaceSlug: "acme", projectId: "project-1" },
      React.createElement("main", null, "Settings")
    )
  );
  assert.equal(fixture.environment.container.querySelector("main"), null);
  const rolesKey = permissionKeys.workspaceProjectRoles(
    "member-1",
    "acme",
    EUserWorkspaceRoles.MEMBER,
    "2026-10-24T00:00:00Z"
  );
  await act(async () => fixture.client.setQueryData(rolesKey, { "project-1": 20 }));
  await settleQuery();
  assert.equal(fixture.environment.container.querySelector("main")?.textContent, "Settings");
  await act(async () => fixture.client.invalidateQueries({ queryKey: rolesKey, exact: true }));
  await settleQuery();
  assert.equal(
    fixture.environment.container.querySelector("main"),
    null,
    "failed refresh cannot keep cached admin controls visible"
  );
});

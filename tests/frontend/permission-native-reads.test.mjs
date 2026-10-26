import assert from "node:assert/strict";
import test from "node:test";
import { notifyManager, QueryClient, skipToken } from "@tanstack/react-query";
import { EUserPermissions, EUserProjectRoles, EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";
import { act, React } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { waitForQueryState as waitForQuery } from "./helpers/query-state.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const workspaceService = WorkspaceService.prototype;
const identity = await import("../../apps/web/core/queries/identity.ts");
const permissions = await import("../../apps/web/core/queries/permissions.ts");
const { useProjectPermissions, useWorkspacePermissions } = await import("../../apps/web/core/hooks/use-permissions.ts");

function seedAccount(client, userId = "permission-user") {
  client.setQueryData(identity.identityKeys.currentUser(), { id: userId });
}

function workspaceMembership(role, updatedAt, isActive = true) {
  return { is_active: isActive, role, updated_at: updatedAt };
}

function PermissionProbe({ slug = "acme", projectId = "project-a" }) {
  const workspace = useWorkspacePermissions(slug);
  const project = useProjectPermissions(slug, projectId);
  return React.createElement(
    "output",
    { "data-permission-probe": true },
    JSON.stringify({
      workspace: {
        role: workspace.workspaceRole,
        isPending: workspace.isPending,
        isError: workspace.isError,
        canAdmin: workspace.can([EUserPermissions.ADMIN]),
      },
      project: {
        role: project.projectRole,
        isPending: project.isPending,
        isError: project.isError,
        isFetching: project.rolesQuery?.isFetching ?? false,
        canAdmin: project.can([EUserPermissions.ADMIN]),
      },
    })
  );
}

function readProbe(container) {
  return JSON.parse(container.querySelector("[data-permission-probe]").textContent);
}

async function waitForProbe(environment, predicate, attempts = 20) {
  const value = readProbe(environment.container);
  if (predicate(value)) return value;
  if (attempts === 0)
    throw new Error(`Permission probe did not reach expected state: ${environment.container.textContent}`);
  await act(async () => new Promise((resolve) => setTimeout(resolve, 0)));
  return waitForProbe(environment, predicate, attempts - 1);
}

function withDeterministicNotifications(t) {
  notifyManager.setScheduler(queueMicrotask);
  t.after(() => notifyManager.setScheduler((callback) => setTimeout(callback, 0)));
}

async function assertMembershipDenied(t, membership, readLabel) {
  withDeterministicNotifications(t);
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-a/issues/");
  seedAccount(fixture.client);
  t.mock.method(workspaceService, "workspaceMemberMe", async () => membership);
  const roleRead = t.mock.method(workspaceService, "getWorkspaceUserProjectsRole", async () => ({
    "project-a": EUserProjectRoles.ADMIN,
  }));

  await fixture.render(React.createElement(PermissionProbe));
  const state = await waitForProbe(fixture.environment, (value) => !value.workspace.isPending);
  assert.equal(state.workspace.role, undefined);
  assert.equal(state.workspace.canAdmin, false);
  assert.equal(state.project.canAdmin, false);
  assert.equal(roleRead.mock.callCount(), 0, readLabel);
}

await test("membership reads and Query keys stay isolated by account and workspace", async (t) => {
  const firstClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const secondClient = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  seedAccount(firstClient, "user-a");
  seedAccount(secondClient, "user-b");
  t.after(() => {
    firstClient.clear();
    secondClient.clear();
  });
  const requests = [];
  t.mock.method(workspaceService, "workspaceMemberMe", async (slug) => {
    requests.push(slug);
    return workspaceMembership(EUserWorkspaceRoles.MEMBER, `revision-${slug}`);
  });
  const firstOptions = permissions.workspaceMembershipOptions("user-a", "acme");
  const secondOptions = permissions.workspaceMembershipOptions("user-b", "beta");
  await Promise.all([
    firstClient.fetchQuery({ ...firstOptions, staleTime: 0 }),
    secondClient.fetchQuery({ ...secondOptions, staleTime: 0 }),
  ]);

  assert.deepEqual(
    requests.toSorted((left, right) => left.localeCompare(right)),
    ["acme", "beta"]
  );
  assert.equal(firstClient.getQueryData(firstOptions.queryKey).updated_at, "revision-acme");
  assert.equal(secondClient.getQueryData(secondOptions.queryKey).updated_at, "revision-beta");
  assert.notDeepEqual(firstOptions.queryKey, secondOptions.queryKey);
});

await test("membership Query keys distinguish role and server membership revision", () => {
  assert.notDeepEqual(
    permissions.permissionKeys.workspaceProjectRoles("user-a", "acme", EUserWorkspaceRoles.ADMIN, "revision-1"),
    permissions.permissionKeys.workspaceProjectRoles("user-a", "acme", EUserWorkspaceRoles.GUEST, "revision-2")
  );
  assert.notDeepEqual(
    permissions.permissionKeys.workspaceProjectRoles("user-a", "acme", EUserWorkspaceRoles.ADMIN, "revision-1"),
    permissions.permissionKeys.workspaceProjectRoles("user-a", "acme", EUserWorkspaceRoles.ADMIN, "revision-3")
  );
  assert.equal(
    permissions.workspaceProjectRolesOptions("user-a", "acme", 999, "revision-1").queryFn,
    skipToken,
    "unknown workspace roles must not enable a project-role read"
  );
});

await test("workspace membership reads pass cancellation and discard canceled results", async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  seedAccount(client);
  t.after(() => client.clear());
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  let requestSignal;
  t.mock.method(workspaceService, "workspaceMemberMe", (_slug, signal) => {
    requestSignal = signal;
    started.resolve();
    return response.promise;
  });
  const options = permissions.workspaceMembershipOptions("permission-user", "acme");
  const request = client.fetchQuery({ ...options, staleTime: 0 }).catch((error) => error);
  await started.promise;
  await client.cancelQueries({ queryKey: options.queryKey, exact: true });
  response.resolve(workspaceMembership(EUserWorkspaceRoles.ADMIN, "revision-1"));
  await request;

  assert.ok(requestSignal instanceof AbortSignal);
  assert.equal(requestSignal.aborted, true);
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("membership reads reject a same-account late result after session generation changes", async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  seedAccount(client);
  t.after(() => client.clear());
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(workspaceService, "workspaceMemberMe", () => {
    started.resolve();
    return response.promise;
  });
  const options = permissions.workspaceMembershipOptions("permission-user", "acme");
  const request = client.fetchQuery({ ...options, staleTime: 0 });
  const rejected = assert.rejects(request, /scope changed|session changed|identity changed/i);
  await started.promise;
  identity.advanceIdentitySession(client);
  response.resolve(workspaceMembership(EUserWorkspaceRoles.ADMIN, "revision-1"));
  await rejected;
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("inactive null-role membership succeeds as denial and skips project-role reads", async (t) => {
  await assertMembershipDenied(
    t,
    workspaceMembership(null, undefined, false),
    "an inactive membership never enables the project-role query"
  );
});

await test("active membership with an unknown role denies permission and skips project-role reads", async (t) => {
  await assertMembershipDenied(
    t,
    workspaceMembership(999, "revision-1"),
    "an unknown role never enables the project-role query"
  );
});

await test("workspace-admin elevation still requires an entry in the project-role map", async (t) => {
  withDeterministicNotifications(t);
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-a/issues/");
  seedAccount(fixture.client);
  t.mock.method(workspaceService, "workspaceMemberMe", async () =>
    workspaceMembership(EUserWorkspaceRoles.ADMIN, "revision-1")
  );
  t.mock.method(workspaceService, "getWorkspaceUserProjectsRole", async () => ({}));

  await fixture.render(React.createElement(PermissionProbe));
  const state = await waitForProbe(fixture.environment, (value) => !value.project.isPending);
  assert.equal(state.workspace.canAdmin, true);
  assert.equal(state.project.role, undefined);
  assert.equal(state.project.canAdmin, false);
});

await test("a same-revision background role refresh preserves confirmed permissions", async (t) => {
  withDeterministicNotifications(t);
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-a/issues/");
  seedAccount(fixture.client);
  t.mock.method(workspaceService, "workspaceMemberMe", async () =>
    workspaceMembership(EUserWorkspaceRoles.ADMIN, "revision-1")
  );
  const refresh = Promise.withResolvers();
  let roleReads = 0;
  t.mock.method(workspaceService, "getWorkspaceUserProjectsRole", () => {
    roleReads++;
    return roleReads === 1 ? Promise.resolve({ "project-a": EUserProjectRoles.ADMIN }) : refresh.promise;
  });

  await fixture.render(React.createElement(PermissionProbe));
  const roleKey = permissions.permissionKeys.workspaceProjectRoles(
    "permission-user",
    "acme",
    EUserWorkspaceRoles.ADMIN,
    "revision-1"
  );
  await waitForQuery(fixture.client, roleKey, (state) => state?.status === "success");
  await waitForProbe(fixture.environment, (state) => state.project.canAdmin);

  const refreshStarted = waitForQuery(fixture.client, roleKey, (state) => state?.fetchStatus === "fetching");
  await act(async () => {
    void fixture.client.invalidateQueries({ queryKey: roleKey, exact: true });
    await refreshStarted;
  });
  const refreshing = await waitForProbe(fixture.environment, (state) => state.project.isFetching);
  assert.equal(refreshing.project.isPending, false);
  assert.equal(refreshing.project.canAdmin, true);

  await act(async () => {
    refresh.resolve({ "project-a": EUserProjectRoles.ADMIN });
    await waitForQuery(fixture.client, roleKey, (state) => state?.fetchStatus === "idle");
  });
  await waitForProbe(fixture.environment, (state) => !state.project.isFetching && state.project.canAdmin);
});

await test("replacing the exact membership Query prevents its late response from replacing the new result", async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  seedAccount(client);
  t.after(() => client.clear());
  const oldStarted = Promise.withResolvers();
  const oldResponse = Promise.withResolvers();
  let reads = 0;
  t.mock.method(workspaceService, "workspaceMemberMe", () => {
    reads++;
    if (reads === 1) {
      oldStarted.resolve();
      return oldResponse.promise;
    }
    return Promise.resolve(workspaceMembership(EUserWorkspaceRoles.MEMBER, "revision-new"));
  });
  const options = permissions.workspaceMembershipOptions("permission-user", "acme");
  const oldRead = client.fetchQuery({ ...options, staleTime: 0 }).catch((error) => error);
  await oldStarted.promise;
  client.removeQueries({ queryKey: options.queryKey, exact: true });
  await client.fetchQuery({ ...options, staleTime: 0 });
  oldResponse.resolve(workspaceMembership(EUserWorkspaceRoles.ADMIN, "revision-old"));
  await oldRead;
  assert.deepEqual(
    client.getQueryData(options.queryKey),
    workspaceMembership(EUserWorkspaceRoles.MEMBER, "revision-new")
  );
});

await test("project permissions follow admin→guest→admin membership revisions and deny cached roles on membership error", async (t) => {
  withDeterministicNotifications(t);
  const fixture = await createQueryRouterFixture(t, "/acme/projects/project-a/issues/");
  seedAccount(fixture.client);
  const membershipKey = permissions.permissionKeys.workspaceMembership("permission-user", "acme");
  const initialMembership = workspaceMembership(EUserWorkspaceRoles.ADMIN, "revision-1");
  const guestMembership = workspaceMembership(EUserWorkspaceRoles.GUEST, "revision-2");
  const restoredMembership = workspaceMembership(EUserWorkspaceRoles.ADMIN, "revision-3");
  const nextMembership = [
    Promise.resolve(initialMembership),
    Promise.withResolvers(),
    Promise.withResolvers(),
    Promise.withResolvers(),
  ];
  let membershipCall = 0;
  t.mock.method(workspaceService, "workspaceMemberMe", () => {
    const result = nextMembership[membershipCall++];
    if (result?.promise) {
      result.started?.resolve();
      return result.promise;
    }
    return result;
  });

  const adminRoles = { "project-a": EUserProjectRoles.ADMIN };
  const guestRoles = { "project-a": EUserProjectRoles.GUEST };
  const rolesStarted = [null, Promise.withResolvers(), Promise.withResolvers()];
  const rolesResponses = [Promise.resolve(adminRoles), Promise.withResolvers(), Promise.withResolvers()];
  let rolesCall = 0;
  t.mock.method(workspaceService, "getWorkspaceUserProjectsRole", () => {
    const index = rolesCall++;
    rolesStarted[index]?.resolve();
    return rolesResponses[index]?.promise ?? Promise.resolve(adminRoles);
  });

  await fixture.render(React.createElement(PermissionProbe));
  await waitForQuery(fixture.client, membershipKey, (state) => state?.status === "success");
  const initialRoleKey = permissions.permissionKeys.workspaceProjectRoles(
    "permission-user",
    "acme",
    EUserWorkspaceRoles.ADMIN,
    "revision-1"
  );
  await waitForQuery(fixture.client, initialRoleKey, (state) => state?.status === "success");
  await waitForProbe(fixture.environment, (state) => state.project.canAdmin);

  const guestRead = Promise.withResolvers();
  nextMembership[1] = { promise: guestRead.promise, started: Promise.withResolvers() };
  const guestStarted = nextMembership[1].started;
  void fixture.client.invalidateQueries({ queryKey: membershipKey, exact: true });
  await guestStarted.promise;
  guestRead.resolve(guestMembership);
  await waitForQuery(fixture.client, membershipKey, (state) => state?.data?.updated_at === "revision-2");
  const guestRoleKey = permissions.permissionKeys.workspaceProjectRoles(
    "permission-user",
    "acme",
    EUserWorkspaceRoles.GUEST,
    "revision-2"
  );
  await waitForProbe(fixture.environment, (state) => state.project.isPending && !state.project.canAdmin);
  await rolesStarted[1].promise;
  assert.equal(fixture.client.getQueryData(initialRoleKey), adminRoles, "the old admin projection may remain cached");
  rolesResponses[1].resolve(guestRoles);
  await waitForQuery(fixture.client, guestRoleKey, (state) => state?.status === "success");
  await waitForProbe(fixture.environment, (state) => state.project.role === EUserProjectRoles.GUEST);
  assert.equal(readProbe(fixture.environment.container).project.canAdmin, false);

  const adminRead = Promise.withResolvers();
  nextMembership[2] = { promise: adminRead.promise, started: Promise.withResolvers() };
  const adminStarted = nextMembership[2].started;
  void fixture.client.invalidateQueries({ queryKey: membershipKey, exact: true });
  await adminStarted.promise;
  adminRead.resolve(restoredMembership);
  await waitForQuery(fixture.client, membershipKey, (state) => state?.data?.updated_at === "revision-3");
  const restoredRoleKey = permissions.permissionKeys.workspaceProjectRoles(
    "permission-user",
    "acme",
    EUserWorkspaceRoles.ADMIN,
    "revision-3"
  );
  await waitForProbe(fixture.environment, (state) => state.project.isPending && !state.project.canAdmin);
  await rolesStarted[2].promise;
  assert.deepEqual(fixture.client.getQueryData(restoredRoleKey), undefined);
  assert.equal(fixture.client.getQueryData(initialRoleKey), adminRoles);
  rolesResponses[2].resolve(adminRoles);
  await waitForQuery(fixture.client, restoredRoleKey, (state) => state?.status === "success");
  await waitForProbe(fixture.environment, (state) => state.project.canAdmin);

  const failedRead = nextMembership[3];
  const failedStarted = Promise.withResolvers();
  nextMembership[3] = { promise: failedRead.promise, started: failedStarted };
  void fixture.client.invalidateQueries({ queryKey: membershipKey, exact: true });
  await failedStarted.promise;
  failedRead.reject(new Error("membership unavailable"));
  await waitForQuery(fixture.client, membershipKey, (state) => state?.status === "error");
  await waitForProbe(fixture.environment, (state) => state.workspace.isError);
  assert.equal(
    readProbe(fixture.environment.container).project.canAdmin,
    false,
    "membership errors deny cached role-map authorization"
  );
});

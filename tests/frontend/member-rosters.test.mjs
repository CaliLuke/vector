import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, skipToken } from "@tanstack/react-query";
import { EUserProjectRoles, EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { ProjectMemberService } = await import("../../apps/web/core/services/project/project-member.service.ts");
const { memberRosterKeys, workspaceMemberRosterOptions, projectMemberRosterOptions } =
  await import("../../apps/web/core/queries/member-rosters.ts");

await test("member roster services pass read cancellation signals to the transport", async (t) => {
  const workspaceService = new WorkspaceService();
  const projectService = new ProjectMemberService();
  const controller = new AbortController();
  let workspaceRequest;
  let projectRequest;
  t.mock.method(workspaceService, "get", async (...args) => {
    workspaceRequest = args;
    return { data: [] };
  });
  t.mock.method(projectService, "get", async (...args) => {
    projectRequest = args;
    return { data: [] };
  });

  await workspaceService.fetchWorkspaceMembers("acme", controller.signal);
  await projectService.fetchProjectMembers("acme", "project-a", controller.signal);

  assert.deepEqual(workspaceRequest, ["/api/workspaces/acme/members/", {}, { signal: controller.signal }]);
  assert.deepEqual(projectRequest, [
    "/api/workspaces/acme/projects/project-a/members/",
    {},
    { signal: controller.signal },
  ]);
});

function createClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
}

function seedWorkspaceScope(client, { accountId = "alice", workspaceSlug = "acme", role, revision = "rev-1" }) {
  client.setQueryData(identityKeys.currentUser(), { id: accountId });
  client.setQueryData(permissionKeys.workspaceMembership(accountId, workspaceSlug), {
    is_active: true,
    role,
    updated_at: revision,
  });
}

function seedProjectPermissionScope(client, scope, roles = { [scope.projectId]: scope.projectRole }) {
  seedWorkspaceScope(client, {
    accountId: scope.accountId,
    workspaceSlug: scope.workspaceSlug,
    role: scope.membershipRole,
    revision: scope.membershipRevision,
  });
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.membershipRole,
    scope.membershipRevision
  );
  client.setQueryData(rolesKey, roles);
  return rolesKey;
}

function holdFirstRosterRead(t, service, method, nextRead) {
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  let requestSignal;
  let reads = 0;
  t.mock.method(service, method, (...args) => {
    reads++;
    if (reads === 1) {
      requestSignal = args.at(-1);
      started.resolve();
      return response.promise;
    }
    return nextRead(...args);
  });
  return {
    started: started.promise,
    response,
    get requestSignal() {
      return requestSignal;
    },
  };
}

async function assertRosterReadCancellation(client, options, read) {
  const request = client.fetchQuery({ ...options, staleTime: 0 }).catch((error) => error);
  await read.started;
  await client.cancelQueries({ queryKey: options.queryKey, exact: true });
  read.response.resolve([]);
  await request;

  assert.ok(read.requestSignal instanceof AbortSignal);
  assert.equal(read.requestSignal.aborted, true);
  assert.equal(client.getQueryData(options.queryKey), undefined);
}

function workspaceRosterScope({
  accountId = "alice",
  workspaceSlug = "acme",
  role = EUserWorkspaceRoles.MEMBER,
  revision = "rev-1",
} = {}) {
  return { accountId, workspaceSlug, membershipRole: role, membershipRevision: revision };
}

function projectRosterScope({
  accountId = "alice",
  workspaceSlug = "acme",
  projectId = "project-a",
  workspaceRole = EUserWorkspaceRoles.MEMBER,
  projectRole = EUserProjectRoles.MEMBER,
  revision = "rev-1",
} = {}) {
  return {
    accountId,
    workspaceSlug,
    projectId,
    membershipRole: workspaceRole,
    membershipRevision: revision,
    projectRole,
  };
}

await test("workspace roster separates guest and full account projections without merging fields", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  const service = WorkspaceService.prototype;
  let reads = 0;
  t.mock.method(service, "fetchWorkspaceMembers", async () => {
    reads++;
    return reads === 1
      ? [{ id: "guest-row", role: EUserWorkspaceRoles.GUEST, member: { id: "bob", display_name: "Bob" } }]
      : [
          {
            id: "member-row",
            role: EUserWorkspaceRoles.MEMBER,
            member: { id: "bob", display_name: "Bob", email: "bob@example.test", last_login_medium: "email" },
          },
        ];
  });

  seedWorkspaceScope(client, { role: EUserWorkspaceRoles.GUEST, revision: "rev-guest" });
  const guestOptions = workspaceMemberRosterOptions(
    workspaceRosterScope({ role: EUserWorkspaceRoles.GUEST, revision: "rev-guest" })
  );
  const guestRoster = await client.fetchQuery({ ...guestOptions, staleTime: 0 });

  seedWorkspaceScope(client, { role: EUserWorkspaceRoles.MEMBER, revision: "rev-member" });
  const fullOptions = workspaceMemberRosterOptions(
    workspaceRosterScope({ role: EUserWorkspaceRoles.MEMBER, revision: "rev-member" })
  );
  const fullRoster = await client.fetchQuery({ ...fullOptions, staleTime: 0 });

  assert.notDeepEqual(guestOptions.queryKey, fullOptions.queryKey);
  assert.notDeepEqual(
    guestOptions.queryKey,
    memberRosterKeys.workspace(
      workspaceRosterScope({ accountId: "other-account", role: EUserWorkspaceRoles.GUEST, revision: "rev-guest" })
    )
  );
  assert.equal("email" in guestRoster[0].member, false);
  assert.equal("last_login_medium" in guestRoster[0].member, false);
  assert.equal(fullRoster[0].member.email, "bob@example.test");
  assert.equal(client.getQueryData(guestOptions.queryKey)[0].member.email, undefined);
});

await test("workspace roster query forwards cancellation and discards its late response", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  seedWorkspaceScope(client, { role: EUserWorkspaceRoles.MEMBER });
  const read = holdFirstRosterRead(t, WorkspaceService.prototype, "fetchWorkspaceMembers", () => []);
  const options = workspaceMemberRosterOptions(workspaceRosterScope({ role: EUserWorkspaceRoles.MEMBER }));
  await assertRosterReadCancellation(client, options, read);
});

await test("workspace roster rejects results from a replaced exact query", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  seedWorkspaceScope(client, { role: EUserWorkspaceRoles.MEMBER });
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  let reads = 0;
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", () => {
    reads++;
    if (reads === 1) {
      started.resolve();
      return response.promise;
    }
    return Promise.resolve([{ id: "fresh", role: EUserWorkspaceRoles.MEMBER, member: { id: "bob" } }]);
  });
  const options = workspaceMemberRosterOptions(workspaceRosterScope({ role: EUserWorkspaceRoles.MEMBER }));
  const oldRequest = client.fetchQuery({ ...options, staleTime: 0 }).catch((error) => error);

  await started.promise;
  client.removeQueries({ queryKey: options.queryKey, exact: true });
  const freshRoster = await client.fetchQuery({ ...options, staleTime: 0 });
  response.resolve([{ id: "stale", role: EUserWorkspaceRoles.MEMBER, member: { id: "bob" } }]);
  await oldRequest;

  assert.equal(freshRoster[0].id, "fresh");
  assert.equal(client.getQueryData(options.queryKey)[0].id, "fresh");
});

await test("workspace roster rejects a late same-account response after session generation changes", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  seedWorkspaceScope(client, { role: EUserWorkspaceRoles.ADMIN });
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", () => {
    started.resolve();
    return response.promise;
  });
  const options = workspaceMemberRosterOptions(workspaceRosterScope({ role: EUserWorkspaceRoles.ADMIN }));
  const request = client.fetchQuery({ ...options, staleTime: 0 });
  const rejected = assert.rejects(request, /scope changed|session changed|identity changed/i);

  await started.promise;
  advanceIdentitySession(client);
  response.resolve([]);
  await rejected;

  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("workspace roster rejects a response after canonical membership revision changes", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  seedWorkspaceScope(client, { role: EUserWorkspaceRoles.MEMBER });
  const started = Promise.withResolvers();
  const response = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", () => {
    started.resolve();
    return response.promise;
  });
  const options = workspaceMemberRosterOptions(workspaceRosterScope({ role: EUserWorkspaceRoles.MEMBER }));
  const request = client.fetchQuery({ ...options, staleTime: 0 });
  const rejected = assert.rejects(request, /workspace member scope changed/i);

  await started.promise;
  seedWorkspaceScope(client, { role: EUserWorkspaceRoles.MEMBER, revision: "rev-2" });
  response.resolve([]);
  await rejected;

  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("project roster requires confirmed project membership before starting a read", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  seedWorkspaceScope(client, { role: EUserWorkspaceRoles.ADMIN });
  const scope = { ...projectRosterScope({ workspaceRole: EUserWorkspaceRoles.ADMIN }), projectRole: undefined };
  const options = projectMemberRosterOptions(scope);

  assert.equal(options.queryFn, skipToken);
});

await test("project roster honors the wrapper's project-details gate without changing its key", () => {
  const scope = projectRosterScope();
  const defaultOptions = projectMemberRosterOptions(scope);
  const gatedOptions = projectMemberRosterOptions(scope, false);

  assert.deepEqual(gatedOptions.queryKey, defaultOptions.queryKey);
  assert.equal(gatedOptions.queryFn, skipToken);
});

await test("project roster rejects a late response after canonical project access is removed", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  const scope = projectRosterScope({ workspaceRole: EUserWorkspaceRoles.ADMIN, projectRole: EUserProjectRoles.ADMIN });
  const rolesKey = seedProjectPermissionScope(client, scope, { [scope.projectId]: EUserProjectRoles.MEMBER });
  const read = holdFirstRosterRead(t, ProjectMemberService.prototype, "fetchProjectMembers", () => []);
  const options = projectMemberRosterOptions(scope);
  const request = client.fetchQuery({ ...options, staleTime: 0 });
  const rejected = assert.rejects(request, /project member scope changed/i);

  await read.started;
  client.setQueryData(rolesKey, {});
  read.response.resolve([]);
  await rejected;

  assert.ok(read.requestSignal instanceof AbortSignal);
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("project roster query forwards cancellation and discards its late response", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  const scope = projectRosterScope();
  seedProjectPermissionScope(client, scope);
  const read = holdFirstRosterRead(t, ProjectMemberService.prototype, "fetchProjectMembers", () => []);
  const options = projectMemberRosterOptions(scope);
  await assertRosterReadCancellation(client, options, read);
});

await test("project roster rejects results from a replaced exact query", async (t) => {
  const client = createClient();
  t.after(() => client.clear());
  const scope = projectRosterScope();
  seedProjectPermissionScope(client, scope);
  const read = holdFirstRosterRead(t, ProjectMemberService.prototype, "fetchProjectMembers", () =>
    Promise.resolve([
      {
        member: "fresh",
        role: EUserProjectRoles.MEMBER,
        id: "membership-fresh",
        original_role: EUserProjectRoles.MEMBER,
        created_at: "now",
      },
    ])
  );
  const options = projectMemberRosterOptions(scope);
  const oldRequest = client.fetchQuery({ ...options, staleTime: 0 }).catch((error) => error);

  await read.started;
  client.removeQueries({ queryKey: options.queryKey, exact: true });
  const freshRoster = await client.fetchQuery({ ...options, staleTime: 0 });
  read.response.resolve([
    {
      member: "stale",
      role: EUserProjectRoles.MEMBER,
      id: "membership-stale",
      original_role: EUserProjectRoles.MEMBER,
      created_at: "now",
    },
  ]);
  await oldRequest;

  assert.equal(freshRoster[0].member, "fresh");
  assert.equal(client.getQueryData(options.queryKey)[0].member, "fresh");
});

import * as ContractEnums from "../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { MutationObserver, QueryClient, QueryClientProvider, QueryObserver } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import {
  identityKeys,
  permissionKeys,
  WorkspaceService,
  memberRosterKeys,
  inviteWorkspaceOptions,
  removeWorkspaceMemberOptions,
  updateWorkspaceMemberRoleOptions,
  workspaceInvitationKeys,
  workspaceInvitationListOptions,
  updateWorkspaceInvitationOptions,
  useWorkspaceInvitations,
  fixture,
  waitForState,
  waitForFetch,
  workspaceCascadeQueries,
} from "./helpers/member-roster-fixture.mjs";

function InvitationCount() {
  const query = useWorkspaceInvitations("acme");
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("output", null, query.data?.length ?? "hidden"),
    React.createElement("span", null, query.membershipQuery.status)
  );
}

function InvitationCountWithStatus() {
  const query = useWorkspaceInvitations("acme");
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("output", null, query.data?.length ?? "hidden"),
    React.createElement("span", null, query.isError ? "error" : "success")
  );
}

async function promoteTargetMember(t, client) {
  const update = t.mock.method(WorkspaceService.prototype, "updateWorkspaceMember", async () => ({
    id: "m-target",
    member: { id: "target" },
    role: ContractEnums.EUserPermissions.ADMIN,
  }));

  await new MutationObserver(client, updateWorkspaceMemberRoleOptions(client, "actor", "acme", "target")).mutate(
    ContractEnums.EUserPermissions.ADMIN
  );
  return update;
}

await test("workspace role write updates only the exact current workspace projection", async (t) => {
  const { client } = fixture(t);
  const currentScope = {
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.ADMIN,
    membershipRevision: "revision-2",
  };
  const currentKey = memberRosterKeys.workspace(currentScope);
  const priorKey = memberRosterKeys.workspace({ ...currentScope, membershipRevision: "revision-1" });
  const invitationKey = workspaceInvitationKeys.list(currentScope);
  const projectKey = memberRosterKeys.project({
    ...currentScope,
    projectId: "p1",
    projectRole: ContractEnums.EUserPermissions.ADMIN,
  });
  const currentRows = [{ id: "m-target", member: { id: "target" }, role: ContractEnums.EUserPermissions.MEMBER }];
  const oldRows = [{ id: "m-target", member: { id: "target" }, role: ContractEnums.EUserPermissions.GUEST }];
  const invitations = [{ id: "inv-1", role: ContractEnums.EUserPermissions.MEMBER, responded_at: null }];
  const projectMemberships = [{ id: "pm-1", member: "target", role: ContractEnums.EUserPermissions.MEMBER }];
  client.setQueryData(currentKey, currentRows);
  client.setQueryData(priorKey, oldRows);
  client.setQueryData(invitationKey, invitations);
  client.setQueryData(projectKey, projectMemberships);
  const update = await promoteTargetMember(t, client);

  assert.equal(client.getQueryData(currentKey)[0].role, 20);
  assert.equal(client.getQueryData(priorKey), oldRows);
  assert.equal(client.getQueryData(invitationKey), invitations);
  assert.equal(client.getQueryData(projectKey), projectMemberships);
  assert.equal(update.mock.callCount(), 1);
});

await test("accepted bulk invitation stays successful when its captured list refresh fails", async (t) => {
  const { client } = fixture(t);
  const scope = {
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.ADMIN,
    membershipRevision: "revision-2",
  };
  const listKey = workspaceInvitationKeys.list(scope);
  let refreshCalls = 0;
  const failRefresh = t.mock.method(WorkspaceService.prototype, "workspaceInvitations", async () => {
    refreshCalls++;
    throw new Error("refresh offline");
  });
  const listObserver = new QueryObserver(client, workspaceInvitationListOptions(scope));
  const unsubscribe = listObserver.subscribe(() => {});
  t.after(unsubscribe);
  await waitForState(client, listKey, "error");
  const invite = t.mock.method(WorkspaceService.prototype, "inviteWorkspace", async () => ({ message: "sent" }));
  const mutation = new MutationObserver(client, inviteWorkspaceOptions(client, "actor", "acme"));

  await mutation.mutate({ emails: [{ email: "new@example.test", role: ContractEnums.EUserPermissions.MEMBER }] });

  assert.equal(mutation.getCurrentResult().status, "success");
  assert.equal(invite.mock.callCount(), 1);
  assert.ok(failRefresh.mock.callCount() >= 1);
  assert.ok(refreshCalls >= 1);
});

await test("invitation hook hides cached rows after active membership refetch fails", async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const environment = await createReactDomTestEnvironment();
  t.after(async () => {
    await environment.unmount(() => client.clear());
  });
  client.setQueryData(identityKeys.currentUser(), { id: "actor" });
  const membershipKey = permissionKeys.workspaceMembership("actor", "acme");
  client.setQueryData(membershipKey, {
    is_active: true,
    role: ContractEnums.EUserPermissions.ADMIN,
    updated_at: "revision-2",
  });
  client.setQueryData(
    workspaceInvitationKeys.list({
      accountId: "actor",
      workspaceSlug: "acme",
      membershipRole: ContractEnums.EUserPermissions.ADMIN,
      membershipRevision: "revision-2",
    }),
    [{ id: "cached-invitation" }]
  );
  t.mock.method(WorkspaceService.prototype, "workspaceMemberMe", async () => {
    throw new Error("membership refresh offline");
  });
  await environment.render(React.createElement(QueryClientProvider, { client }, React.createElement(InvitationCount)));
  assert.equal(environment.container.querySelector("output").textContent, "1");
  assert.equal(environment.container.querySelector("span").textContent, "success");

  await act(async () => client.invalidateQueries({ queryKey: membershipKey, exact: true }));
  await environment.render(React.createElement(QueryClientProvider, { client }, React.createElement(InvitationCount)));

  assert.equal(client.getQueryState(membershipKey).status, "error");
  assert.equal(environment.container.querySelector("output").textContent, "hidden");
  assert.equal(environment.container.querySelector("span").textContent, "error");
});

await test("invitation hook hides cached rows after invitation-list refetch fails and keeps the error", async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  const environment = await createReactDomTestEnvironment();
  t.after(async () => {
    await environment.unmount(() => client.clear());
  });
  client.setQueryData(identityKeys.currentUser(), { id: "actor" });
  client.setQueryData(permissionKeys.workspaceMembership("actor", "acme"), {
    is_active: true,
    role: ContractEnums.EUserPermissions.ADMIN,
    updated_at: "revision-2",
  });
  const invitationKey = workspaceInvitationKeys.list({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.ADMIN,
    membershipRevision: "revision-2",
  });
  client.setQueryData(invitationKey, [{ id: "cached-invitation" }]);
  t.mock.method(WorkspaceService.prototype, "workspaceInvitations", async () => {
    throw new Error("invitation refresh offline");
  });
  const renderInvitations = () =>
    environment.render(
      React.createElement(QueryClientProvider, { client }, React.createElement(InvitationCountWithStatus))
    );
  await renderInvitations();
  assert.equal(environment.container.querySelector("output").textContent, "1");
  assert.equal(environment.container.querySelector("span").textContent, "success");

  await act(async () => client.invalidateQueries({ queryKey: invitationKey, exact: true }));
  await renderInvitations();

  assert.equal(client.getQueryState(invitationKey).status, "error");
  assert.equal(environment.container.querySelector("output").textContent, "hidden");
  assert.equal(environment.container.querySelector("span").textContent, "error");
});

await test("inactive workspace membership cannot authorize a role write", async (t) => {
  const { client, membershipKey } = fixture(t);
  client.setQueryData(membershipKey, {
    is_active: false,
    role: ContractEnums.EUserPermissions.ADMIN,
    updated_at: "revision-2",
  });
  const update = t.mock.method(WorkspaceService.prototype, "updateWorkspaceMember", async () => {
    throw new Error("must not send");
  });
  await assert.rejects(
    new MutationObserver(client, updateWorkspaceMemberRoleOptions(client, "actor", "acme", "target")).mutate(
      ContractEnums.EUserPermissions.ADMIN
    ),
    /admin/
  );
  assert.equal(update.mock.callCount(), 0);
});

await test("accepted workspace guest role change refreshes captured project rosters and metadata", async (t) => {
  const { client } = fixture(t);
  const captured = workspaceCascadeQueries(client);
  t.after(() => captured.stop.forEach((unsubscribe) => unsubscribe()));
  const update = t.mock.method(WorkspaceService.prototype, "updateWorkspaceMember", async () => ({
    id: "target-membership",
    member: { id: "target" },
    role: ContractEnums.EUserPermissions.GUEST,
    is_active: true,
  }));
  const mutation = new MutationObserver(client, updateWorkspaceMemberRoleOptions(client, "actor", "acme", "target"));

  await mutation.mutate(ContractEnums.EUserPermissions.GUEST);

  assert.equal(mutation.getCurrentResult().status, "success");
  assert.equal(update.mock.callCount(), 1);
  assert.deepEqual(
    [captured.projectRosterKey, captured.detailKey, captured.catalogueKey].map(
      (key) => client.getQueryState(key).status
    ),
    ["error", "error", "error"]
  );
});

await test("accepted workspace member removal refreshes captured project rosters and metadata", async (t) => {
  const { client } = fixture(t);
  const captured = workspaceCascadeQueries(client);
  t.after(() => captured.stop.forEach((unsubscribe) => unsubscribe()));
  const remove = t.mock.method(WorkspaceService.prototype, "deleteWorkspaceMember", async () => {});
  const mutation = new MutationObserver(
    client,
    removeWorkspaceMemberOptions(client, "actor", "acme", "target-membership", "target")
  );

  await mutation.mutate();

  assert.equal(mutation.getCurrentResult().status, "success");
  assert.equal(remove.mock.callCount(), 1);
  assert.deepEqual(
    [captured.projectRosterKey, captured.detailKey, captured.catalogueKey].map(
      (key) => client.getQueryState(key).status
    ),
    ["error", "error", "error"]
  );
});

await test("accepted member role cancels a deferred stale roster read before cache reconciliation", async (t) => {
  const { client } = fixture(t);
  const rosterKey = memberRosterKeys.workspace({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.ADMIN,
    membershipRevision: "revision-2",
  });
  const original = [{ id: "m-target", member: { id: "target" }, role: ContractEnums.EUserPermissions.MEMBER }];
  client.setQueryData(rosterKey, original);
  let resolveOldRead;
  const observer = new QueryObserver(client, {
    queryKey: rosterKey,
    queryFn: () => new Promise((resolve) => (resolveOldRead = resolve)),
    staleTime: 0,
    retry: false,
  });
  const unsubscribe = observer.subscribe(() => {});
  t.after(unsubscribe);
  await waitForFetch(client, rosterKey);
  const update = await promoteTargetMember(t, client);
  assert.equal(client.getQueryData(rosterKey)[0].role, 20);

  resolveOldRead([{ id: "m-target", member: { id: "target" }, role: ContractEnums.EUserPermissions.MEMBER }]);
  await new Promise(setImmediate);
  assert.equal(client.getQueryData(rosterKey)[0].role, 20);
  assert.equal(update.mock.callCount(), 1);
});

await test("accepted member role preserves a roster query replaced during cancellation", async (t) => {
  const { client } = fixture(t);
  const rosterKey = memberRosterKeys.workspace({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.ADMIN,
    membershipRevision: "revision-2",
  });
  client.setQueryData(rosterKey, [
    { id: "m-target", member: { id: "target" }, role: ContractEnums.EUserPermissions.MEMBER },
  ]);
  const replacement = [{ id: "replacement-row", member: { id: "target" }, role: ContractEnums.EUserPermissions.GUEST }];
  let resolveReplacement;
  const replacementReady = new Promise((resolve) => (resolveReplacement = resolve));
  let sawFetch = false;
  let replaced = false;
  const originalQuery = client.getQueryCache().find({ queryKey: rosterKey, exact: true });
  const observer = new QueryObserver(client, {
    queryKey: rosterKey,
    queryFn: () => new Promise(() => {}),
    staleTime: 0,
    retry: false,
  });
  const unsubscribeCache = client.getQueryCache().subscribe((event) => {
    if (event.type !== "updated" || event.query !== originalQuery) return;
    if (event.query.state.fetchStatus === "fetching") sawFetch = true;
    if (!sawFetch || replaced || event.query.state.fetchStatus !== "idle") return;
    replaced = true;
    unsubscribeCache();
    unsubscribeObserver();
    client.removeQueries({ queryKey: rosterKey, exact: true });
    client.setQueryData(rosterKey, replacement);
    resolveReplacement();
  });
  t.after(unsubscribeCache);
  const unsubscribeObserver = observer.subscribe(() => {});
  t.after(unsubscribeObserver);
  await waitForFetch(client, rosterKey);
  t.mock.method(WorkspaceService.prototype, "updateWorkspaceMember", async () => ({
    id: "m-target",
    member: { id: "target" },
    role: ContractEnums.EUserPermissions.ADMIN,
  }));

  await new MutationObserver(client, updateWorkspaceMemberRoleOptions(client, "actor", "acme", "target")).mutate(
    ContractEnums.EUserPermissions.ADMIN
  );
  await replacementReady;

  assert.equal(client.getQueryData(rosterKey), replacement);
});

await test("failed concurrent member role write cannot undo another accepted row", async (t) => {
  const { client } = fixture(t);
  const rosterKey = memberRosterKeys.workspace({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.ADMIN,
    membershipRevision: "revision-2",
  });
  client.setQueryData(rosterKey, [
    { id: "m-a", member: { id: "member-a" }, role: ContractEnums.EUserPermissions.MEMBER },
    { id: "m-b", member: { id: "member-b" }, role: ContractEnums.EUserPermissions.MEMBER },
  ]);
  let rejectA;
  let resolveB;
  let markAStarted;
  let markBStarted;
  const startedA = new Promise((resolve) => (markAStarted = resolve));
  const startedB = new Promise((resolve) => (markBStarted = resolve));
  t.mock.method(WorkspaceService.prototype, "updateWorkspaceMember", (_slug, membershipId) => {
    if (membershipId === "m-a") {
      markAStarted();
      return new Promise((_resolve, reject) => (rejectA = reject));
    }
    markBStarted();
    return new Promise((resolve) => (resolveB = resolve));
  });
  const mutationA = new MutationObserver(client, updateWorkspaceMemberRoleOptions(client, "actor", "acme", "member-a"));
  const mutationB = new MutationObserver(client, updateWorkspaceMemberRoleOptions(client, "actor", "acme", "member-b"));

  const failedWrite = mutationA.mutate(ContractEnums.EUserPermissions.ADMIN);
  const acceptedWrite = mutationB.mutate(ContractEnums.EUserPermissions.ADMIN);
  await Promise.all([startedA, startedB]);
  rejectA(new Error("member A rejected"));
  await assert.rejects(failedWrite, /member A rejected/);
  resolveB({ id: "m-b", member: { id: "member-b" }, role: ContractEnums.EUserPermissions.ADMIN });
  await acceptedWrite;

  const rows = client.getQueryData(rosterKey);
  assert.equal(rows.find((row) => row.id === "m-a").role, 15);
  assert.equal(rows.find((row) => row.id === "m-b").role, 20);
});

await test("workspace role and removal mutations serialize per member before cache reconciliation", async (t) => {
  const { client } = fixture(t);
  const rosterKey = memberRosterKeys.workspace({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.ADMIN,
    membershipRevision: "revision-2",
  });
  client.setQueryData(rosterKey, [
    { id: "membership-target", member: { id: "target" }, role: ContractEnums.EUserPermissions.MEMBER, is_active: true },
  ]);
  let resolveRole;
  const roleStarted = new Promise((resolve) => {
    t.mock.method(WorkspaceService.prototype, "updateWorkspaceMember", () => {
      resolve();
      return new Promise((done) => (resolveRole = done));
    });
  });
  const remove = t.mock.method(WorkspaceService.prototype, "deleteWorkspaceMember", async () => {});
  const roleMutation = new MutationObserver(
    client,
    updateWorkspaceMemberRoleOptions(client, "actor", "acme", "target")
  );
  const removeMutation = new MutationObserver(
    client,
    removeWorkspaceMemberOptions(client, "actor", "acme", "membership-target", "target")
  );

  const roleWrite = roleMutation.mutate(ContractEnums.EUserPermissions.MEMBER);
  await roleStarted;
  const removal = removeMutation.mutate();
  await new Promise(setImmediate);
  assert.equal(remove.mock.callCount(), 0);

  resolveRole({
    id: "membership-target",
    member: { id: "target" },
    role: ContractEnums.EUserPermissions.MEMBER,
    is_active: true,
  });
  await roleWrite;
  await removal;

  assert.equal(remove.mock.callCount(), 1);
  assert.equal(client.getQueryData(rosterKey)[0].is_active, false);
});

await test("failed invitation A update cannot roll back accepted invitation B", async (t) => {
  const { client } = fixture(t);
  const invitationKey = workspaceInvitationKeys.list({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.ADMIN,
    membershipRevision: "revision-2",
  });
  const invitationA = { id: "inv-a", email: "a@example.test", role: ContractEnums.EUserPermissions.MEMBER };
  const invitationB = { id: "inv-b", email: "b@example.test", role: ContractEnums.EUserPermissions.MEMBER };
  client.setQueryData(invitationKey, [invitationA, invitationB]);
  let rejectA;
  let resolveB;
  let markAStarted;
  let markBStarted;
  const startedA = new Promise((resolve) => (markAStarted = resolve));
  const startedB = new Promise((resolve) => (markBStarted = resolve));
  t.mock.method(WorkspaceService.prototype, "updateWorkspaceInvitation", (_slug, invitationId) => {
    if (invitationId === "inv-a") {
      markAStarted();
      return new Promise((_resolve, reject) => (rejectA = reject));
    }
    markBStarted();
    return new Promise((resolve) => (resolveB = resolve));
  });
  const mutationA = new MutationObserver(client, updateWorkspaceInvitationOptions(client, "actor", "acme", "inv-a"));
  const mutationB = new MutationObserver(client, updateWorkspaceInvitationOptions(client, "actor", "acme", "inv-b"));

  const failedWrite = mutationA.mutate({ role: ContractEnums.EUserPermissions.ADMIN });
  const acceptedWrite = mutationB.mutate({ role: ContractEnums.EUserPermissions.GUEST });
  await Promise.all([startedA, startedB]);
  resolveB({
    ...invitationB,
    role: ContractEnums.EUserPermissions.GUEST,
    responded_at: null,
    workspace: { logo_url: null },
  });
  await acceptedWrite;
  rejectA(new Error("invitation A rejected"));
  await assert.rejects(failedWrite, /invitation A rejected/);

  const rows = client.getQueryData(invitationKey);
  assert.equal(rows.find((row) => row.id === "inv-a").role, 15);
  assert.equal(rows.find((row) => row.id === "inv-b").role, 5);
});

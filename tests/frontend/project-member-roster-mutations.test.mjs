import assert from "node:assert/strict";
import test from "node:test";
import { MutationObserver, QueryObserver } from "@tanstack/react-query";
import {
  projectKeys,
  ProjectMemberService,
  memberRosterKeys,
  addProjectMembersOptions,
  removeProjectMemberOptions,
  updateProjectMemberRoleOptions,
  fixture,
  seedProjectMutationAccess,
  seedTargetProjectMutationAccess,
  seedProjectMemberships,
  failMetadataRefresh,
} from "./helpers/member-roster-fixture.mjs";

await test("project bulk-add and removal mutations serialize per project before roster reconciliation", async (t) => {
  const { client } = fixture(t);
  seedProjectMutationAccess(client, 20, 20, [
    { id: "actor-member", member: { id: "actor" }, role: 20, is_active: true },
    { id: "target-member", member: { id: "target" }, role: 15, is_active: true },
  ]);
  const rosterKey = memberRosterKeys.project({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: 20,
    membershipRevision: "revision-2",
    projectRole: 20,
    projectId: "p1",
  });
  client.setQueryData(rosterKey, []);
  let resolveAdd;
  let markAddStarted;
  const addStarted = new Promise((resolve) => (markAddStarted = resolve));
  t.mock.method(ProjectMemberService.prototype, "bulkAddMembersToProject", () => {
    markAddStarted();
    return new Promise((resolve) => (resolveAdd = resolve));
  });
  const remove = t.mock.method(ProjectMemberService.prototype, "deleteProjectMember", async () => {});
  const addMutation = new MutationObserver(client, addProjectMembersOptions(client, "actor", "acme", "p1", 20));
  const removeMutation = new MutationObserver(
    client,
    removeProjectMemberOptions(client, "actor", "acme", "p1", "target", 20)
  );

  const addWrite = addMutation.mutate({ members: [{ member_id: "target", role: 15 }] });
  await addStarted;
  const removal = removeMutation.mutate();
  await new Promise(setImmediate);
  assert.equal(remove.mock.callCount(), 0);

  resolveAdd([{ id: "canonical-target-membership", member: "target", role: 15 }]);
  await addWrite;
  await removal;

  assert.equal(remove.mock.callCount(), 1);
  assert.equal(remove.mock.calls[0].arguments[2], "canonical-target-membership");
  assert.equal(client.getQueryData(rosterKey).length, 0);
});

await test("project role write rejects elevated roles for a guest workspace member before transport", async (t) => {
  const { client } = fixture(t);
  seedTargetProjectMutationAccess(client, 20, 5);
  seedProjectMemberships(client, 20, 20, [{ id: "target-project-member", member: "target", role: 5 }]);
  const update = t.mock.method(ProjectMemberService.prototype, "updateProjectMember", async () => {
    throw new Error("must not send");
  });

  await assert.rejects(
    new MutationObserver(client, updateProjectMemberRoleOptions(client, "actor", "acme", "p1", "target", 20)).mutate(
      15
    ),
    /current role/
  );
  assert.equal(update.mock.callCount(), 0);
});

await test("project admin role mutation respects actor rank for target and assigned roles", async (t) => {
  const { client } = fixture(t);
  seedProjectMutationAccess(client, 15, 20, [
    { id: "actor-membership", member: { id: "actor" }, role: 15, is_active: true },
    { id: "target-workspace-member", member: { id: "target" }, role: 15, is_active: true },
  ]);
  const projectKey = seedProjectMemberships(client, 15, 20, [
    { id: "target-project-member", member: "target", role: 20 },
  ]);
  const update = t.mock.method(ProjectMemberService.prototype, "updateProjectMember", async () => {
    throw new Error("must not send");
  });
  const mutation = new MutationObserver(
    client,
    updateProjectMemberRoleOptions(client, "actor", "acme", "p1", "target", 20)
  );

  await assert.rejects(mutation.mutate(15), /current role/);
  client.setQueryData(projectKey, [{ id: "target-project-member", member: "target", role: 15 }]);
  await assert.rejects(mutation.mutate(20), /current role/);

  assert.equal(update.mock.callCount(), 0);
});

await test("workspace admin keeps the project role-rank exception", async (t) => {
  const { client } = fixture(t);
  seedTargetProjectMutationAccess(client, 20, 15);
  seedProjectMemberships(client, 20, 20, [{ id: "target-project-member", member: "target", role: 20 }]);
  const update = t.mock.method(ProjectMemberService.prototype, "updateProjectMember", async () => ({
    id: "target-project-member",
    member: "target",
    role: 20,
  }));

  await new MutationObserver(
    client,
    updateProjectMemberRoleOptions(client, "actor", "acme", "p1", "target", 20)
  ).mutate(20);

  assert.equal(update.mock.callCount(), 1);
});

await test("project bulk add requires active targets and honors guest/admin workspace role caps", async (t) => {
  const { client } = fixture(t);
  seedProjectMutationAccess(client, 20, 20, [
    { id: "actor-membership", member: { id: "actor" }, role: 20, is_active: true },
    { id: "guest-membership", member: { id: "guest" }, role: 5, is_active: true },
    { id: "admin-membership", member: { id: "admin" }, role: 20, is_active: true },
  ]);
  const add = t.mock.method(ProjectMemberService.prototype, "bulkAddMembersToProject", async () => []);
  const mutation = new MutationObserver(client, addProjectMembersOptions(client, "actor", "acme", "p1", 20));

  await assert.rejects(mutation.mutate({ members: [{ member_id: "missing", role: 15 }] }), /Project member roles/);
  await assert.rejects(mutation.mutate({ members: [{ member_id: "guest", role: 15 }] }), /Project member roles/);
  await assert.rejects(mutation.mutate({ members: [{ member_id: "admin", role: 15 }] }), /Project member roles/);

  assert.equal(add.mock.callCount(), 0);
});

await test("accepted project bulk add canonicalizes roster IDs and tolerates captured metadata refresh failures", async (t) => {
  const { client } = fixture(t);
  seedProjectMutationAccess(client, 15, 20, [
    { id: "existing-workspace-member", member: { id: "existing" }, role: 15, is_active: true },
    { id: "new-workspace-member", member: { id: "new" }, role: 15, is_active: true },
  ]);
  const rosterKey = memberRosterKeys.project({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: 15,
    membershipRevision: "revision-2",
    projectRole: 20,
    projectId: "p1",
  });
  client.setQueryData(rosterKey, [
    { id: "old-member-row", member: "existing", role: 15 },
    { id: "other-row", member: "other", role: 15 },
  ]);
  const projectScope = {
    accountId: "actor",
    sessionGeneration: 0,
    workspaceSlug: "acme",
    workspaceRole: 15,
    membershipRevision: "revision-2",
    projectRoles: [["p1", 20]],
    projectId: "p1",
    projectMembershipRole: 20,
    projectRole: 20,
  };
  const detailKey = projectKeys.detail(projectScope);
  const catalogueKey = projectKeys.workspace(projectScope, "full");
  client.setQueryData(detailKey, { members: ["existing"] });
  client.setQueryData(catalogueKey, [{ id: "p1", members: ["existing"] }]);
  const detailObserver = new QueryObserver(client, {
    queryKey: detailKey,
    queryFn: failMetadataRefresh,
    staleTime: Infinity,
    retry: false,
  });
  const catalogueObserver = new QueryObserver(client, {
    queryKey: catalogueKey,
    queryFn: failMetadataRefresh,
    staleTime: Infinity,
    retry: false,
  });
  const stopDetail = detailObserver.subscribe(() => {});
  const stopCatalogue = catalogueObserver.subscribe(() => {});
  t.after(stopDetail);
  t.after(stopCatalogue);
  const add = t.mock.method(ProjectMemberService.prototype, "bulkAddMembersToProject", async () => [
    { id: "canonical-existing-row", member: "existing", role: 20 },
    { id: "new-row", member: "new", role: 15 },
  ]);

  const mutation = new MutationObserver(client, addProjectMembersOptions(client, "actor", "acme", "p1", 20));
  await mutation.mutate({ members: [{ member_id: "new", role: 15 }] });

  const roster = client.getQueryData(rosterKey);
  assert.equal(mutation.getCurrentResult().status, "success");
  assert.equal(roster.filter((member) => member.member === "existing").length, 1);
  assert.equal(roster.find((member) => member.member === "existing").id, "canonical-existing-row");
  assert.equal(roster.filter((member) => member.member === "new").length, 1);
  assert.equal(client.getQueryState(detailKey).status, "error");
  assert.equal(client.getQueryState(catalogueKey).status, "error");
  assert.equal(add.mock.callCount(), 1);
});

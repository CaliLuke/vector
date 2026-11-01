import * as ContractEnums from "../../packages/types/dist/index.mjs";
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
    { id: "actor-member", member: { id: "actor" }, role: ContractEnums.EUserPermissions.ADMIN, is_active: true },
    { id: "target-member", member: { id: "target" }, role: ContractEnums.EUserPermissions.MEMBER, is_active: true },
  ]);
  const rosterKey = memberRosterKeys.project({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.ADMIN,
    membershipRevision: "revision-2",
    projectRole: ContractEnums.EUserPermissions.ADMIN,
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
  const addMutation = new MutationObserver(
    client,
    addProjectMembersOptions(client, "actor", "acme", "p1", ContractEnums.EUserPermissions.ADMIN)
  );
  const removeMutation = new MutationObserver(
    client,
    removeProjectMemberOptions(client, "actor", "acme", "p1", "target", ContractEnums.EUserPermissions.ADMIN)
  );

  const addWrite = addMutation.mutate({
    members: [{ member_id: "target", role: ContractEnums.EUserPermissions.MEMBER }],
  });
  await addStarted;
  const removal = removeMutation.mutate();
  await new Promise(setImmediate);
  assert.equal(remove.mock.callCount(), 0);

  resolveAdd([{ id: "canonical-target-membership", member: "target", role: ContractEnums.EUserPermissions.MEMBER }]);
  await addWrite;
  await removal;

  assert.equal(remove.mock.callCount(), 1);
  assert.equal(remove.mock.calls[0].arguments[2], "canonical-target-membership");
  assert.equal(client.getQueryData(rosterKey).length, 0);
});

await test("project role write rejects elevated roles for a guest workspace member before transport", async (t) => {
  const { client } = fixture(t);
  seedTargetProjectMutationAccess(client, 20, 5);
  seedProjectMemberships(client, 20, 20, [
    { id: "target-project-member", member: "target", role: ContractEnums.EUserPermissions.GUEST },
  ]);
  const update = t.mock.method(ProjectMemberService.prototype, "updateProjectMember", async () => {
    throw new Error("must not send");
  });

  await assert.rejects(
    new MutationObserver(
      client,
      updateProjectMemberRoleOptions(client, "actor", "acme", "p1", "target", ContractEnums.EUserPermissions.ADMIN)
    ).mutate(ContractEnums.EUserPermissions.MEMBER),
    /current role/
  );
  assert.equal(update.mock.callCount(), 0);
});

await test("project admin role mutation respects actor rank for target and assigned roles", async (t) => {
  const { client } = fixture(t);
  seedProjectMutationAccess(client, 15, 20, [
    { id: "actor-membership", member: { id: "actor" }, role: ContractEnums.EUserPermissions.MEMBER, is_active: true },
    {
      id: "target-workspace-member",
      member: { id: "target" },
      role: ContractEnums.EUserPermissions.MEMBER,
      is_active: true,
    },
  ]);
  const projectKey = seedProjectMemberships(client, 15, 20, [
    { id: "target-project-member", member: "target", role: ContractEnums.EUserPermissions.ADMIN },
  ]);
  const update = t.mock.method(ProjectMemberService.prototype, "updateProjectMember", async () => {
    throw new Error("must not send");
  });
  const mutation = new MutationObserver(
    client,
    updateProjectMemberRoleOptions(client, "actor", "acme", "p1", "target", ContractEnums.EUserPermissions.ADMIN)
  );

  await assert.rejects(mutation.mutate(ContractEnums.EUserPermissions.MEMBER), /current role/);
  client.setQueryData(projectKey, [
    { id: "target-project-member", member: "target", role: ContractEnums.EUserPermissions.MEMBER },
  ]);
  await assert.rejects(mutation.mutate(ContractEnums.EUserPermissions.ADMIN), /current role/);

  assert.equal(update.mock.callCount(), 0);
});

await test("workspace admin keeps the project role-rank exception", async (t) => {
  const { client } = fixture(t);
  seedTargetProjectMutationAccess(client, 20, 15);
  seedProjectMemberships(client, 20, 20, [
    { id: "target-project-member", member: "target", role: ContractEnums.EUserPermissions.ADMIN },
  ]);
  const update = t.mock.method(ProjectMemberService.prototype, "updateProjectMember", async () => ({
    id: "target-project-member",
    member: "target",
    role: ContractEnums.EUserPermissions.ADMIN,
  }));

  await new MutationObserver(
    client,
    updateProjectMemberRoleOptions(client, "actor", "acme", "p1", "target", ContractEnums.EUserPermissions.ADMIN)
  ).mutate(ContractEnums.EUserPermissions.ADMIN);

  assert.equal(update.mock.callCount(), 1);
});

await test("project bulk add requires active targets and honors guest/admin workspace role caps", async (t) => {
  const { client } = fixture(t);
  seedProjectMutationAccess(client, 20, 20, [
    { id: "actor-membership", member: { id: "actor" }, role: ContractEnums.EUserPermissions.ADMIN, is_active: true },
    { id: "guest-membership", member: { id: "guest" }, role: ContractEnums.EUserPermissions.GUEST, is_active: true },
    { id: "admin-membership", member: { id: "admin" }, role: ContractEnums.EUserPermissions.ADMIN, is_active: true },
  ]);
  const add = t.mock.method(ProjectMemberService.prototype, "bulkAddMembersToProject", async () => []);
  const mutation = new MutationObserver(
    client,
    addProjectMembersOptions(client, "actor", "acme", "p1", ContractEnums.EUserPermissions.ADMIN)
  );

  await assert.rejects(
    mutation.mutate({ members: [{ member_id: "missing", role: ContractEnums.EUserPermissions.MEMBER }] }),
    /Project member roles/
  );
  await assert.rejects(
    mutation.mutate({ members: [{ member_id: "guest", role: ContractEnums.EUserPermissions.MEMBER }] }),
    /Project member roles/
  );
  await assert.rejects(
    mutation.mutate({ members: [{ member_id: "admin", role: ContractEnums.EUserPermissions.MEMBER }] }),
    /Project member roles/
  );

  assert.equal(add.mock.callCount(), 0);
});

await test("accepted project bulk add canonicalizes roster IDs and tolerates captured metadata refresh failures", async (t) => {
  const { client } = fixture(t);
  seedProjectMutationAccess(client, 15, 20, [
    {
      id: "existing-workspace-member",
      member: { id: "existing" },
      role: ContractEnums.EUserPermissions.MEMBER,
      is_active: true,
    },
    { id: "new-workspace-member", member: { id: "new" }, role: ContractEnums.EUserPermissions.MEMBER, is_active: true },
  ]);
  const rosterKey = memberRosterKeys.project({
    accountId: "actor",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.MEMBER,
    membershipRevision: "revision-2",
    projectRole: ContractEnums.EUserPermissions.ADMIN,
    projectId: "p1",
  });
  client.setQueryData(rosterKey, [
    { id: "old-member-row", member: "existing", role: ContractEnums.EUserPermissions.MEMBER },
    { id: "other-row", member: "other", role: ContractEnums.EUserPermissions.MEMBER },
  ]);
  const projectScope = {
    accountId: "actor",
    sessionGeneration: 0,
    workspaceSlug: "acme",
    workspaceRole: ContractEnums.EUserPermissions.MEMBER,
    membershipRevision: "revision-2",
    projectRoles: [["p1", 20]],
    projectId: "p1",
    projectMembershipRole: ContractEnums.EUserPermissions.ADMIN,
    projectRole: ContractEnums.EUserPermissions.ADMIN,
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
    { id: "canonical-existing-row", member: "existing", role: ContractEnums.EUserPermissions.ADMIN },
    { id: "new-row", member: "new", role: ContractEnums.EUserPermissions.MEMBER },
  ]);

  const mutation = new MutationObserver(
    client,
    addProjectMembersOptions(client, "actor", "acme", "p1", ContractEnums.EUserPermissions.ADMIN)
  );
  await mutation.mutate({ members: [{ member_id: "new", role: ContractEnums.EUserPermissions.MEMBER }] });

  const roster = client.getQueryData(rosterKey);
  assert.equal(mutation.getCurrentResult().status, "success");
  assert.equal(roster.filter((member) => member.member === "existing").length, 1);
  assert.equal(roster.find((member) => member.member === "existing").id, "canonical-existing-row");
  assert.equal(roster.filter((member) => member.member === "new").length, 1);
  assert.equal(client.getQueryState(detailKey).status, "error");
  assert.equal(client.getQueryState(catalogueKey).status, "error");
  assert.equal(add.mock.callCount(), 1);
});

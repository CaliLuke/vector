import * as ContractEnums from "../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { MutationObserver, QueryClient } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

registerWebModuleHooks();
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { memberRosterKeys } = await import("../../apps/web/core/queries/member-rosters.ts");
const { ProjectMemberService } = await import("../../apps/web/core/services/project/project-member.service.ts");
const { updateProjectMemberRoleOptions } = await import("../../apps/web/core/queries/project-member-mutations.ts");

async function fixture(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(client, { userId: "alice", workspaceSlug: "acme", projectRoles: { p1: 20 } });

  const membershipRevision = "2026-10-24T00:00:00Z";
  const rosterScope = {
    accountId: "alice",
    workspaceSlug: "acme",
    membershipRole: ContractEnums.EUserPermissions.ADMIN,
    membershipRevision,
  };
  const workspaceRosterKey = memberRosterKeys.workspace(rosterScope);
  const projectRosterKey = memberRosterKeys.project({
    ...rosterScope,
    projectId: "p1",
    projectRole: ContractEnums.EUserPermissions.ADMIN,
  });
  client.setQueryData(workspaceRosterKey, [
    { id: "membership-a", member: { id: "alice" }, role: ContractEnums.EUserPermissions.ADMIN, is_active: true },
    { id: "membership-b", member: { id: "bob" }, role: ContractEnums.EUserPermissions.MEMBER, is_active: true },
  ]);
  client.setQueryData(projectRosterKey, [
    { id: "membership-a", member: "alice", role: ContractEnums.EUserPermissions.ADMIN, original_role: 20 },
    { id: "membership-b", member: "bob", role: ContractEnums.EUserPermissions.MEMBER, original_role: 15 },
  ]);
  const rolesKey = permissionKeys.workspaceProjectRoles(
    "alice",
    "acme",
    ContractEnums.EUserPermissions.ADMIN,
    membershipRevision
  );
  return { client, projectRosterKey, rolesKey };
}

await test("failed member role write preserves canonical permissions and another accepted member change", async (t) => {
  const { client, projectRosterKey, rolesKey } = await fixture(t);
  let rejectAlice;
  let markAliceStarted;
  let markBobStarted;
  const aliceStarted = new Promise((resolve) => (markAliceStarted = resolve));
  const bobStarted = new Promise((resolve) => (markBobStarted = resolve));
  t.mock.method(ProjectMemberService.prototype, "updateProjectMember", (_slug, _project, memberId, change) => {
    if (memberId === "membership-a") {
      markAliceStarted();
      return new Promise((_resolve, reject) => (rejectAlice = reject));
    }
    markBobStarted();
    return Promise.resolve({ id: "membership-b", member: "bob", role: change.role, original_role: change.role });
  });
  const aliceMutation = new MutationObserver(
    client,
    updateProjectMemberRoleOptions(client, "alice", "acme", "p1", "alice", ContractEnums.EUserPermissions.ADMIN)
  );
  const bobMutation = new MutationObserver(
    client,
    updateProjectMemberRoleOptions(client, "alice", "acme", "p1", "bob", ContractEnums.EUserPermissions.ADMIN)
  );

  const rejectedWrite = aliceMutation.mutate(ContractEnums.EUserPermissions.GUEST);
  const rejected = assert.rejects(rejectedWrite, /not accepted/);
  await aliceStarted;
  const acceptedWrite = bobMutation.mutate(ContractEnums.EUserPermissions.GUEST);
  rejectAlice(new Error("not accepted"));
  await rejected;
  await bobStarted;
  await acceptedWrite;

  const memberships = client.getQueryData(projectRosterKey);
  assert.equal(memberships.find((membership) => membership.member === "alice").role, 20);
  assert.equal(memberships.find((membership) => membership.member === "bob").role, 5);
  assert.deepEqual(client.getQueryData(rolesKey), { p1: 20 });
});

await test("accepted self-role change resets the stale project permission projection", async (t) => {
  const { client, projectRosterKey, rolesKey } = await fixture(t);
  t.mock.method(ProjectMemberService.prototype, "updateProjectMember", async (_slug, _projectId, id, change) => ({
    id,
    member: "alice",
    role: change.role,
    original_role: change.role,
  }));
  const mutation = new MutationObserver(
    client,
    updateProjectMemberRoleOptions(client, "alice", "acme", "p1", "alice", ContractEnums.EUserPermissions.ADMIN)
  );

  await mutation.mutate(ContractEnums.EUserPermissions.GUEST);

  assert.equal(client.getQueryData(projectRosterKey)[0].role, 5);
  assert.equal(client.getQueryData(rolesKey), undefined);
});

await test("late accepted role response cannot reset a new same-account permission session", async (t) => {
  const { client, rolesKey } = await fixture(t);
  let accept;
  let markStarted;
  const started = new Promise((resolve) => (markStarted = resolve));
  t.mock.method(
    ProjectMemberService.prototype,
    "updateProjectMember",
    () =>
      new Promise((resolve) => {
        accept = resolve;
        markStarted();
      })
  );
  const pending = new MutationObserver(
    client,
    updateProjectMemberRoleOptions(client, "alice", "acme", "p1", "alice", ContractEnums.EUserPermissions.ADMIN)
  ).mutate(ContractEnums.EUserPermissions.GUEST);

  await started;
  advanceIdentitySession(client);
  client.setQueryData(rolesKey, { p1: 15 });
  accept({ id: "membership-a", member: "alice", role: ContractEnums.EUserPermissions.GUEST, original_role: 5 });
  await pending;

  assert.deepEqual(client.getQueryData(rolesKey), { p1: 15 });
});

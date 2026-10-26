import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";

registerWebModuleHooks();

const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { memberRosterKeys } = await import("../../apps/web/core/queries/member-rosters.ts");
const { useProjectMemberRoster, useWorkspaceMemberRoster } =
  await import("../../apps/web/core/hooks/use-member-rosters.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { EUserPermissions, EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");
const workspaceService = WorkspaceService.prototype;

function ProjectRosterProbe({ canLoad }) {
  const roster = useProjectMemberRoster("acme", "p1", { canLoad });
  return React.createElement(
    "output",
    {
      "data-query-count": String(roster.query.data?.length ?? "none"),
      "data-membership-count": String(roster.memberships?.length ?? "hidden"),
      "data-member-row": roster.membershipsByMemberId?.get("bob")?.id ?? "hidden",
      "data-member-ids": roster.memberIds ? [...roster.memberIds].join(",") : "hidden",
    },
    String(roster.memberships?.length ?? "hidden")
  );
}

function WorkspaceRosterProbe() {
  const roster = useWorkspaceMemberRoster("acme");
  return React.createElement(
    "div",
    null,
    React.createElement(
      "output",
      { "data-roster-status": true },
      JSON.stringify({
        isLoading: roster.isLoading,
        isError: roster.isError,
        memberCount: roster.members?.length ?? "none",
      })
    ),
    roster.isError && React.createElement("button", { onClick: () => void roster.retry() }, "Retry")
  );
}

function readRosterStatus(container) {
  return JSON.parse(container.querySelector("[data-roster-status]").textContent);
}

async function waitForRosterStatus(environment, predicate, attempts = 20) {
  const value = readRosterStatus(environment.container);
  if (predicate(value)) return value;
  if (attempts === 0)
    throw new Error(`Workspace roster did not reach expected state: ${environment.container.textContent}`);
  await React.act(async () => new Promise((resolve) => setTimeout(resolve, 0)));
  return waitForRosterStatus(environment, predicate, attempts - 1);
}

await test("project roster readiness hides cached derived data and restores it when ready", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    projectId: "p1",
    membershipRole: EUserWorkspaceRoles.ADMIN,
    membershipRevision: "2026-10-24T00:00:00Z",
    projectRole: EUserPermissions.ADMIN,
  };
  const memberships = [
    {
      id: "membership-bob",
      member: "bob",
      role: EUserPermissions.MEMBER,
      original_role: EUserPermissions.MEMBER,
      created_at: "2026-10-20T00:00:00Z",
    },
  ];
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  await seedPermissionQueries(fixture.client, {
    userId: "alice",
    workspaceSlug: "acme",
    workspaceRole: scope.membershipRole,
    projectRoles: { p1: scope.projectRole },
  });
  fixture.client.setQueryData(memberRosterKeys.project(scope), memberships);

  await fixture.render(React.createElement(ProjectRosterProbe, { canLoad: true }));
  let output = fixture.environment.container.querySelector("output");
  assert.equal(output?.dataset.queryCount, "1");
  assert.equal(output?.dataset.membershipCount, "1");
  assert.equal(output?.dataset.memberRow, "membership-bob");
  assert.equal(output?.dataset.memberIds, "bob");

  await fixture.render(React.createElement(ProjectRosterProbe, { canLoad: false }));
  output = fixture.environment.container.querySelector("output");
  assert.equal(output?.dataset.queryCount, "1");
  assert.equal(output?.dataset.membershipCount, "hidden");
  assert.equal(output?.dataset.memberRow, "hidden");
  assert.equal(output?.dataset.memberIds, "hidden");
  assert.equal(output?.textContent, "hidden");

  await fixture.render(React.createElement(ProjectRosterProbe, { canLoad: true }));
  output = fixture.environment.container.querySelector("output");
  assert.equal(output?.dataset.queryCount, "1");
  assert.equal(output?.dataset.membershipCount, "1");
  assert.equal(output?.dataset.memberRow, "membership-bob");
  assert.equal(output?.dataset.memberIds, "bob");
});

await test("workspace roster distinguishes a pending permission read from an empty roster", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  const membershipResponse = Promise.withResolvers();
  const membershipStarted = Promise.withResolvers();
  t.mock.method(workspaceService, "workspaceMemberMe", () => {
    membershipStarted.resolve();
    return membershipResponse.promise;
  });
  t.mock.method(workspaceService, "fetchWorkspaceMembers", async () => []);

  await fixture.render(React.createElement(WorkspaceRosterProbe));
  await membershipStarted.promise;
  const pending = readRosterStatus(fixture.environment.container);
  assert.equal(pending.isLoading, true);
  assert.equal(pending.isError, false);
  assert.equal(pending.memberCount, "none");

  membershipResponse.resolve({ is_active: true, role: EUserWorkspaceRoles.ADMIN, updated_at: "revision-1" });
  const empty = await waitForRosterStatus(fixture.environment, (value) => !value.isLoading);
  assert.equal(empty.isError, false);
  assert.equal(empty.memberCount, 0);
});

await test("workspace roster retry refetches failed permission before the roster", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  fixture.client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  let membershipReads = 0;
  let rosterReads = 0;
  t.mock.method(workspaceService, "workspaceMemberMe", async () => {
    membershipReads += 1;
    if (membershipReads === 1) throw new Error("membership read failed");
    return { is_active: true, role: EUserWorkspaceRoles.ADMIN, updated_at: "revision-1" };
  });
  t.mock.method(workspaceService, "fetchWorkspaceMembers", async () => {
    rosterReads += 1;
    return [];
  });

  await fixture.render(React.createElement(WorkspaceRosterProbe));
  await waitForRosterStatus(fixture.environment, (value) => value.isError);
  assert.equal(membershipReads, 1);
  assert.equal(rosterReads, 0);

  const retry = fixture.environment.container.querySelector("button");
  await React.act(async () => retry.click());
  const empty = await waitForRosterStatus(fixture.environment, (value) => !value.isLoading && !value.isError);
  assert.equal(empty.memberCount, 0);
  assert.equal(membershipReads, 2);
  assert.equal(rosterReads, 1);
});

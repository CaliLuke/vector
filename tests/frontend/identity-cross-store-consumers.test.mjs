import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";

registerWebModuleHooks();

const identity = await import("../../apps/web/core/queries/identity.ts");
const permissions = await import("../../apps/web/core/queries/permissions.ts");
const { getCalendarState } =
  await import("../../apps/web/core/components/issues/issue-layouts/calendar/calendar-state.ts");
const { workspaceRosterMemberIds, projectRosterMemberIds } = await import("../../apps/web/helpers/member-rosters.ts");
const { usePagePermissions } = await import("../../apps/web/core/hooks/use-page-permissions.ts");

function createIdentityClient() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity, staleTime: Infinity } },
  });
  client.setQueryData(identity.currentUserOptions().queryKey, { id: "alice" });
  for (const userId of ["alice", "bob"]) {
    const membership = { is_active: true, role: 15, updated_at: "revision-1" };
    client.setQueryData(permissions.permissionKeys.workspaceMembership(userId, "acme"), membership);
    client.setQueryData(
      permissions.permissionKeys.workspaceProjectRoles(userId, "acme", membership.role, membership.updated_at),
      { "project-1": 15 }
    );
  }
  return client;
}

function PagePermissionProbe({ page, permission = "canCurrentUserAccessPage" }) {
  const pagePermissions = usePagePermissions(page, "acme");
  return React.createElement("output", null, String(pagePermissions?.[permission]));
}

async function mountPagePermissionProbe(t, { id, access }) {
  const environment = await createReactDomTestEnvironment();
  const client = createIdentityClient();
  t.after(async () => {
    await environment.unmount();
    client.clear();
  });

  const page = {
    id,
    name: access === 1 ? "Private page" : "Public page",
    owned_by: "alice",
    access,
    project_ids: ["project-1"],
  };

  const render = async (permission = "canCurrentUserAccessPage") =>
    environment.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(PagePermissionProbe, { page, permission })
      )
    );
  await render();

  return {
    client,
    page,
    render,
    rolesKey: permissions.permissionKeys.workspaceProjectRoles("alice", "acme", 15, "revision-1"),
    get result() {
      return environment.container.querySelector("output")?.textContent;
    },
  };
}

await test("page permission derives ownership from the subscribed current-user query", async (t) => {
  const fixture = await mountPagePermissionProbe(t, { id: "page-1", access: 1 });
  assert.equal(fixture.result, "true");

  await React.act(async () => {
    fixture.client.setQueryData(identity.currentUserOptions().queryKey, { id: "bob" });
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.equal(fixture.result, "false");
});

await test("page controls react to the observed workspace project-role query", async (t) => {
  const fixture = await mountPagePermissionProbe(t, { id: "page-2", access: 0 });
  assert.equal(fixture.result, "true");

  await React.act(async () => {
    fixture.client.setQueryData(fixture.rolesKey, {});
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.equal(
    fixture.result,
    "false",
    "a public page still requires an observed project membership in this workspace route"
  );
  await fixture.render("canCurrentUserEditPage");
  assert.equal(fixture.result, "false", "cached ownership cannot override missing project membership");
});

await test("calendar week bounds derive from the profile start-of-week preference", () => {
  const activeMonthDate = new Date(2026, 9, 1);
  const activeWeekDate = new Date(2026, 9, 24);
  assert.deepEqual(getCalendarState(activeMonthDate, activeWeekDate, 0).weekDateRange, {
    startDate: "2026-10-18",
    endDate: "2026-10-24",
  });

  assert.deepEqual(getCalendarState(activeMonthDate, activeWeekDate, 1).weekDateRange, {
    startDate: "2026-10-19",
    endDate: "2026-10-25",
  });
});

await test("workspace and project member order use the explicit current-user input", () => {
  const members = [
    { id: "wa", member: { id: "alice", display_name: "Alice" }, role: 15 },
    { id: "wb", member: { id: "bob", display_name: "Bob" }, role: 15 },
  ];
  const summaries = new Map(members.map((row) => [row.member.id, row]));
  const memberships = members.map((row) => ({ member: row.member.id, role: row.role }));

  assert.deepEqual(workspaceRosterMemberIds(members, "bob"), ["bob", "alice"]);
  assert.deepEqual(workspaceRosterMemberIds(members, "alice"), ["alice", "bob"]);
  assert.deepEqual(projectRosterMemberIds(memberships, summaries, "bob"), ["bob", "alice"]);
  assert.deepEqual(projectRosterMemberIds(memberships, summaries, "alice"), ["alice", "bob"]);
});

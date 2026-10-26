import assert from "node:assert/strict";
import test from "node:test";
import {
  React,
  act,
  createPermissionUiFixture,
  registerPermissionUiStubs,
  settleQuery,
} from "./helpers/permission-ui-fixture.mjs";

globalThis.memberRolePermissionTest = { React };
const reactBinding = "const React=globalThis.memberRolePermissionTest.React;";
const uiStubs = {
  "@plane/ui": `${reactBinding}
    export function CustomSelect({value,onChange,disabled,children}) {
      return React.createElement('select', { 'aria-label':'Project member role', value, disabled, onChange:(event)=>onChange(Number(event.target.value)) }, children);
    }
    CustomSelect.Option = ({value,children}) => React.createElement('option', {value}, children);
    export const CustomMenu = () => null;`,
  "@plane/propel/toast": "export const setToast=()=>{}; export const TOAST_TYPE={ERROR:'error'};",
};
registerPermissionUiStubs("/project/settings/member-columns.tsx", uiStubs);

const { AccountTypeColumn } = await import("../../apps/web/core/components/project/settings/member-columns.tsx");
const { ProjectMemberService } = await import("../../apps/web/core/services/project/project-member.service.ts");
const { EUserPermissions, EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");
const { memberRosterKeys } = await import("../../apps/web/core/queries/member-rosters.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");

const member = { id: "alice", display_name: "Alice", email: "alice@example.test" };

async function renderColumn(t, { workspaceRole, projectRoles, rolesRead, useRealRoleMutation = false } = {}) {
  const canonicalWorkspaceRole = workspaceRole ?? EUserWorkspaceRoles.MEMBER;
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/settings/projects/p1/members/",
    workspaceRole: canonicalWorkspaceRole,
    projectRoles,
    rolesRead,
  });
  const rowData = {
    original_role: EUserPermissions.ADMIN,
    workspaceRole: canonicalWorkspaceRole,
    member,
  };
  const updates = [];

  if (useRealRoleMutation) {
    const membership = fixture.client.getQueryData(permissionKeys.workspaceMembership("alice", "acme"));
    const rosterScope = {
      accountId: "alice",
      workspaceSlug: "acme",
      membershipRole: canonicalWorkspaceRole,
      membershipRevision: membership.updated_at,
    };
    fixture.client.setQueryData(memberRosterKeys.workspace(rosterScope), [
      { id: "workspace-membership-alice", member, role: canonicalWorkspaceRole, is_active: true },
    ]);
    fixture.client.setQueryData(memberRosterKeys.project({ ...rosterScope, projectId: "p1", projectRole: 20 }), [
      { id: "membership-alice", member: "alice", role: EUserPermissions.ADMIN, original_role: EUserPermissions.ADMIN },
    ]);
    t.mock.method(ProjectMemberService.prototype, "updateProjectMember", async (_slug, _projectId, id, change) => {
      updates.push({ id, role: change.role });
      return { id, member: "alice", role: change.role, original_role: change.role };
    });
  }

  await fixture.render(React.createElement(AccountTypeColumn, { rowData, projectId: "p1", workspaceSlug: "acme" }));
  return { ...fixture, updates };
}

function roleSelect(fixture) {
  return fixture.environment.container.querySelector('[aria-label="Project member role"]');
}

await test("stale roster admin cannot edit self when canonical roles are workspace and project member", async (t) => {
  const fixture = await renderColumn(t, {
    workspaceRole: EUserWorkspaceRoles.MEMBER,
    projectRoles: { p1: EUserPermissions.MEMBER },
  });

  assert.equal(roleSelect(fixture), null);
});

await test("workspace admin can edit self when canonical project membership is known", async (t) => {
  const fixture = await renderColumn(t, {
    workspaceRole: EUserWorkspaceRoles.ADMIN,
    projectRoles: { p1: EUserPermissions.MEMBER },
  });

  assert.ok(roleSelect(fixture));
});

await test("cached roster admin cannot edit self when successful project role read has no membership", async (t) => {
  const fixture = await renderColumn(t, {
    workspaceRole: EUserWorkspaceRoles.ADMIN,
    projectRoles: {},
  });

  assert.equal(roleSelect(fixture), null);
});

await test("cached roster admin cannot edit self when the project role read fails", async (t) => {
  const fixture = await renderColumn(t, {
    workspaceRole: EUserWorkspaceRoles.ADMIN,
    rolesRead: async () => {
      throw new Error("roles unavailable");
    },
  });

  assert.equal(roleSelect(fixture), null);
});

await test("accepted self-role write resets the old project permission before another edit is offered", async (t) => {
  const fixture = await renderColumn(t, {
    workspaceRole: EUserWorkspaceRoles.ADMIN,
    projectRoles: { p1: EUserPermissions.ADMIN },
    useRealRoleMutation: true,
    rolesRead: async () => ({}),
  });
  const select = roleSelect(fixture);
  assert.ok(select);

  await act(async () => {
    select.value = String(EUserPermissions.MEMBER);
    select.dispatchEvent(new Event("change", { bubbles: true }));
  });
  await settleQuery();
  await settleQuery();

  assert.deepEqual(fixture.updates, [{ id: "membership-alice", role: EUserPermissions.MEMBER }]);
  assert.equal(roleSelect(fixture), null);
});

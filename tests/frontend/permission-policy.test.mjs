import assert from "node:assert/strict";
import test from "node:test";
import { EUserPermissions, EUserProjectRoles, EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const {
  evaluateWorkspacePermission,
  evaluateProjectPermission,
  effectiveProjectRole,
  isWorkspaceRole,
  canChangeProjectMemberRole,
  canAssignProjectMemberRole,
  projectRoleAllowedForWorkspace,
} = await import("../../apps/web/helpers/permission-policy.ts");

await test("workspace permissions require an active membership and an allowed role", () => {
  assert.equal(evaluateWorkspacePermission(true, EUserWorkspaceRoles.MEMBER, [EUserPermissions.MEMBER]), true);
  assert.equal(evaluateWorkspacePermission(false, EUserWorkspaceRoles.ADMIN, [EUserPermissions.ADMIN]), false);
  assert.equal(evaluateWorkspacePermission(true, null, [EUserPermissions.ADMIN]), false);
  assert.equal(isWorkspaceRole(999), false);
  assert.equal(evaluateWorkspacePermission(true, 999, [EUserPermissions.ADMIN]), false);
});

await test("project permissions require membership before applying workspace-admin elevation", () => {
  const emptyRoles = {};
  assert.equal(
    evaluateProjectPermission(true, EUserWorkspaceRoles.ADMIN, emptyRoles, "project-a", [EUserPermissions.ADMIN]),
    false,
    "workspace admin does not gain access until project membership exists"
  );
  assert.equal(
    effectiveProjectRole(EUserWorkspaceRoles.ADMIN, { "project-a": EUserProjectRoles.GUEST }, "project-a"),
    EUserProjectRoles.ADMIN
  );
  assert.equal(
    evaluateProjectPermission(true, EUserWorkspaceRoles.ADMIN, { "project-a": EUserProjectRoles.GUEST }, "project-a", [
      EUserPermissions.ADMIN,
    ]),
    true
  );
  assert.equal(
    evaluateProjectPermission(false, EUserWorkspaceRoles.ADMIN, { "project-a": EUserProjectRoles.ADMIN }, "project-a", [
      EUserPermissions.ADMIN,
    ]),
    false,
    "an inactive workspace denies access even with a cached project role"
  );
  assert.equal(
    evaluateProjectPermission(true, null, { "project-a": EUserProjectRoles.ADMIN }, "project-a", [
      EUserPermissions.ADMIN,
    ]),
    false,
    "an active membership with no workspace role still denies project access"
  );
  assert.equal(
    effectiveProjectRole(999, { "project-a": EUserProjectRoles.ADMIN }, "project-a"),
    undefined,
    "an unknown workspace role cannot inherit project access"
  );
  for (const invalidRole of [undefined, 999]) {
    const roles = { "project-a": invalidRole };
    assert.equal(
      evaluateProjectPermission(true, EUserWorkspaceRoles.ADMIN, roles, "project-a", [EUserPermissions.ADMIN]),
      false,
      "workspace-admin elevation requires a valid project role entry"
    );
  }
});

await test("project member management requires confirmed administrator access", () => {
  for (const [actor, target, workspace, project, targetWorkspace, allowed] of [
    ["actor", "target", 20, 20, 15, true],
    ["actor", "actor", 20, 20, 20, true],
    ["actor", "target", 15, 20, 15, true],
    ["actor", "actor", 15, 20, 15, false],
    ["actor", "target", 15, 15, 15, false],
    ["actor", "target", 5, 20, 15, false],
    [undefined, "target", 20, 20, 15, false],
    ["actor", "target", 20, undefined, 15, false],
    ["actor", "target", 20, 999, 15, false],
    ["actor", "target", 20, 20, 999, false],
  ])
    assert.equal(canChangeProjectMemberRole(actor, target, workspace, project, targetWorkspace), allowed);
});

await test("project assignments stay inside current authority and workspace guest access", () => {
  for (const [role, workspace, project, allowed] of [
    [20, 15, 20, true],
    [15, 20, 20, true],
    [5, 5, 20, true],
    [15, 5, 20, false],
    [20, 15, 15, false],
    ["20", 15, 20, false],
    [999, 15, 20, false],
    [15, 999, 20, false],
    [15, 15, undefined, false],
    [15, 15, 999, false],
  ])
    assert.equal(projectRoleAllowedForWorkspace(role, workspace, project), allowed);
  assert.equal(canAssignProjectMemberRole("actor", "target", 15, 20, 20, 15), true);
  assert.equal(canAssignProjectMemberRole("actor", "actor", 15, 20, 15, 15), false);
  assert.equal(canAssignProjectMemberRole("actor", "target", 20, 20, 15, 5), false);
});

import assert from "node:assert/strict";
import test from "node:test";
import { EUserPermissions, EUserProjectRoles, EUserWorkspaceRoles } from "../../packages/types/dist/index.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { evaluateWorkspacePermission, evaluateProjectPermission, effectiveProjectRole, isWorkspaceRole } =
  await import("../../apps/web/helpers/permission-policy.ts");

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

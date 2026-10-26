import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { webRequire } from "./helpers/react.mjs";
registerWebModuleHooks();
const { QueryClient } = webRequire("@tanstack/react-query");
const { captureProjectAccessScope, projectRolesRevision } =
  await import("../../apps/web/core/queries/project-access-scope.ts");
const { identityKeys, advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");

function setup(t) {
  const client = new QueryClient();
  t.after(() => client.clear());
  const scope = {
    accountId: "alice",
    workspaceSlug: "acme",
    membershipRole: 20,
    membershipRevision: "v1",
    sessionGeneration: 0,
  };
  client.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  client.setQueryData(permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug), {
    is_active: true,
    role: scope.membershipRole,
    updated_at: scope.membershipRevision,
  });
  const rolesKey = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.membershipRole,
    scope.membershipRevision
  );
  const roles = { project: 5 };
  client.setQueryData(rolesKey, roles);
  return { client, scope, roles, rolesKey };
}

await test("workspace catalogue ownership expires when its permission query is replaced with identical roles", (t) => {
  const { client, scope, roles, rolesKey } = setup(t);
  const owner = captureProjectAccessScope(client, { ...scope, rolesRevision: projectRolesRevision(roles) });
  assert.equal(owner.isVerified(), true);
  client.removeQueries({ queryKey: rolesKey, exact: true });
  client.setQueryData(rolesKey, roles);
  assert.equal(owner.isCurrent(), false);
  assert.equal(owner.isVerified(), false);
});

await test("raw project membership changes expire ownership even when the effective admin role is unchanged", (t) => {
  const { client, scope, rolesKey } = setup(t);
  const owner = captureProjectAccessScope(client, {
    ...scope,
    projectId: "project",
    projectRole: 20,
    projectMembershipRole: 5,
  });
  assert.equal(owner.isVerified(), true);
  client.setQueryData(rolesKey, { project: 15 });
  assert.equal(owner.isCurrent(), false);
});

await test("a previous session scope cannot capture new ownership after signing in again as the same account", (t) => {
  const { client, scope } = setup(t);
  advanceIdentitySession(client);
  assert.equal(captureProjectAccessScope(client, scope).isVerified(), false);
});

await test("workspace-only access does not require a project membership snapshot", (t) => {
  const { client, scope, rolesKey } = setup(t);
  client.removeQueries({ queryKey: rolesKey, exact: true });
  const owner = captureProjectAccessScope(client, { ...scope, projectMembershipRole: undefined });
  assert.equal(owner.isVerified(), true);
});

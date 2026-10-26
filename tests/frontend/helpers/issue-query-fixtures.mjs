import { registerWebModuleHooks } from "./web-module-hooks.mjs";
registerWebModuleHooks();
const { identityKeys } = await import("../../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../../apps/web/core/queries/permissions.ts");

export function projectIssueScope(overrides = {}) {
  return {
    accountId: "alice",
    sessionGeneration: 0,
    workspaceSlug: "acme",
    workspaceRole: 15,
    membershipRevision: "membership-1",
    projectId: "p1",
    projectRole: 20,
    projectMembershipRole: 20,
    ...overrides,
  };
}

/** Seed identity and permission Query records for a scoped issue read. */
export function seedIssueQueryScope(client, scope) {
  const accountId = scope.accountId ?? "alice";
  const workspaceSlug = scope.workspaceSlug ?? "acme";
  const role = scope.membershipRole ?? scope.workspaceRole ?? 15;
  const revision = scope.membershipRevision ?? scope.revision ?? "read-scope-1";
  const projectRoles =
    scope.projectRoles ??
    (scope.projectId ? { [scope.projectId]: scope.projectMembershipRole ?? scope.projectRole ?? 20 } : {});

  const membershipKey = permissionKeys.workspaceMembership(accountId, workspaceSlug);
  const rolesKey = permissionKeys.workspaceProjectRoles(accountId, workspaceSlug, role, revision);
  client.setQueryData(identityKeys.currentUser(), { id: accountId });
  client.setQueryData(membershipKey, {
    is_active: true,
    role,
    updated_at: revision,
  });
  client.setQueryData(rolesKey, projectRoles);
  return { membershipKey, rolesKey };
}

/** Create a native QueryClient with scoped issue permissions and per-test cleanup. */
export function createIssueQueryClient(t, QueryClient, scope) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } },
  });
  const access = seedIssueQueryScope(client, scope);
  t.after(() => client.clear());
  return { client, ...access };
}

/** Seed only the canonical Query records needed by a scoped issue-list read. */
export function seedIssueReadAccess(
  client,
  { accountId = "alice", workspaceSlug = "acme", projectRoles = { p1: 20 }, revision = "read-scope-1" } = {}
) {
  seedIssueQueryScope(client, {
    accountId,
    workspaceSlug,
    membershipRole: 15,
    projectRoles,
    membershipRevision: revision,
  });
}

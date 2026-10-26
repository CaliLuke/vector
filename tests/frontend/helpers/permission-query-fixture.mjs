import { EUserWorkspaceRoles } from "../../../packages/types/dist/index.mjs";

export async function seedPermissionQueries(
  client,
  {
    userId,
    workspaceSlug,
    workspaceRole = EUserWorkspaceRoles.ADMIN,
    projectRoles = {},
    revision = "2026-10-24T00:00:00Z",
  }
) {
  const { permissionKeys } = await import("../../../apps/web/core/queries/permissions.ts");
  const updatedAt = revision;
  client.setQueryData(permissionKeys.workspaceMembership(userId, workspaceSlug), {
    is_active: true,
    role: workspaceRole,
    updated_at: updatedAt,
  });
  client.setQueryData(
    permissionKeys.workspaceProjectRoles(userId, workspaceSlug, workspaceRole, updatedAt),
    projectRoles
  );
}

export async function createPermissionQueryClient(
  t,
  { accountId, workspaceSlug, workspaceRole, membershipRevision, projectId, projectMembershipRole }
) {
  const { webRequire } = await import("./react.mjs");
  const { QueryClient } = webRequire("@tanstack/react-query");
  const { identityKeys } = await import("../../../apps/web/core/queries/identity.ts");
  const { permissionKeys } = await import("../../../apps/web/core/queries/permissions.ts");
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: accountId });
  await seedPermissionQueries(client, {
    userId: accountId,
    workspaceSlug,
    workspaceRole,
    revision: membershipRevision,
    projectRoles: { [projectId]: projectMembershipRole },
  });
  return {
    client,
    membershipKey: permissionKeys.workspaceMembership(accountId, workspaceSlug),
    rolesKey: permissionKeys.workspaceProjectRoles(accountId, workspaceSlug, workspaceRole, membershipRevision),
  };
}

import { useQueryClient } from "@tanstack/react-query";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useProjectPermissions } from "@/hooks/use-permissions";

/** Shared permission boundary for the usage-free type/property discovery reads. */
export function useWorkItemCatalogAccess(workspaceSlug: string, projectId: string | undefined, canLoad: boolean) {
  const accountId = useCurrentUserId();
  const generation = useIdentityGeneration(useQueryClient());
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  const membership = permissions.membershipQuery.isSuccess ? permissions.membershipQuery.data : undefined;
  const permissionError = permissions.membershipQuery.isError || Boolean(projectId && permissions.rolesQuery?.isError);
  const allowed =
    canLoad &&
    permissions.workspaceRole !== undefined &&
    (!projectId || permissions.projectRole !== undefined) &&
    !permissionError;
  const scope = {
    accountId,
    generation,
    workspaceSlug,
    projectId,
    membershipRole: permissions.workspaceRole,
    membershipRevision: membership?.updated_at,
    projectRole: permissions.projectRole,
  };
  const retry = async (refetch: () => Promise<unknown>) => {
    if (permissions.membershipQuery.isError) await permissions.membershipQuery.refetch();
    else if (projectId && permissions.rolesQuery.isError) await permissions.rolesQuery.refetch();
    else if (allowed) await refetch();
  };
  return { scope, permissions, allowed, permissionError, retry };
}

import { queryOptions, skipToken } from "@tanstack/react-query";
import type { TUserPermissions } from "@plane/types";
import type { TWorkspaceRosterScope } from "./member-rosters";
import { captureItemReadScope } from "./work-item-read-scope";
import { getIdentitySessionGeneration } from "./identity";
import { permissionKeys } from "./permissions";
import { WorkspaceService } from "@/services/workspace.service";

const service = new WorkspaceService();

export type WorkspaceSearchScope = TWorkspaceRosterScope & {
  generation: number;
  roles: Record<string, TUserPermissions> | undefined;
  rolesUpdatedAt: number;
};

export function workspaceSearchOptions(
  scope: WorkspaceSearchScope,
  params: { search: string; project_id?: string; workspace_search: boolean },
  enabled = true
) {
  const queryKey = [
    "workspace",
    scope.workspaceSlug,
    "search",
    scope.accountId,
    scope.generation,
    scope.membershipRole,
    scope.membershipRevision,
    scope.roles,
    scope.rolesUpdatedAt,
    params,
  ] as const;
  return queryOptions({
    queryKey,
    queryFn:
      enabled &&
      scope.accountId &&
      scope.workspaceSlug &&
      scope.membershipRevision &&
      scope.roles &&
      params.search.trim()
        ? async ({ client, signal }) => {
            const current = captureItemReadScope(
              client,
              { ...scope, projectId: undefined, projectRole: undefined },
              queryKey
            );
            const rolesKey = permissionKeys.workspaceProjectRoles(
              scope.accountId,
              scope.workspaceSlug,
              scope.membershipRole,
              scope.membershipRevision
            );
            const rolesQuery = client.getQueryCache().find({ queryKey: rolesKey, exact: true });
            const check = () => {
              if (
                !current() ||
                signal.aborted ||
                getIdentitySessionGeneration(client) !== scope.generation ||
                !rolesQuery ||
                client.getQueryCache().find({ queryKey: rolesKey, exact: true }) !== rolesQuery ||
                rolesQuery.state.status !== "success" ||
                rolesQuery.state.data !== scope.roles ||
                rolesQuery.state.dataUpdatedAt !== scope.rolesUpdatedAt
              )
                throw new Error("Search access changed. Try again.");
            };
            check();
            const result = await service.searchWorkspace(scope.workspaceSlug!, params, signal);
            check();
            return result;
          }
        : skipToken,
    retry: false,
    staleTime: 0,
    gcTime: 0,
    refetchOnWindowFocus: true,
  });
}

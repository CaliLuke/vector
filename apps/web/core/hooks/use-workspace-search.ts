import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { UseQueryResult } from "@tanstack/react-query";
import { WORKSPACE_DEFAULT_SEARCH_RESULT } from "@plane/constants";
import type { IWorkspaceSearchResults } from "@plane/types";
import { useCurrentUser } from "@/hooks/use-current-user";
import { useWorkspaceProjectRoles } from "@/hooks/use-permissions";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { workspaceSearchOptions } from "@/queries/workspace-search";

function searchDisplay(
  query: UseQueryResult<IWorkspaceSearchResults>,
  ready: boolean,
  searching: boolean,
  settled: boolean,
  permissionError: boolean
) {
  if (permissionError) return { status: "error" as const, results: WORKSPACE_DEFAULT_SEARCH_RESULT };
  if (!searching) return { status: "idle" as const, results: WORKSPACE_DEFAULT_SEARCH_RESULT };
  if (!ready || !settled || query.isPending || query.isFetching)
    return { status: "loading" as const, results: WORKSPACE_DEFAULT_SEARCH_RESULT };
  if (!query.isSuccess) return { status: "error" as const, results: WORKSPACE_DEFAULT_SEARCH_RESULT };
  const count = Object.values(query.data.results).reduce((total, rows) => total + rows.length, 0);
  return { status: count === 0 ? ("empty" as const) : ("results" as const), results: query.data };
}

export function useWorkspaceSearch(
  workspaceSlug: string | undefined,
  projectId: string | undefined,
  isWorkspaceLevel: boolean,
  searchTerm: string,
  debouncedSearchTerm: string,
  enabled: boolean
) {
  const client = useQueryClient();
  const generation = useIdentityGeneration(client);
  const { data: user } = useCurrentUser();
  const permissions = useWorkspaceProjectRoles(workspaceSlug);
  const ready = enabled && !!permissions.roles && !permissions.isError && !permissions.isPending;
  const searching = searchTerm.trim() !== "";
  const settled = searchTerm === debouncedSearchTerm;
  const query = useQuery(
    workspaceSearchOptions(
      {
        accountId: user?.id,
        workspaceSlug,
        membershipRole: permissions.workspaceRole,
        membershipRevision: permissions.membershipQuery.data?.updated_at,
        generation,
        roles: permissions.roles,
        rolesUpdatedAt: permissions.rolesQuery.dataUpdatedAt,
      },
      {
        ...(projectId ? { project_id: projectId } : {}),
        search: debouncedSearchTerm,
        workspace_search: !projectId || isWorkspaceLevel,
      },
      ready && searching && settled
    )
  );
  const retry = () => {
    if (!permissions.roles) {
      void permissions.membershipQuery.refetch();
      void permissions.rolesQuery.refetch();
    } else void query.refetch();
  };

  const denied = !permissions.isPending && !permissions.isError && !permissions.roles;
  if (denied) return { status: "denied" as const, results: WORKSPACE_DEFAULT_SEARCH_RESULT, retry };
  return { ...searchDisplay(query, ready, searching, settled, permissions.isError), retry };
}

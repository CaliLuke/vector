import { queryOptions, skipToken } from "@tanstack/react-query";
import { projectReleasesService } from "@/services/project-releases.service";
import { captureReleaseAccess, releaseErrorStatus, setReleaseAccess } from "./project-release-access";
import type { TProjectRosterScope } from "./member-rosters";
export type ReleaseScope = TProjectRosterScope & { workspaceSlug: string; projectId: string; generation: number };
export function projectReleaseOptions(scope: ReleaseScope) {
  const queryKey = [
    "workspace",
    scope.workspaceSlug,
    "release-catalog",
    scope.accountId,
    scope.generation,
    scope.projectId,
    scope.membershipRole,
    scope.membershipRevision,
    scope.projectRole,
  ] as const;
  return queryOptions({
    queryKey,
    queryFn:
      scope.accountId && scope.membershipRevision && scope.projectRole !== undefined
        ? async ({ client, signal }) => {
            const access = captureReleaseAccess(client, scope, queryKey);
            const check = () => {
              access.check();
              if (signal.aborted) throw new Error("Release request cancelled.");
            };
            check();
            try {
              const result = await projectReleasesService.catalog(scope.workspaceSlug, scope.projectId, signal);
              check();
              return result;
            } catch (cause) {
              if (!signal.aborted && access.current() && [401, 403, 404].includes(releaseErrorStatus(cause) ?? 0)) {
                setReleaseAccess(client, scope, true);
                client
                  .getQueryCache()
                  .find<unknown, unknown>({ queryKey, exact: true })
                  ?.setState({ data: undefined, error: cause, status: "error" });
              }
              throw cause;
            }
          }
        : skipToken,
    retry: false,
    gcTime: 0,
    refetchOnWindowFocus: true,
  });
}

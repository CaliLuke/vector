import { EUserPermissions } from "@plane/types";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { projectReleaseOptions } from "@/queries/project-releases";
import type { ReleaseScope } from "@/queries/project-releases";
import {
  captureReleaseAccess,
  releaseAccessOptions,
  releaseErrorStatus,
  setReleaseAccess,
} from "@/queries/project-release-access";
import { projectReleasesService } from "@/services/project-releases.service";
import type { ReleaseMutation } from "@/services/project-releases.service";

type Attempt = { input: ReleaseMutation; check: () => void; current: () => boolean };
export function useReleaseCatalog(scope: ReleaseScope) {
  const client = useQueryClient(),
    options = projectReleaseOptions(scope);
  const query = useQuery(options);
  const accessQuery = useQuery(releaseAccessOptions(scope));
  const capture = () => captureReleaseAccess(client, scope, options.queryKey);
  const mutation = useMutation({
    mutationKey: ["release-catalog", scope.accountId, scope.generation, scope.workspaceSlug, scope.projectId],
    scope: { id: `release-catalog/${scope.accountId}/${scope.generation}/${scope.workspaceSlug}/${scope.projectId}` },
    retry: false,
    mutationFn: ({ input, check }: Attempt) => {
      check();
      return projectReleasesService.mutate(scope.workspaceSlug, scope.projectId, input, check);
    },
    onSuccess: async (catalog, { current }) => {
      if (!current()) return;
      await client.cancelQueries({ queryKey: options.queryKey, exact: true });
      if (!current()) return;
      client.setQueryData(options.queryKey, catalog);
      const captured = new Set(client.getQueryCache().findAll({ queryKey: ["workspace", scope.workspaceSlug] }));
      void client.invalidateQueries({
        predicate: (entry) => current() && captured.has(entry) && entry.queryKey[2] !== "item-edit-source",
      });
    },
    onError: async (cause, { current }) => {
      if (!current()) return;
      if ([401, 403].includes(releaseErrorStatus(cause) ?? 0)) {
        setReleaseAccess(client, scope, true);
        const cancellation = client.cancelQueries({ queryKey: options.queryKey, exact: true });
        client
          .getQueryCache()
          .find({ queryKey: options.queryKey, exact: true })
          ?.setState({ data: undefined, error: cause, status: "error", fetchStatus: "idle" });
        await cancellation;
      }
    },
  });
  const save = async (input: ReleaseMutation) => {
    const access = capture();
    access.check();
    if (scope.projectRole !== EUserPermissions.ADMIN || !query.data?.can_manage)
      throw new Error("A project administrator is required to manage releases.");
    await mutation.mutateAsync({ input, ...access });
  };
  const reload = async () => {
    const { check } = capture();
    check();
    const result = await query.refetch({ throwOnError: true });
    check();
    if (!result.data) throw new Error("Could not load releases.");
    return result.data;
  };
  const retry = async () => {
    await client.cancelQueries({ queryKey: options.queryKey, exact: true });
    setReleaseAccess(client, scope, false);
    await query.refetch();
  };
  return { query, save, reload, retry, denied: accessQuery.data?.denied ?? false, pending: mutation.isPending };
}

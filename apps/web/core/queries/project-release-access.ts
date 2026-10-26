import { queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import type { ReleaseScope } from "./project-releases";
import { captureItemReadScope } from "./work-item-read-scope";
import { getIdentitySessionGeneration } from "./identity";
type Access = { epoch: number; denied: boolean };
export function releaseAccessOptions(scope: ReleaseScope) {
  return queryOptions({
    queryKey: [
      "workspace",
      scope.workspaceSlug,
      "release-access",
      scope.accountId,
      scope.generation,
      scope.projectId,
      scope.membershipRole,
      scope.membershipRevision,
      scope.projectRole,
    ],
    queryFn: skipToken,
    initialData: (): Access => ({ epoch: 0, denied: false }),
    staleTime: Infinity,
    gcTime: Infinity,
  });
}
export function setReleaseAccess(client: QueryClient, scope: ReleaseScope, denied: boolean) {
  client.setQueryData(releaseAccessOptions(scope).queryKey, (previous: Access | undefined) => ({
    epoch: (previous?.epoch ?? 0) + 1,
    denied,
  }));
}
export function captureReleaseAccess(client: QueryClient, scope: ReleaseScope, queryKey: readonly unknown[]) {
  const authorized = captureItemReadScope(client, scope, queryKey);
  const accessKey = releaseAccessOptions(scope).queryKey;
  const epoch = client.getQueryData(accessKey)?.epoch ?? 0;
  const current = () => {
    const access = client.getQueryData(accessKey);
    return (
      authorized() &&
      getIdentitySessionGeneration(client) === scope.generation &&
      !access?.denied &&
      (access?.epoch ?? 0) === epoch
    );
  };
  const check = () => {
    if (!current()) throw new Error("Project access changed. Reload before saving.");
  };
  return { current, check };
}
export function releaseErrorStatus(cause: unknown) {
  if (!cause || typeof cause !== "object") return undefined;
  const error = cause as { status?: number; response?: { status?: number } };
  return error.response?.status ?? error.status;
}

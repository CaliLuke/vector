import { readVisibleItem } from "./work-item-access";
import { queryOptions, skipToken } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import { captureItemReadScope } from "./work-item-read-scope";
import type { TProjectRosterScope } from "./member-rosters";
import { workItemSchemaService } from "@/services/work-item-schema.service";
import { nextFormulaClockBoundary } from "./work-item-clock";

type Scope = TProjectRosterScope & { issueId: string; generation: number; refreshVersion?: string };

export function itemDetailSchemaOptions(scope: Scope) {
  const input = { mode: "edit", project_id: scope.projectId, issue_id: scope.issueId } as const;
  const queryKey = [
    "workspace",
    scope.workspaceSlug,
    "work-item-schema",
    scope.accountId,
    input,
    "detail",
    scope.membershipRevision,
    scope.membershipRole,
    scope.projectRole,
    scope.refreshVersion,
    scope.generation,
  ] as const;
  const enabled = Boolean(
    scope.accountId &&
    scope.workspaceSlug &&
    scope.projectId &&
    scope.issueId &&
    scope.membershipRevision &&
    scope.projectRole !== undefined
  );
  return queryOptions({
    queryKey,
    queryFn: enabled
      ? async ({ client, signal }) => {
          const current = captureItemReadScope(client, scope, queryKey);
          return readVisibleItem(
            client,
            {
              ...scope,
              workspaceSlug: scope.workspaceSlug!,
              projectId: scope.projectId!,
              generation: scope.generation,
            },
            current,
            () => workItemSchemaService.schema(scope.workspaceSlug!, input, signal)
          );
        }
      : skipToken,
    retry: false,
    staleTime: 0,
    refetchOnMount: true,
    refetchOnWindowFocus: true,
    refetchInterval: (query) =>
      query.state.data?.properties.some(
        (field) => field.kind === "Formula" && /\b(?:NOW|TODAY)\s*\(/i.test(field.configuration.expression)
      )
        ? nextFormulaClockBoundary()
        : false,
  });
}

export function refreshItemSchema(
  client: QueryClient,
  workspaceSlug: string,
  issueId: string,
  options?: { throwOnError?: boolean }
) {
  return client.invalidateQueries(
    {
      predicate: ({ queryKey }) => {
        if (queryKey[0] !== "workspace" || queryKey[1] !== workspaceSlug || queryKey[2] !== "work-item-schema")
          return false;
        const input = queryKey[4];
        return typeof input === "object" && input !== null && "issue_id" in input && input.issue_id === issueId;
      },
    },
    options
  );
}

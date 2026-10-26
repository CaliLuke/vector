import { EUserPermissions } from "@plane/types";
import { captureVisibleItemRead } from "@/queries/work-item-access";
import { queryOptions, skipToken } from "@tanstack/react-query";
import type { TProjectRosterScope } from "./member-rosters";
import { workItemEditsService } from "@/services/work-item-edits.service";
import { workItemSchemaService } from "@/services/work-item-schema.service";

export type PublishedItemEditScope = TProjectRosterScope & {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  generation: number;
};

/** A retained edit baseline belongs to one editor intent, never a live read view. */
export function publishedItemEditSourceOptions(scope: PublishedItemEditScope, intentId: string) {
  const queryKey = [
    "workspace",
    scope.workspaceSlug,
    "item-edit-source",
    scope.accountId,
    scope.generation,
    scope.projectId,
    scope.issueId,
    scope.membershipRole,
    scope.membershipRevision,
    scope.projectRole,
    intentId,
  ] as const;
  return queryOptions({
    queryKey,
    queryFn:
      scope.accountId &&
      scope.membershipRevision &&
      (scope.projectRole === EUserPermissions.MEMBER || scope.projectRole === EUserPermissions.ADMIN)
        ? async ({ client, signal }) => {
            const check = captureVisibleItemRead(client, scope, queryKey, signal);
            check();
            const [source, schema] = await Promise.all([
              workItemEditsService.source(scope.workspaceSlug, "edit", scope.issueId, scope.projectId, signal),
              workItemSchemaService.schema(
                scope.workspaceSlug,
                { mode: "edit", project_id: scope.projectId, issue_id: scope.issueId },
                signal
              ),
            ]);
            check();
            // Fields and source can race an external type change. Never initialize
            // a draft using another type's controls or another item's fingerprint.
            if (
              !("sequence_id" in source.item) ||
              source.item.id !== scope.issueId ||
              source.item.project_id !== scope.projectId ||
              (source.item.type_id ?? "") !== schema.type_id
            )
              throw new Error("The work item changed while loading. Reload its current values.");
            return { source: { ...source, item: source.item }, schema };
          }
        : skipToken,
    retry: false,
    staleTime: "static",
    gcTime: 0,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });
}

import sortBy from "lodash-es/sortBy.js";
import { queryOptions, skipToken, type QueryClient } from "@tanstack/react-query";
import type { TIssue, TUserPermissions, IUserProjectsRole } from "@plane/types";
import { IssueService } from "@/services/issue/issue.service";
import { permissionKeys } from "./permissions";
import { captureItemReadScope } from "./work-item-read-scope";
import { getIdentitySessionGeneration } from "./identity";
import type { TProjectRosterScope } from "./member-rosters";
import { nextFormulaClockBoundary } from "./work-item-clock";

export type TableValueTarget = Pick<TIssue, "id" | "updated_at">;
export type TableValueResult = Pick<TIssue, "property_values" | "properties_evaluated_at">;
export class PropertyCatalogChanged extends Error {
  constructor() {
    super("Property definitions changed. Reload custom columns.");
  }
}
const service = new IssueService();
export type TableValueScope = TProjectRosterScope & {
  generation: number;
  membershipProjectRole: TUserPermissions | undefined;
};

export function captureTablePropertyRead(client: QueryClient, scope: TableValueScope, queryKey: readonly unknown[]) {
  const access = captureItemReadScope(client, scope, queryKey);
  const key = permissionKeys.workspaceProjectRoles(
    scope.accountId,
    scope.workspaceSlug,
    scope.membershipRole,
    scope.membershipRevision
  );
  return () =>
    access() &&
    getIdentitySessionGeneration(client) === scope.generation &&
    scope.membershipProjectRole !== undefined &&
    client.getQueryData<IUserProjectsRole>(key)?.[scope.projectId ?? ""] === scope.membershipProjectRole;
}

/** A batch is owned by its verified reader, never seeded from the shared item map. */
export function tablePropertyValuesOptions(
  scope: TableValueScope,
  revision: number,
  items: TableValueTarget[],
  enabled = true,
  hasFormulaColumn = false
) {
  const queryKey = [
    "workspace",
    scope.workspaceSlug,
    "table-property-values",
    scope.accountId,
    scope.generation,
    scope.projectId,
    scope.membershipRole,
    scope.membershipRevision,
    scope.projectRole,
    scope.membershipProjectRole,
    revision,
    items.map((item) => [item.id, item.updated_at]),
  ] as const;
  const ready =
    enabled &&
    !!scope.accountId &&
    !!scope.workspaceSlug &&
    !!scope.projectId &&
    !!scope.membershipRevision &&
    scope.projectRole !== undefined &&
    items.length > 0;
  return queryOptions({
    queryKey,
    queryFn: ready
      ? async ({ client, signal }) => {
          const access = captureTablePropertyRead(client, scope, queryKey);
          const check = () => {
            if (signal.aborted || !access() || getIdentitySessionGeneration(client) !== scope.generation)
              throw new Error("Work item access changed.");
          };
          check();
          const rows = await service.retrieveIssues(
            scope.workspaceSlug!,
            scope.projectId!,
            items.map((item) => item.id),
            { includeProperties: true, signal }
          );
          check();
          const requested = new Set(items.map((item) => item.id));
          const values: Record<string, TableValueResult> = {};
          for (const row of rows) {
            if (!requested.has(row.id) || row.project_id !== scope.projectId || values[row.id])
              throw new Error("Invalid custom property response.");
            if (row.properties_revision !== revision) throw new PropertyCatalogChanged();
            if (!row.property_values || !row.properties_evaluated_at)
              throw new Error("Custom property values are unavailable.");
            values[row.id] = {
              property_values: row.property_values,
              properties_evaluated_at: row.properties_evaluated_at,
            };
          }
          return values;
        }
      : skipToken,
    retry: false,
    staleTime: 30_000,
    gcTime: 60_000,
    refetchOnMount: hasFormulaColumn ? "always" : true,
    refetchOnWindowFocus: hasFormulaColumn ? "always" : true,
    refetchInterval: ready && hasFormulaColumn ? nextFormulaClockBoundary : false,
    refetchIntervalInBackground: false,
  });
}

export function tablePropertyBatches(items: TIssue[], size = 100) {
  const projects = new Map<string, Map<string, TableValueTarget>>();
  for (const item of items) {
    if (!item.project_id || item.is_draft || item.archived_at) continue;
    const project = projects.get(item.project_id) ?? new Map<string, TableValueTarget>();
    project.set(item.id, { id: item.id, updated_at: item.updated_at });
    projects.set(item.project_id, project);
  }
  return sortBy([...projects], ([project]) => project).flatMap(([projectId, values]) => {
    const rows = sortBy([...values.values()], (item) => item.id);
    const batches: { projectId: string; items: TableValueTarget[] }[] = [];
    for (let index = 0; index < rows.length; index += size)
      batches.push({ projectId, items: rows.slice(index, index + size) });
    return batches;
  });
}

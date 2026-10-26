import { createContext, useContext } from "react";
import { useQueries, useQueryClient } from "@tanstack/react-query";
import type { TIssue } from "@plane/types";
import { useWorkspaceProjectRoles } from "./use-permissions";
import { useWorkItemCatalogAccess } from "./use-work-item-catalog-access";
import { effectiveProjectRole } from "@/helpers/permission-policy";
import {
  tablePropertyBatches,
  tablePropertyValuesOptions,
  captureTablePropertyRead,
} from "@/queries/work-item-table-values";
import type { TableValueResult } from "@/queries/work-item-table-values";

type TableValues = { status: "pending" | "error" | "unavailable" | "ready"; data?: TableValueResult };
export const TablePropertyValuesContext = createContext<Record<string, TableValues>>({});
export const useTablePropertyValue = (issueId: string) => useContext(TablePropertyValuesContext)[issueId];

export function useWorkItemTableValues(
  workspaceSlug: string,
  items: TIssue[],
  revision: number | undefined,
  enabled: boolean
) {
  const client = useQueryClient();
  const access = useWorkItemCatalogAccess(workspaceSlug, undefined, enabled);
  const permissions = useWorkspaceProjectRoles(workspaceSlug);
  const batches = enabled ? tablePropertyBatches(items) : [];
  const scopes = batches.map(({ projectId }) => ({
    ...access.scope,
    projectId,
    membershipProjectRole: permissions.roles?.[projectId],
    projectRole: effectiveProjectRole(access.scope.membershipRole ?? null, permissions.roles, projectId) ?? undefined,
  }));
  const permissionError = access.permissionError || permissions.isError;
  const allowed = access.allowed && permissions.roles !== undefined && revision !== undefined;
  const options = batches.map((batch, index) =>
    tablePropertyValuesOptions(scopes[index], revision ?? 0, batch.items, allowed)
  );
  const queries = useQueries({ queries: options });
  const values: Record<string, TableValues> = {};
  for (const [index, batch] of batches.entries()) {
    const query = queries[index];
    const granted = allowed && captureTablePropertyRead(client, scopes[index], options[index].queryKey)();
    for (const item of batch.items) {
      const data = granted && query.isSuccess ? query.data[item.id] : undefined;
      values[item.id] = tableValueState({
        data,
        denied: !granted,
        pending: !allowed || (granted && query.isPending),
        error: permissionError || query.isError,
      });
    }
  }
  return {
    values,
    isError: enabled && (permissionError || queries.some((query) => query.isError)),
    retry: async () => {
      if (access.permissionError) await access.retry(async () => {});
      else if (permissions.isError) await permissions.rolesQuery.refetch();
      else await Promise.all(queries.map((query) => query.refetch()));
    },
  };
}

function tableValueState({
  data,
  denied,
  pending,
  error,
}: {
  data?: TableValueResult;
  denied: boolean;
  pending: boolean;
  error: boolean;
}): TableValues {
  if (error) return { status: "error" };
  if (pending) return { status: "pending" };
  if (denied || !data) return { status: "unavailable" };
  return { status: "ready", data };
}

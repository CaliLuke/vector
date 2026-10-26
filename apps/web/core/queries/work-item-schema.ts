import { queryOptions, skipToken } from "@tanstack/react-query";
import { workItemSchemaService } from "@/services/work-item-schema.service";
import type { ItemSchemaInput } from "@/services/work-item-schema.service";
import { workItemTypesService } from "@/services/work-item-types.service";

export type ItemSchemaScope = {
  userId: string | undefined;
  workspaceSlug: string | undefined;
  input: ItemSchemaInput;
};

export function itemSchemaOptions({ userId, workspaceSlug, input }: ItemSchemaScope) {
  const ready = input.mode === "draft" || Boolean(input.project_id);
  return queryOptions({
    queryKey: ["workspace", workspaceSlug, "work-item-schema", userId, input] as const,
    queryFn:
      userId && workspaceSlug && ready
        ? ({ signal }) => workItemSchemaService.schema(workspaceSlug, input, signal)
        : skipToken,
    retry: false,
    // Keep the field configuration current when a modal/detail opens again.
    staleTime: 0,
  });
}

export function itemTypeChoicesOptions(
  userId: string | undefined,
  workspaceSlug: string | undefined,
  projectId?: string
) {
  return queryOptions({
    queryKey: ["workspace", workspaceSlug, "item-type-choices", userId, projectId] as const,
    queryFn:
      userId && workspaceSlug
        ? async ({ signal }) => {
            if (projectId) return workItemTypesService.projectCatalog(workspaceSlug, projectId, signal);
            const catalog = await workItemTypesService.catalog(workspaceSlug, signal);
            return {
              enabled: catalog.enabled,
              types: catalog.types.map((type) => Object.assign({}, type, { available: type.active })),
            };
          }
        : skipToken,
    retry: false,
    staleTime: 0,
  });
}

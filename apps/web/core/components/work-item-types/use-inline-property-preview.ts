import { captureVisibleItemRead } from "@/queries/work-item-access";
import { useMemo } from "react";
import { skipToken, useQuery } from "@tanstack/react-query";
import type { TIssue } from "@plane/types";
import useDebounce from "@/hooks/use-debounce";
import type { PublishedItemEditScope } from "@/queries/published-item-edit";
import { workItemSchemaService } from "@/services/work-item-schema.service";
import type { ItemSchema } from "@/services/work-item-schema.service";
import { submittedPropertyValues } from "./item-property-draft";
import type { PropertyValues } from "./item-property-draft";

export function useInlinePropertyPreview(
  scope: PublishedItemEditScope,
  schema: ItemSchema,
  item: Partial<TIssue>,
  values: PropertyValues
) {
  const body = useMemo(
    () => ({
      name: item.name ?? "",
      priority: item.priority ?? "none",
      start_date: item.start_date ?? "",
      target_date: item.target_date ?? "",
      custom_properties: submittedPropertyValues(values, schema),
    }),
    [item, values, schema]
  );
  const debounced = useDebounce(body, 250);
  const hasFormulas = schema.properties.some((field) => field.kind === "Formula");
  const queryKey = [
    "workspace",
    scope.workspaceSlug,
    "inline-property-preview",
    scope,
    debounced,
    schema.revision,
  ] as const;
  const query = useQuery({
    queryKey,
    queryFn:
      hasFormulas && body === debounced
        ? async ({ client, signal }) => {
            const check = captureVisibleItemRead(client, scope, queryKey, signal);
            check();
            const result = await workItemSchemaService.preview(
              scope.workspaceSlug,
              { mode: "edit", issue_id: scope.issueId, project_id: scope.projectId },
              debounced,
              signal,
              check
            );
            check();
            return result;
          }
        : skipToken,
    retry: false,
    gcTime: 0,
    refetchOnWindowFocus: false,
    refetchInterval: hasFormulas ? 60_000 : false,
  });
  return {
    computed: body === debounced ? query.data?.computed : undefined,
    error: query.isError,
    cause: query.error,
    changed:
      query.data !== undefined &&
      (!query.data.enabled || query.data.revision !== schema.revision || query.data.type_id !== schema.type_id),
  };
}

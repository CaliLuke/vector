import { useCurrentUserId } from "@/hooks/use-current-user";
import { useConfigurationMutation } from "./configuration-mutation";
import { useQuery } from "@tanstack/react-query";
import { useParams } from "@tanstack/react-router";
import { Button } from "@plane/propel/button";
import { typeErrorMessage, workItemTypesService } from "@/services/work-item-types.service";
import type { TypeMutation } from "@/services/work-item-types.service";
import { workItemPropertiesService } from "@/services/work-item-properties.service";
import type { PropertyMutation } from "@/services/work-item-properties.service";
import { propertyForEditor, propertyToSave } from "./property-adapter";
import { TypeLibrary } from "./library";
import { EmptyState } from "./controls";

export function WorkspaceTypeLibrary() {
  const accountId = useCurrentUserId();
  const { workspaceSlug = "" } = useParams({ strict: false });
  const queryKey = ["workspace", workspaceSlug, "work-item-types"];
  const query = useQuery({
    queryKey,
    queryFn: ({ signal }) => workItemPropertiesService.catalog(workspaceSlug, signal),
    enabled: !!workspaceSlug,
  });
  const mutation = useConfigurationMutation(workspaceSlug, (input: TypeMutation) =>
    workItemTypesService.mutate(workspaceSlug, input)
  );
  const propertyMutation = useConfigurationMutation(workspaceSlug, (input: PropertyMutation) =>
    workItemPropertiesService.mutate(workspaceSlug, input)
  );
  if (query.isPending)
    return <output aria-label="Loading work item types" className="h-24 animate-pulse rounded-lg bg-layer-1" />;
  if (!query.data)
    return (
      <EmptyState title="Couldn't load work item types" description={typeErrorMessage(query.error)}>
        <Button
          onClick={() => {
            void query.refetch();
          }}
        >
          Try again
        </Button>
      </EmptyState>
    );
  const catalog = query.data;
  return (
    <>
      {(mutation.error || propertyMutation.error) && (
        <p role="alert" className="mb-4 text-13 text-danger-primary">
          {typeErrorMessage(mutation.error ?? propertyMutation.error)}
        </p>
      )}
      {!catalog.enabled ? (
        <EmptyState
          title="Give every kind of work its own type"
          description="Enable work item types to create shared definitions across your workspace. Existing work without a type will become Task. Once enabled, work item types cannot be turned off."
        >
          <Button
            disabled={!catalog.can_manage || mutation.isPending}
            onClick={() => mutation.mutate({ action: "enable", revision: catalog.revision })}
          >
            Turn on Work item types
          </Button>
        </EmptyState>
      ) : (
        <TypeLibrary
          key={`${workspaceSlug}:${accountId}`}
          readOnly={!catalog.can_manage || mutation.isPending || propertyMutation.isPending}
          persistence={{
            types: catalog.types,
            properties: catalog.properties.map((record) => propertyForEditor(record.definition, record.used)),
            attachments: catalog.attachments,
            projects: catalog.projects,
            members: catalog.members,
            saveProperty: async (property, creating = false, revision = catalog.revision) => {
              const accepted = await propertyMutation.mutateAsync({
                action: creating ? "create" : "update",
                revision,
                definition: propertyToSave(property),
              });
              const record = accepted.properties.find((item) => item.definition.id === property.id);
              if (!record) throw new Error("The saved property is missing from the accepted catalog.");
              return { property: propertyForEditor(record.definition, record.used), revision: accepted.revision };
            },
            attachProperties: async (typeId, propertyIds, revision = catalog.revision) => {
              await propertyMutation.mutateAsync({
                action: "attach",
                revision,
                type_id: typeId,
                property_ids: propertyIds,
              });
            },
            propertyRemoval: {
              scope: workspaceSlug,
              accountId,
              preview: (input, signal) => workItemPropertiesService.previewRemoval(workspaceSlug, input, signal),
              commit: (input) => propertyMutation.mutateAsync(input),
              refreshRevision: async () => {
                const result = await query.refetch();
                if (result.error || !result.data) throw result.error ?? new Error("Couldn't load configuration.");
                return result.data.revision;
              },
            },
            previewFormula: (input, signal) => workItemPropertiesService.previewFormula(workspaceSlug, input, signal),
            projectCount: catalog.project_count,
            revision: catalog.revision,
            save: async (type, creating = false, revision = catalog.revision) => {
              await mutation.mutateAsync({ action: creating ? "create" : "update", type, revision });
            },
            remove: async (id, revision = catalog.revision) => {
              await mutation.mutateAsync({ action: "delete", id, revision });
            },
          }}
        />
      )}
    </>
  );
}

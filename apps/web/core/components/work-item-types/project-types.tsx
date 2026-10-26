import { ProjectTypeDefault } from "./project-type-default";
import { useConfigurationMutation } from "./configuration-mutation";
import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Download } from "lucide-react";
import { Button } from "@plane/propel/button";
import { ToggleSwitch } from "@plane/ui";
import { typeErrorMessage, workItemTypesService } from "@/services/work-item-types.service";
import type { ProjectTypeMutation, ProjectWorkItemType } from "@/services/work-item-types.service";
import { EmptyState, PreviewDialog, TypeIcon } from "./controls";
import { ProjectTypeImport } from "./project-type-import";

export function ProjectTypes({ workspaceSlug, projectId }: { workspaceSlug: string; projectId: string }) {
  const queryKey = ["workspace", workspaceSlug, "project", projectId, "work-item-types"];
  const query = useQuery({
    queryKey,
    queryFn: ({ signal }) => workItemTypesService.projectCatalog(workspaceSlug, projectId, signal),
  });
  const [importing, setImporting] = useState(false);
  const [removing, setRemoving] = useState<ProjectWorkItemType>();
  const mutation = useConfigurationMutation(workspaceSlug, (input: ProjectTypeMutation) =>
    workItemTypesService.mutateProject(workspaceSlug, projectId, input)
  );
  if (query.isPending)
    return (
      <output aria-label="Loading project work item types" className="block h-24 animate-pulse rounded bg-layer-1" />
    );
  if (!query.data)
    return (
      <EmptyState title="Couldn't load work item types" description={typeErrorMessage(query.error)}>
        <Button onClick={() => void query.refetch()}>Try again</Button>
      </EmptyState>
    );
  const catalog = query.data;
  const libraryLink = (
    <Link
      to="/$workspaceSlug/settings/work-item-types"
      params={{ workspaceSlug }}
      search={{ screen: undefined, state: undefined }}
      className="text-accent-primary"
    >
      Workspace type library
    </Link>
  );
  if (!catalog.enabled)
    return (
      <EmptyState
        title="Work item types are not enabled"
        description="A workspace administrator can enable types in the workspace library."
      >
        {libraryLink}
      </EmptyState>
    );
  const imported = catalog.types.filter((type) => type.imported || type.default || type.mandatory);
  const available = catalog.types.filter((type) => !type.imported && !type.default && !type.mandatory);
  const disabled = !catalog.can_manage || mutation.isPending;
  const change = (action: ProjectTypeMutation["action"], id: string) =>
    mutation.mutate({ action, type_ids: [id], revision: catalog.revision });
  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        {libraryLink}
        {catalog.can_manage && (
          <Button
            prependIcon={<Download />}
            disabled={mutation.isPending}
            onClick={() => {
              mutation.reset();
              setImporting(true);
            }}
          >
            Import from workspace
          </Button>
        )}
      </div>
      {mutation.error && !importing && !removing && (
        <p role="alert" className="mb-4 text-13 text-danger-primary">
          {typeErrorMessage(mutation.error)}
        </p>
      )}
      <ProjectTypeDefault
        types={catalog.types}
        value={catalog.default_type_id}
        disabled={disabled}
        onChange={(id) =>
          mutation.mutate({ action: "set_default", type_ids: id ? [id] : [], revision: catalog.revision })
        }
      />
      <div className="space-y-3">
        {imported.map((type) => (
          <div key={type.id} className="flex flex-wrap items-center gap-3 rounded-lg border border-subtle p-4">
            <TypeIcon type={type} />
            <div className="min-w-0 flex-1">
              <h3 className="text-14 font-medium break-words">{type.name}</h3>
              <p className="mt-1 text-12 break-words text-tertiary">{type.description}</p>
            </div>
            {type.default || type.mandatory ? (
              <span className="rounded bg-layer-2 px-2 py-1 text-11 text-secondary">
                {type.default ? "Built-in" : "Mandatory"}
              </span>
            ) : (
              <>
                <span className="text-11 text-tertiary">
                  {!type.active ? "Inactive in workspace" : type.project_active ? "Available" : "Paused"}
                </span>
                <ToggleSwitch
                  label={`Use ${type.name} in ${catalog.project_name}`}
                  value={type.project_active}
                  disabled={disabled || !type.active}
                  onChange={(active) => change(active ? "resume" : "pause", type.id)}
                />
                {catalog.can_manage && (
                  <Button
                    variant="secondary"
                    disabled={disabled}
                    onClick={() => {
                      mutation.reset();
                      setRemoving(type);
                    }}
                  >
                    Remove {type.name}
                  </Button>
                )}
              </>
            )}
          </div>
        ))}
      </div>
      {importing && (
        <ProjectTypeImport
          types={available}
          projectName={catalog.project_name}
          canManage={catalog.can_manage}
          onClose={() => setImporting(false)}
          onSave={async (ids) => {
            await mutation.mutateAsync({ action: "import", type_ids: ids, revision: catalog.revision });
          }}
        />
      )}
      {removing && (
        <PreviewDialog
          title={`Remove ${removing.name}?`}
          description="This type will no longer be available for new work in this project. Existing work items and drafts keep their type and values."
          onClose={() => {
            if (!mutation.isPending) setRemoving(undefined);
          }}
          footer={
            <>
              <Button variant="secondary" disabled={mutation.isPending} onClick={() => setRemoving(undefined)}>
                Cancel
              </Button>
              <Button
                disabled={disabled}
                onClick={() =>
                  mutation.mutate(
                    { action: "remove", type_ids: [removing.id], revision: catalog.revision },
                    { onSuccess: () => setRemoving(undefined) }
                  )
                }
              >
                Remove type
              </Button>
            </>
          }
        >
          {mutation.error && (
            <p role="alert" className="mb-3 text-13 text-danger-primary">
              {typeErrorMessage(mutation.error)}
            </p>
          )}
          <p className="text-13 text-secondary">You can import this type again later.</p>
        </PreviewDialog>
      )}
    </div>
  );
}

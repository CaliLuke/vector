import { useId, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { Download } from "lucide-react";
import { Button } from "@plane/propel/button";
import { ToggleSwitch } from "@plane/ui";
import { EmptyState, PreviewDialog, TypeIcon, inputStyle } from "./controls";
import { useTypePreview } from "./preview-provider";

export function ProjectTypesPreview({ readOnly = false }: { readOnly?: boolean }) {
  const { content, setContent, announce } = useTypePreview();
  const importFormId = useId();
  const [projectId, setProjectId] = useState(content.projects[0].id);
  const [importing, setImporting] = useState(false);
  const form = useForm({
    defaultValues: { selectedTypeIds: [] as string[] },
    validators: {
      onSubmit: ({ value }) => (value.selectedTypeIds.length ? undefined : "Choose at least one type to import."),
    },
    onSubmit: ({ value }) => {
      setContent((current) => ({
        ...current,
        projects: current.projects.map((item) =>
          item.id === projectId ? { ...item, types: [...new Set([...item.types, ...value.selectedTypeIds])] } : item
        ),
      }));
      setImporting(false);
      announce(`${value.selectedTypeIds.length} types imported into ${project.name}.`);
      form.reset({ selectedTypeIds: [] });
    },
  });
  const selected = useStore(form.store, (state) => state.values.selectedTypeIds);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const project = content.projects.find((item) => item.id === projectId)!;
  const types = content.types.filter((type) => project.types.includes(type.id) || type.mandatory);
  const available = content.types.filter((type) => !types.some((item) => item.id === type.id));
  return (
    <div>
      <div className="mb-7 flex flex-wrap items-end justify-between gap-4">
        <label className="space-y-2 text-13">
          <span className="block text-tertiary">Sample project</span>
          <select
            className={inputStyle}
            value={projectId}
            onChange={(event) => {
              setProjectId(event.target.value);
              form.reset({ selectedTypeIds: [] });
            }}
          >
            {content.projects.map((item) => (
              <option key={item.id} value={item.id}>
                {item.name}
              </option>
            ))}
          </select>
        </label>
        <Button
          disabled={readOnly}
          prependIcon={<Download />}
          onClick={() => {
            form.reset({ selectedTypeIds: [] });
            setImporting(true);
          }}
        >
          Import from workspace
        </Button>
      </div>
      <p className="mb-5 text-13 text-tertiary">
        Choose the types this project needs. Definitions and properties are managed at workspace level.
      </p>
      <div className="space-y-3">
        {types.map((type) => (
          <div key={type.id} className="flex items-center gap-3 rounded-lg border border-subtle p-4">
            <TypeIcon type={type} />
            <div className="min-w-0 flex-1">
              <h3 className="text-14 font-medium">{type.name}</h3>
              <p className="mt-1 text-12 text-tertiary">{type.description}</p>
            </div>
            {type.mandatory ? (
              <span className="rounded bg-layer-2 px-2 py-1 text-11 text-secondary">
                {type.id === "task" ? "Default" : "Mandatory"}
              </span>
            ) : (
              <>
                <span className="text-11 text-tertiary">{type.active ? "Available" : "Inactive in workspace"}</span>
                <ToggleSwitch
                  label={`Use ${type.name} in ${project.name}`}
                  value={project.types.includes(type.id)}
                  disabled={readOnly}
                  onChange={(active) => {
                    setContent((current) => ({
                      ...current,
                      projects: current.projects.map((item) =>
                        item.id === projectId
                          ? {
                              ...item,
                              types: active ? [...item.types, type.id] : item.types.filter((id) => id !== type.id),
                            }
                          : item
                      ),
                    }));
                    announce(`${type.name} removed from new selections. Existing sample items are kept.`);
                  }}
                />
              </>
            )}
          </div>
        ))}
      </div>
      {importing && (
        <PreviewDialog
          title="Import from workspace"
          description={`Add work item types to ${project.name}.`}
          onClose={() => {
            if (!isSubmitting) setImporting(false);
          }}
          footer={
            <>
              <Button type="button" variant="secondary" disabled={isSubmitting} onClick={() => setImporting(false)}>
                Cancel
              </Button>
              <Button type="submit" form={importFormId} disabled={!selected.length || isSubmitting}>
                Add{selected.length > 0 && ` (${selected.length})`}
              </Button>
            </>
          }
        >
          <form
            id={importFormId}
            onSubmit={(event) => {
              event.preventDefault();
              form.setErrorMap({ onSubmit: undefined });
              void form.handleSubmit().catch(() => undefined);
            }}
          >
            {available.length ? (
              available.map((type) => (
                <label
                  key={type.id}
                  className="flex cursor-pointer items-center gap-3 rounded px-2 py-3 hover:bg-layer-1"
                >
                  <form.Field name="selectedTypeIds">
                    {(field) => (
                      <input
                        type="checkbox"
                        checked={field.state.value.includes(type.id)}
                        disabled={isSubmitting}
                        onChange={(event) =>
                          field.handleChange(
                            event.target.checked
                              ? [...field.state.value, type.id]
                              : field.state.value.filter((id) => id !== type.id)
                          )
                        }
                      />
                    )}
                  </form.Field>
                  <TypeIcon type={type} />
                  <span className="flex-1">
                    <span className="block text-13 font-medium">{type.name}</span>
                    <span className="block text-12 text-tertiary">{type.description}</span>
                  </span>
                  {!type.active && <span className="text-11 text-tertiary">Inactive</span>}
                </label>
              ))
            ) : (
              <EmptyState
                title="All types are already available"
                description="Create more types in the workspace library to import them here."
              />
            )}
          </form>
        </PreviewDialog>
      )}
    </div>
  );
}

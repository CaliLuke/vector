import { useForm, useStore } from "@tanstack/react-form";
import { Button } from "@plane/propel/button";
import { Form, FormErrors } from "@plane/ui";
import { typeErrorMessage } from "@/services/work-item-types.service";
import type { ProjectWorkItemType } from "@/services/work-item-types.service";
import { EmptyState, PreviewDialog, TypeIcon } from "./controls";

export function ProjectTypeImport({
  types,
  projectName,
  canManage,
  onSave,
  onClose,
}: {
  types: ProjectWorkItemType[];
  projectName: string;
  canManage: boolean;
  onSave: (ids: string[]) => Promise<void>;
  onClose: () => void;
}) {
  const form = useForm({
    defaultValues: { selected: [] as ProjectWorkItemType[] },
    validators: {
      onSubmit: ({ value }) => {
        if (!canManage) return "A project administrator is required to import types.";
        if (!value.selected.length) return "Choose at least one type to import.";
        if (value.selected.some((selection) => !types.some((type) => type.id === selection.id && type.active)))
          return "One of the selected types is no longer available. Review your selection.";
        return undefined;
      },
    },
    onSubmit: async ({ value }) => {
      try {
        await onSave(value.selected.map((type) => type.id));
        onClose();
      } catch (error) {
        form.setErrorMap({ onSubmit: { form: typeErrorMessage(error), fields: {} } });
        throw error;
      }
    },
  });
  const selected = useStore(form.store, (state) => state.values.selected);
  const submitting = useStore(form.store, (state) => state.isSubmitting);
  const errors = useStore(form.store, (state) => state.errors);
  const choices = [...types, ...selected.filter((selection) => !types.some((type) => type.id === selection.id))];
  return (
    <PreviewDialog
      title="Import from workspace"
      description={`Add work item types to ${projectName}.`}
      onClose={() => {
        if (!submitting) onClose();
      }}
      footer={
        <>
          <Button variant="secondary" disabled={submitting} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" form="project-types-import" disabled={!canManage || !selected.length || submitting}>
            Add{selected.length > 0 && ` (${selected.length})`}
          </Button>
        </>
      }
    >
      <Form
        id="project-types-import"
        onSubmit={() => {
          form.setErrorMap({ onSubmit: undefined });
          return form.handleSubmit().catch(() => undefined);
        }}
      >
        <FormErrors errors={errors} />
        {choices.length ? (
          <form.Field name="selected">
            {(field) => (
              <div className="space-y-1">
                {choices.map((type) => (
                  <label key={type.id} className="flex items-center gap-3 rounded px-2 py-3 hover:bg-layer-1">
                    <input
                      type="checkbox"
                      checked={field.state.value.some((selection) => selection.id === type.id)}
                      disabled={
                        submitting ||
                        !canManage ||
                        (!type.active && !selected.some((selection) => selection.id === type.id))
                      }
                      onChange={(event) =>
                        field.handleChange(
                          event.target.checked
                            ? [...field.state.value, type]
                            : field.state.value.filter((selection) => selection.id !== type.id)
                        )
                      }
                    />
                    <TypeIcon type={type} />
                    <span className="min-w-0 flex-1">
                      <span className="block text-13 font-medium break-words">{type.name}</span>
                      <span className="block text-12 break-words text-tertiary">{type.description}</span>
                    </span>
                    {!types.some((available) => available.id === type.id) && (
                      <span className="text-11 text-tertiary">No longer available</span>
                    )}
                    {!type.active && <span className="text-11 text-tertiary">Inactive in workspace</span>}
                  </label>
                ))}
              </div>
            )}
          </form.Field>
        ) : (
          <EmptyState
            title="All types are already available"
            description="Create and activate more types in the workspace library to import them here."
          />
        )}
      </Form>
    </PreviewDialog>
  );
}

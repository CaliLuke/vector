import { useId } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { Button } from "@plane/propel/button";
import { Form } from "@plane/ui";
import { typeErrorMessage } from "@/services/work-item-types.service";
import type { PreviewContent, PreviewType } from "./preview-content";
import { useTypePreview } from "./preview-provider";
import { EmptyState, PreviewDialog } from "./controls";

export function AttachProperties({
  type,
  onClose,
  onAttach,
  properties: availableProperties,
  onManageProperties,
}: {
  type: PreviewType;
  onClose: () => void;
  onAttach?: (properties: string[]) => Promise<void>;
  properties?: PreviewContent["properties"];
  onManageProperties?: () => void;
}) {
  const { content, setContent, announce } = useTypePreview();
  const formId = useId();
  const options = availableProperties ?? content.properties;
  const available = options.filter((property) => !type.properties.includes(property.id));
  const form = useForm({
    defaultValues: { selected: [] as string[] },
    validators: {
      onSubmit: ({ value }) => {
        if (!value.selected.length) return "Choose at least one property.";
        const labels = [...type.properties, ...value.selected].map((id) =>
          options
            .find((property) => property.id === id)
            ?.name.trim()
            .toLowerCase()
        );
        return new Set(labels).size !== labels.length ? "Properties on a type must have unique titles." : undefined;
      },
    },
    onSubmit: async ({ value }) => {
      const properties = [...type.properties, ...value.selected];
      if (onAttach) await onAttach(properties);
      else
        setContent((current) => ({
          ...current,
          types: current.types.map((item) => (item.id === type.id ? { ...item, properties } : item)),
        }));
      announce(`${value.selected.length} properties attached to ${type.name}.`);
      onClose();
    },
  });
  const selected = useStore(form.store, (state) => state.values.selected);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  return (
    <PreviewDialog
      title="Add properties"
      description={`Choose properties for ${type.name}.`}
      onClose={() => {
        if (!isSubmitting) onClose();
      }}
      footer={
        <>
          <Button variant="secondary" disabled={isSubmitting} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" form={formId} disabled={isSubmitting || selected.length === 0}>
            Add{selected.length > 0 && ` (${selected.length})`}
          </Button>
        </>
      }
    >
      <Form
        id={formId}
        onSubmit={async () => {
          form.setErrorMap({ onSubmit: undefined });
          await form.handleSubmit().catch((cause: unknown) => {
            form.setErrorMap({ onSubmit: { form: typeErrorMessage(cause), fields: {} } });
          });
        }}
      >
        {!available.length ? (
          <EmptyState
            title={options.length === 0 ? "No properties yet" : "All properties are attached"}
            description={
              options.length === 0
                ? "Create a property in the workspace library, then attach it to this type."
                : "Every workspace property is already attached to this type."
            }
          >
            {onManageProperties && (
              <Button variant="secondary" onClick={onManageProperties}>
                Manage properties
              </Button>
            )}
          </EmptyState>
        ) : (
          available.map((property) => (
            <label
              key={property.id}
              className="flex cursor-pointer items-center gap-3 rounded px-2 py-3 hover:bg-layer-1"
            >
              <input
                type="checkbox"
                disabled={isSubmitting}
                checked={selected.includes(property.id)}
                onChange={(event) =>
                  form.setFieldValue(
                    "selected",
                    event.target.checked ? [...selected, property.id] : selected.filter((id) => id !== property.id)
                  )
                }
              />
              <span className="flex-1 text-13">{property.name}</span>
              <span className="rounded bg-layer-2 px-2 py-1 text-11 text-secondary">{property.kind}</span>
            </label>
          ))
        )}
        {formError && (
          <p role="alert" className="text-danger-primary">
            {formError}
          </p>
        )}
      </Form>
    </PreviewDialog>
  );
}

import { useId } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { Button } from "@plane/propel/button";
import { Form } from "@plane/ui";
import { PreviewDialog } from "./controls";
import type { PreviewType } from "./preview-content";
import { useTypePreview } from "./preview-provider";
import { preserveProperties } from "./preview-actions";

export function DeleteType({ type, onClose }: { type: PreviewType; onClose: () => void }) {
  const { content, setContent, announce } = useTypePreview();
  const formId = useId();
  const affected = content.items.filter((item) => item.type === type.id);
  const choices = content.types.filter(
    (item) => item.id !== type.id && item.active && (!content.hierarchyEnabled || item.level === type.level)
  );
  const form = useForm({
    defaultValues: { replacement: "" },
    validators: {
      onSubmit: ({ value }) => {
        if (affected.length && !value.replacement) return "Choose a compatible replacement type.";
        if (value.replacement && !choices.some((item) => item.id === value.replacement))
          return "Choose an active replacement at the same hierarchy level.";
        return undefined;
      },
    },
    onSubmit: ({ value }) => {
      const nextType = content.types.find((item) => item.id === value.replacement);
      setContent((current) => ({
        ...current,
        types: current.types.filter((item) => item.id !== type.id),
        projects: current.projects.map((project) => ({
          ...project,
          types: [
            ...new Set(
              project.types
                .filter((id) => id !== type.id)
                .concat(nextType && project.types.includes(type.id) ? [nextType.id] : [])
            ),
          ],
        })),
        items: current.items.map((item) =>
          item.type !== type.id || !nextType
            ? item
            : {
                ...preserveProperties(
                  item,
                  current.properties.filter(
                    (property) => type.properties.includes(property.id) && !nextType.properties.includes(property.id)
                  ),
                  `Type changed from ${type.name} to ${nextType.name}`
                ),
                type: nextType.id,
              }
        ),
      }));
      announce(`${type.name} deleted.`);
      onClose();
    },
  });
  const replacement = useStore(form.store, (state) => state.values.replacement);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  return (
    <PreviewDialog
      title={`Delete ${type.name}?`}
      description={`${affected.length} work items need a replacement type.`}
      onClose={() => {
        if (!isSubmitting) onClose();
      }}
      footer={
        <>
          <Button variant="secondary" disabled={isSubmitting} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" form={formId} variant="error-fill" disabled={isSubmitting}>
            Delete type
          </Button>
        </>
      }
    >
      <Form id={formId} onSubmit={() => form.handleSubmit()} className="space-y-4 text-13 text-secondary">
        <p>Work items are kept. Properties exclusive to the old type move into their descriptions.</p>
        {affected.length > 0 && (
          <label className="block">
            Replacement type
            <select
              className="mt-2 block w-full rounded border border-strong bg-surface-1 p-2"
              value={replacement}
              onChange={(event) => form.setFieldValue("replacement", event.target.value)}
            >
              <option value="">Choose a compatible type</option>
              {choices.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
          </label>
        )}
        {!choices.length && !!affected.length && (
          <p role="alert">No active type is available at this level. Move its work items before deletion.</p>
        )}
        {formError && <p role="alert">{formError}</p>}
      </Form>
    </PreviewDialog>
  );
}

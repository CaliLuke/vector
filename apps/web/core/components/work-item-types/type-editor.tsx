import { typeErrorMessage } from "@/services/work-item-types.service";
import { useForm, useStore } from "@tanstack/react-form";
import { Button } from "@plane/propel/button";
import { Form } from "@plane/ui";
import { inputStyle, PreviewDialog } from "./controls";
import { TypeIconPicker } from "./type-icon-picker";
import type { PreviewType } from "./preview-content";
import { useTypePreview } from "./preview-provider";

export function TypeEditor({
  type,
  onClose,
  types,
  onSave,
}: {
  type?: PreviewType;
  onClose: () => void;
  types?: PreviewType[];
  onSave?: (type: PreviewType, creating: boolean) => Promise<void>;
}) {
  const { content, setContent, announce } = useTypePreview();
  const form = useForm({
    defaultValues:
      type ??
      ({
        id: "",
        name: "",
        description: "",
        icon: "task",
        color: "#8b5cf6",
        active: false,
        mandatory: false,
        properties: [],
        level: 0,
      } satisfies PreviewType),
    validators: {
      onSubmit: ({ value }) => {
        const name = value.name.trim();
        if (!name) return "Enter a name for this work item type.";
        if (
          (types ?? content.types).some(
            (item) => item.id !== type?.id && item.name.toLowerCase() === name.toLowerCase()
          )
        )
          return "A work item type with this name already exists.";
        return undefined;
      },
    },
    onSubmit: async ({ value }) => {
      form.setErrorMap({ onSubmit: undefined });
      const name = value.name.trim();
      const saved = { ...value, name, id: type?.id ?? crypto.randomUUID() };
      if (onSave) {
        try {
          await onSave(saved, !type);
        } catch (failure) {
          form.setErrorMap({ onSubmit: { form: typeErrorMessage(failure), fields: {} } });
          throw failure;
        }
      } else
        setContent((current) => ({
          ...current,
          types: type ? current.types.map((item) => (item.id === type.id ? saved : item)) : [...current.types, saved],
        }));
      announce(type ? `${name} updated.` : `${name} created. Turn it on when you're ready to use it.`);
      onClose();
    },
  });
  const values = useStore(form.store, (state) => state.values);
  const submitting = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  return (
    <PreviewDialog
      title={type ? "Edit work item type" : "Create work item type"}
      onClose={() => {
        if (!submitting) onClose();
      }}
      footer={
        <>
          <Button variant="secondary" disabled={submitting} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" form="type-editor" disabled={submitting}>
            {type ? "Update work item type" : "Add work item type"}
          </Button>
        </>
      }
    >
      <Form
        id="type-editor"
        onSubmit={async () => {
          form.setErrorMap({ onSubmit: undefined });
          await form.handleSubmit().catch(() => {});
        }}
        className="space-y-3"
      >
        <div className="flex items-center gap-3">
          <TypeIconPicker
            value={values}
            disabled={submitting}
            onChange={({ icon, color }) => {
              form.setFieldValue("icon", icon);
              form.setFieldValue("color", color);
            }}
          />
          <input
            aria-label="Name"
            data-autofocus
            maxLength={255}
            className={inputStyle}
            placeholder="Name"
            value={values.name}
            disabled={submitting}
            onChange={(event) => form.setFieldValue("name", event.target.value)}
            aria-invalid={!!error}
          />
        </div>
        {error && (
          <p role="alert" className="text-13 text-danger-primary">
            {error}
          </p>
        )}
        <textarea
          aria-label="Description"
          rows={4}
          maxLength={10000}
          className={inputStyle}
          placeholder="Description"
          value={values.description}
          disabled={submitting}
          onChange={(event) => form.setFieldValue("description", event.target.value)}
        />
      </Form>
    </PreviewDialog>
  );
}

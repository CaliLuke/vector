import { FormulaEditor, type FormulaPersistence } from "./formula-editor";
import { PropertyDefaultInput } from "./property-default-input";
import { useState } from "react";
import { typeErrorMessage } from "@/services/work-item-types.service";
import { useForm, useStore } from "@tanstack/react-form";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@plane/propel/button";
import { Form, ToggleSwitch } from "@plane/ui";
import { Choice, Field, inputStyle, PreviewDialog, PropertyIcon } from "./controls";
import { newProperty, propertyKinds } from "./preview-content";
import type { PreviewProperty, PropertyKind, PreviewContent } from "./preview-content";
import { useTypePreview } from "./preview-provider";
export function PropertyEditor({
  property,
  onClose,
  library,
  members,
  onSave,
  formula,
}: {
  property?: PreviewProperty;
  onClose: () => void;
  library?: Pick<PreviewContent, "types" | "properties"> & Partial<Pick<PreviewContent, "projects">>;
  formula?: FormulaPersistence;
  members?: { id: string; name: string }[];
  onSave?: (property: PreviewProperty, creating: boolean) => Promise<void>;
}) {
  const { content: previewContent, setContent, announce } = useTypePreview();
  const content = library ?? previewContent;
  const [initial] = useState(
    () =>
      property ?? {
        ...newProperty(),
        id: crypto.randomUUID(),
        levels: newProperty().levels.map((level) => ({
          id: crypto.randomUUID(),
          name: level.name,
          required: level.required,
        })),
      }
  );
  const form = useForm({
    defaultValues: initial,
    validators: {
      onSubmit: ({ value }) => {
        const name = value.name.trim();
        if (!name) return "Enter a property title.";
        if (["state", "due date", "cycle", "modules"].includes(name.toLowerCase()))
          return "This name is reserved for a standard work item property.";
        const attachedTypes = content.types.filter((type) => type.properties.includes(value.id));
        if (
          attachedTypes.some((type) =>
            content.properties.some(
              (other) =>
                other.id !== value.id &&
                type.properties.includes(other.id) &&
                other.name.toLowerCase() === name.toLowerCase()
            )
          )
        )
          return "A property with this title is already attached to one of these types.";
        if (value.kind === "Dropdown" && (!value.options.length || value.options.some((option) => !option.name.trim())))
          return "Add at least one named option.";
        if (value.kind === "Formula" && !value.expression.trim()) return "Enter a formula expression.";
        return undefined;
      },
    },
    onSubmit: async ({ value }) => {
      const saved = {
        ...value,
        id: value.id,
        name: value.name.trim(),
      };
      if (onSave) await onSave(saved, !property);
      else
        setContent((current) => ({
          ...current,
          properties: property
            ? current.properties.map((item) => (item.id === property.id ? saved : item))
            : [...current.properties, saved],
        }));
      announce(`${saved.name} ${property ? "updated" : "created"}.`);
      onClose();
    },
  });
  const value = useStore(form.store, (state) => state.values);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const submitting = useStore(form.store, (state) => state.isSubmitting);
  const cannotRequire =
    value.kind === "Boolean" ||
    value.kind === "Formula" ||
    value.kind === "Cascading" ||
    (value.kind === "Text" && value.format === "Read-only");
  const supportsDefault = ["Number", "Dropdown", "Member picker"].includes(value.kind);
  const changeKind = (kind: PropertyKind) => {
    const kindDefaults = newProperty(kind);
    form.setFieldValue("kind", kind);
    form.setFieldValue("required", false);
    form.setFieldValue("defaultValue", "");
    form.setFieldValue("format", kind === "Date" ? "Jan 15, 2025" : kindDefaults.format);
    form.setFieldValue("multiple", false);
    form.setFieldValue("options", []);
    form.setFieldValue(
      "levels",
      kind === "Cascading"
        ? kindDefaults.levels.map((level) => ({ id: crypto.randomUUID(), name: level.name, required: level.required }))
        : []
    );
    form.setFieldValue("expression", "");
    form.setFieldValue("result", kind === "Formula" ? "Number" : "");
  };
  const renderTextAndSelectionSettings = () => (
    <>
      {value.kind === "Text" && (
        <>
          <Field label="Text format" hint={property ? "The text format cannot change after creation." : undefined}>
            {() => (
              <Choice
                label="Text format"
                value={value.format}
                options={["Single line", "Paragraph", "Read-only"]}
                disabled={!!property}
                onChange={(next) => {
                  form.setFieldValue("format", next);
                  if (next === "Read-only") form.setFieldValue("required", false);
                }}
              />
            )}
          </Field>
          {value.format === "Read-only" && (
            <Field label="Fixed text">
              {(id) => (
                <textarea
                  id={id}
                  className={inputStyle}
                  rows={3}
                  value={value.defaultValue}
                  onChange={(event) => form.setFieldValue("defaultValue", event.target.value)}
                />
              )}
            </Field>
          )}
        </>
      )}
      {["Dropdown", "Member picker", "Release picker"].includes(value.kind) && (
        <fieldset className="space-y-2">
          <legend className="mb-2 text-13 font-medium text-primary">Selection</legend>
          <div className="flex gap-6">
            {[false, true].map((multiple) => (
              <label key={String(multiple)} className="flex items-center gap-2 text-13">
                <input
                  type="radio"
                  name="selection-mode"
                  checked={value.multiple === multiple}
                  disabled={property?.used}
                  onChange={() => form.setFieldValue("multiple", multiple)}
                />
                {multiple ? "Multi select" : "Single select"}
              </label>
            ))}
          </div>
        </fieldset>
      )}
      {value.kind === "Dropdown" && (
        <div className="space-y-3">
          <p className="text-13 font-medium">Options</p>
          {value.options.map((option, index) => (
            <div key={option.id} className="flex items-center gap-2">
              <input
                aria-label={`Option ${index + 1}`}
                className={inputStyle}
                disabled={property?.used}
                value={option.name}
                onChange={(event) =>
                  form.setFieldValue(
                    "options",
                    value.options.map((item) =>
                      item.id === option.id
                        ? {
                            ...item,
                            name: event.target.value,
                          }
                        : item
                    )
                  )
                }
              />
              <select
                aria-label={`Parent of option ${index + 1}`}
                className={`${inputStyle} max-w-36`}
                disabled={property?.used}
                value={option.parent}
                onChange={(event) =>
                  form.setFieldValue(
                    "options",
                    value.options.map((item) =>
                      item.id === option.id
                        ? {
                            ...item,
                            parent: event.target.value,
                          }
                        : item
                    )
                  )
                }
              >
                <option value="">No parent</option>
                {value.options.slice(0, index).map((parent) => (
                  <option key={parent.id} value={parent.id}>
                    {parent.name}
                  </option>
                ))}
              </select>
              <button
                type="button"
                aria-label={`Remove option ${index + 1}`}
                disabled={property?.used}
                onClick={() =>
                  form.setFieldValue(
                    "options",
                    value.options.filter((item) => item.id !== option.id)
                  )
                }
                className="p-1 text-tertiary disabled:opacity-40"
              >
                <Trash2 className="size-4" />
              </button>
            </div>
          ))}
          <Button
            variant="secondary"
            disabled={property?.used}
            prependIcon={<Plus />}
            onClick={() =>
              form.setFieldValue("options", [
                ...value.options,
                {
                  id: crypto.randomUUID(),
                  name: "",
                  parent: "",
                  level: 0,
                  nextLabel: "",
                },
              ])
            }
          >
            Add option
          </Button>
        </div>
      )}
      {value.kind === "Date" && (
        <Field label="Display format">
          {() => (
            <Choice
              label="Display format"
              value={value.format}
              options={["Jan 15, 2025", "15/01/2025", "01/15/2025", "2025/01/15"]}
              onChange={(next) => form.setFieldValue("format", next)}
            />
          )}
        </Field>
      )}
    </>
  );
  const renderCascadeAndFormulaSettings = () => (
    <>
      {value.kind === "Cascading" && (
        <div className="space-y-3">
          <p className="text-13 text-secondary">
            Choices narrow from left to right. Configure options after creating the property.
          </p>
          {value.levels.map((level, index) => (
            <div key={level.id} className="flex items-center gap-3">
              <input
                aria-label={`Level ${index + 1} name`}
                maxLength={100}
                className={inputStyle}
                value={level.name}
                onChange={(event) =>
                  form.setFieldValue(
                    "levels",
                    value.levels.map((item, i) =>
                      i === index
                        ? {
                            ...item,
                            name: event.target.value,
                          }
                        : item
                    )
                  )
                }
              />
              <label className="flex shrink-0 items-center gap-2 text-12">
                <input
                  type="checkbox"
                  checked={level.required}
                  onChange={(event) =>
                    form.setFieldValue(
                      "levels",
                      value.levels.map((item, i) => ({
                        ...item,
                        required: event.target.checked
                          ? i <= index || item.required
                          : i >= index
                            ? false
                            : item.required,
                      }))
                    )
                  }
                />
                Required
              </label>
            </div>
          ))}
          <Button
            variant="secondary"
            disabled={value.levels.length === 4}
            onClick={() =>
              form.setFieldValue("levels", [
                ...value.levels,
                {
                  id: crypto.randomUUID(),
                  name: `Level ${value.levels.length + 1}`,
                  required: false,
                },
              ])
            }
          >
            Add level
          </Button>
        </div>
      )}
      {value.kind === "Formula" && (
        <FormulaEditor
          property={value}
          library={{ ...content, projects: content.projects ?? [] }}
          persistence={formula}
          onChange={(expression, result) => {
            form.setFieldValue("expression", expression);
            form.setFieldValue("result", result);
          }}
        />
      )}
      {supportsDefault && (
        <Field
          label="Default value"
          hint={
            value.required ? "Mandatory properties cannot have a default value." : "Used when creating a new work item."
          }
        >
          {(id) => (
            <PropertyDefaultInput
              id={id}
              property={value}
              members={members}
              value={value.defaultValue}
              onChange={(defaultValue) => form.setFieldValue("defaultValue", defaultValue)}
            />
          )}
        </Field>
      )}
      {value.kind === "Member picker" && (
        <p className="text-12 text-tertiary">Selected members are subscribed to the work item when they have access.</p>
      )}
    </>
  );
  const renderDefaultSettings = () => (
    <>
      {value.kind === "Release picker" && (
        <p className="text-12 text-tertiary">
          This field stores release references. It does not add the work item to release scope.
        </p>
      )}
      {value.kind === "Rich text" && (
        <p className="text-12 text-tertiary">
          A document editor with version history. Content is searchable; this field cannot be filtered or grouped.
        </p>
      )}
    </>
  );
  return (
    <PreviewDialog
      title={property ? "Edit property" : "Create property"}
      onClose={() => {
        if (!submitting) onClose();
      }}
      footer={
        <>
          <label className="mr-auto flex items-center gap-2 text-13 text-secondary">
            <input
              type="checkbox"
              disabled={submitting || cannotRequire}
              checked={!cannotRequire && value.required}
              onChange={(event) => {
                form.setFieldValue("required", event.target.checked);
                if (event.target.checked) form.setFieldValue("defaultValue", "");
              }}
            />
            Mandatory property
          </label>
          <Button variant="secondary" disabled={submitting} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" disabled={submitting} form="property-editor">
            {property ? "Save changes" : "Create"}
          </Button>
        </>
      }
    >
      <Form
        id="property-editor"
        onSubmit={async () => {
          form.setErrorMap({ onSubmit: undefined });
          await form.handleSubmit().catch((cause: unknown) => {
            form.setErrorMap({
              onSubmit: {
                form: typeErrorMessage(cause),
                fields: {},
              },
            });
          });
        }}
        className="space-y-5"
      >
        <fieldset disabled={submitting} className="space-y-5">
          <Field label="Title">
            {(id) => (
              <input
                id={id}
                className={inputStyle}
                value={value.name}
                placeholder="e.g. Severity"
                onChange={(event) => form.setFieldValue("name", event.target.value)}
              />
            )}
          </Field>
          <Field label="Description">
            {(id) => (
              <textarea
                id={id}
                rows={2}
                className={inputStyle}
                placeholder="Help people understand this property"
                value={value.description}
                onChange={(event) => form.setFieldValue("description", event.target.value)}
              />
            )}
          </Field>
          <Field
            label="Property type"
            hint={property?.used ? "This property has values. Its type and selection mode are locked." : undefined}
          >
            {() => (
              <div className="flex items-center gap-2">
                <PropertyIcon kind={value.kind} />
                <Choice
                  label="Property type"
                  value={value.kind}
                  options={[...propertyKinds]}
                  disabled={property?.used}
                  onChange={(next) => {
                    const kind = propertyKinds.find((candidate) => candidate === next);
                    if (kind) changeKind(kind);
                  }}
                />
              </div>
            )}
          </Field>
          {renderTextAndSelectionSettings()}

          {renderCascadeAndFormulaSettings()}

          {renderDefaultSettings()}

          <div className="flex items-center justify-between border-t border-subtle pt-4">
            <div>
              <p className="text-13 font-medium">Active</p>
              <p className="text-12 text-tertiary">Show this property on work items.</p>
            </div>
            <ToggleSwitch
              label="Property active"
              value={value.active}
              onChange={(next) => form.setFieldValue("active", next)}
            />
          </div>
        </fieldset>
        {formError && (
          <p role="alert" className="text-13 text-danger-primary">
            {formError}
          </p>
        )}
      </Form>
    </PreviewDialog>
  );
}

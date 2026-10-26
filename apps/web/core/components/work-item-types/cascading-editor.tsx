import { propertyToSave } from "./property-adapter";
import { usePropertyEditReview, type PropertyEditPersistence } from "./property-edit-review";
import { typeErrorMessage } from "@/services/work-item-types.service";
import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { Plus, Trash2 } from "lucide-react";
import { Button } from "@plane/propel/button";
import { Form } from "@plane/ui";
import { Confirmation, inputStyle, PreviewDialog } from "./controls";
import type { PreviewOption, PreviewProperty } from "./preview-content";
import { useTypePreview } from "./preview-provider";
import { effectiveProperty, preserveProperties, removedCascadeValues } from "./preview-actions";

export function CascadingEditor({
  property,
  onClose,
  onSave,
  review,
}: {
  property: PreviewProperty;
  onClose: () => void;
  onSave?: (property: PreviewProperty) => Promise<void>;
  review?: PropertyEditPersistence;
}) {
  const { content, setContent, announce } = useTypePreview();
  const [context, setContext] = useState("default");
  const [path, setPath] = useState<string[]>([]);
  const [confirm, setConfirm] = useState(false);
  const [reviewDialog, setReviewing] = usePropertyEditReview(property, review, onClose);
  const form = useForm({
    defaultValues: structuredClone(property),
    validators: {
      onSubmit: ({ value }) => {
        if (value.levels.length < 2 || value.levels.length > 4) return "Choose between two and four levels.";
        if (value.levels.some((level) => !level.name.trim())) return "Give every level a name.";
        if (value.options.some((option) => !option.name.trim())) return "Give every option a name.";
        if (value.contexts.some((item) => item.options?.some((option) => !option.name.trim())))
          return "Give every option a name.";
        return undefined;
      },
    },
    onSubmit: ({ value }) => save(value),
  });
  const draft = useStore(form.store, (state) => state.values);
  const affected = onSave || review ? [] : removedCascadeValues(content, property, draft);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const options =
    context === "default"
      ? draft.options
      : (draft.contexts.find((item) => item.id === context)?.options ?? draft.options);
  const inheritsOptions =
    context !== "default" && draft.contexts.find((item) => item.id === context)?.options === undefined;
  const setOptions = (next: PreviewOption[]) => {
    if (inheritsOptions) return;
    if (context === "default") form.setFieldValue("options", next);
    else
      form.setFieldValue(
        "contexts",
        draft.contexts.map((item) => (item.id === context ? { ...item, options: next } : item))
      );
  };
  async function save(saved: PreviewProperty) {
    if (review) {
      setReviewing(propertyToSave(saved));
      return;
    }
    if (onSave) {
      await onSave(saved);
      announce(`${property.name} options updated.`);
      onClose();
      return;
    }
    const affectedAtSubmit = removedCascadeValues(content, property, saved);
    setContent((current) => ({
      ...current,
      properties: current.properties.map((item) => (item.id === saved.id ? saved : item)),
      items: current.items.map((item) =>
        affectedAtSubmit.some((other) => item.id === other.id)
          ? preserveProperties(
              item,
              [effectiveProperty(property, item.project, item.type)],
              "Cascading options removed"
            )
          : item
      ),
    }));
    announce(`${property.name} options updated.`);
    onClose();
  }
  const removeOption = (id: string) => {
    const children = new Map<string, string[]>();
    for (const option of options) children.set(option.parent, [...(children.get(option.parent) ?? []), option.id]);
    const ids = new Set<string>();
    const pending = [id];
    while (pending.length) {
      const current = pending.pop()!;
      if (ids.has(current)) continue;
      ids.add(current);
      pending.push(...(children.get(current) ?? []));
    }
    setOptions(options.filter((option) => !ids.has(option.id)));
    setPath((current) => current.filter((item) => !ids.has(item)));
  };
  if (confirm)
    return (
      <Confirmation
        title="Remove cascading values?"
        token="Confirm"
        action="Remove and preserve values"
        onClose={() => setConfirm(false)}
        onConfirm={async () => {
          await form.handleSubmit();
          if (!form.state.isSubmitSuccessful)
            throw new Error(form.state.errors[0] ?? "Review the cascading field errors before confirming.");
        }}
      >
        <p>
          {affected.length} work items use removed options. Their values will move into the description. Removing a
          level also removes all levels below it.
        </p>
      </Confirmation>
    );
  if (reviewDialog) return reviewDialog;
  return (
    <PreviewDialog
      wide
      title={`Manage levels and options · ${property.name}`}
      onClose={() => {
        if (!isSubmitting) onClose();
      }}
      footer={
        <>
          <Button variant="secondary" disabled={isSubmitting} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" form="cascading-options-form" disabled={isSubmitting}>
            Save changes
          </Button>
        </>
      }
    >
      <Form
        id="cascading-options-form"
        onSubmit={async () => {
          form.setErrorMap({ onSubmit: undefined });
          if (!onSave && !review && removedCascadeValues(content, property, draft).length) {
            await form.validate("submit");
            if (!form.state.isValid) return;
            setConfirm(true);
            return;
          }
          await form.handleSubmit().catch((cause: unknown) => {
            form.setErrorMap({ onSubmit: { form: typeErrorMessage(cause), fields: {} } });
          });
        }}
      >
        <fieldset disabled={isSubmitting} className="space-y-4">
          <label className="mb-5 block text-13">
            Context
            <select
              className={`${inputStyle} mt-2`}
              value={context}
              onChange={(event) => {
                setContext(event.target.value);
                setPath([]);
              }}
            >
              <option value="default">Default · all projects and types</option>
              {draft.contexts.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
          </label>
          {context !== "default" && (
            <label className="flex items-center gap-2 text-13">
              <input
                type="checkbox"
                checked={inheritsOptions}
                onChange={(event) => {
                  form.setFieldValue(
                    "contexts",
                    draft.contexts.map((item) =>
                      item.id === context
                        ? { ...item, options: event.target.checked ? undefined : structuredClone(draft.options) }
                        : item
                    )
                  );
                  setPath([]);
                }}
              />
              Use base options
            </label>
          )}
          {inheritsOptions && (
            <p className="text-12 text-secondary">
              Base options are read-only here. Turn off Use base options to configure this context's choices.
            </p>
          )}
          <div className="flex gap-3 overflow-x-auto pb-3">
            {draft.levels.map((level, index) => (
              <div key={level.id} className="w-48 shrink-0 rounded-lg border border-subtle">
                <div className="space-y-2 border-b border-subtle bg-layer-1 p-3">
                  <input
                    aria-label={`Level ${index + 1} name`}
                    className={`${inputStyle} font-medium`}
                    maxLength={100}
                    value={level.name}
                    onChange={(event) =>
                      form.setFieldValue(
                        "levels",
                        draft.levels.map((item, i) => (i === index ? { ...item, name: event.target.value } : item))
                      )
                    }
                  />
                  <div className="flex items-center justify-between">
                    <label className="flex items-center gap-2 text-11">
                      <input
                        type="checkbox"
                        checked={level.required}
                        onChange={(event) =>
                          form.setFieldValue(
                            "levels",
                            draft.levels.map((item, i) => ({
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
                    {index >= 2 && (
                      <button
                        type="button"
                        aria-label={`Delete ${level.name} and following levels`}
                        onClick={() => {
                          form.setFieldValue("levels", draft.levels.slice(0, index));
                          setPath((current) => current.slice(0, index));
                          form.setFieldValue(
                            "options",
                            draft.options.filter((option) => option.level < index)
                          );
                          form.setFieldValue(
                            "contexts",
                            draft.contexts.map((item) => ({
                              ...item,
                              options: item.options?.filter((option) => option.level < index),
                            }))
                          );
                        }}
                      >
                        <Trash2 className="size-3.5 text-tertiary" />
                      </button>
                    )}
                  </div>
                </div>
                <div className="min-h-40 space-y-2 p-2">
                  {index > 0 && !path[index - 1] ? (
                    <p className="p-2 text-12 text-tertiary">Choose {draft.levels[index - 1].name} first.</p>
                  ) : (
                    <>
                      {options
                        .filter((option) => option.level === index && option.parent === (path[index - 1] ?? ""))
                        .map((option) => (
                          <div
                            key={option.id}
                            className={`rounded-md border p-2 ${path[index] === option.id ? "border-accent-strong bg-accent-primary/5" : "border-subtle"}`}
                          >
                            <div className="flex gap-1">
                              <input
                                aria-label={`Option ${option.name || "name"}`}
                                className="min-w-0 flex-1 bg-transparent text-12 outline-none"
                                readOnly={inheritsOptions}
                                value={option.name}
                                onFocus={() => setPath((current) => [...current.slice(0, index), option.id])}
                                onChange={(event) =>
                                  setOptions(
                                    options.map((item) =>
                                      item.id === option.id ? { ...item, name: event.target.value } : item
                                    )
                                  )
                                }
                              />
                              <button
                                type="button"
                                aria-label={`Delete ${option.name}`}
                                disabled={inheritsOptions}
                                onClick={() => removeOption(option.id)}
                              >
                                <Trash2 className="size-3 text-tertiary" />
                              </button>
                            </div>
                            {index < draft.levels.length - 1 && (
                              <input
                                aria-label={`Next field label for ${option.name}`}
                                className="mt-2 w-full border-t border-subtle bg-transparent pt-2 text-11 text-tertiary outline-none"
                                placeholder="Next field label (optional)"
                                readOnly={inheritsOptions}
                                value={option.nextLabel}
                                onChange={(event) =>
                                  setOptions(
                                    options.map((item) =>
                                      item.id === option.id ? { ...item, nextLabel: event.target.value } : item
                                    )
                                  )
                                }
                              />
                            )}
                          </div>
                        ))}
                      <Button
                        type="button"
                        variant="ghost"
                        prependIcon={<Plus />}
                        disabled={inheritsOptions}
                        onClick={() =>
                          setOptions([
                            ...options,
                            {
                              id: crypto.randomUUID(),
                              name: "New option",
                              parent: path[index - 1] ?? "",
                              level: index,
                              nextLabel: "",
                            },
                          ])
                        }
                      >
                        Add option
                      </Button>
                    </>
                  )}
                </div>
              </div>
            ))}
          </div>
          <Button
            type="button"
            className="mt-4"
            variant="secondary"
            disabled={draft.levels.length >= 4}
            prependIcon={<Plus />}
            onClick={() =>
              form.setFieldValue("levels", [
                ...draft.levels,
                { id: crypto.randomUUID(), name: `Level ${draft.levels.length + 1}`, required: false },
              ])
            }
          >
            Add level
          </Button>
          <p className="mt-3 text-12 text-tertiary">
            Two to four levels. Levels cannot be reordered. Select a parent option to manage its children.
          </p>
          {formError && (
            <p role="alert" className="text-12 text-danger-primary">
              {formError}
            </p>
          )}
        </fieldset>
      </Form>
    </PreviewDialog>
  );
}

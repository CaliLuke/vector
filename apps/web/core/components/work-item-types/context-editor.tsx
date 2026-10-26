import { PropertyDefaultInput, propertyDefaultLabel } from "./property-default-input";
import { ContextResolutionPreview } from "./context-resolution-preview";
import { propertyToSave } from "./property-adapter";
import { usePropertyEditReview, type PropertyEditPersistence } from "./property-edit-review";
import { typeErrorMessage } from "@/services/work-item-types.service";
import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { Plus, Globe2 } from "lucide-react";
import { Button } from "@plane/propel/button";
import { Form } from "@plane/ui";
import { Choice, Field, inputStyle, PreviewDialog } from "./controls";
import type { PreviewContent, PreviewContext, PreviewProperty } from "./preview-content";
import { useTypePreview } from "./preview-provider";
import { contextOverlap } from "./preview-actions";
const emptyContext = (): PreviewContext => ({
  id: "",
  name: "",
  projects: [],
  types: [],
  required: null,
  active: null,
  defaultValue: null,
  format: null,
});
function savedContext(contexts: PreviewContext[], selected: string) {
  return contexts.find((context) => context.id === selected) ?? emptyContext();
}
const UNAPPLIED_CONTEXT_ERROR = "Apply this context before saving.";
export function ContextEditor({
  property,
  onClose,
  library,
  members,
  onSave,
  review,
}: {
  property: PreviewProperty;
  onClose: () => void;
  library?: Pick<PreviewContent, "types" | "projects">;
  members?: { id: string; name: string }[];
  onSave?: (property: PreviewProperty) => Promise<void>;
  review?: PropertyEditPersistence;
}) {
  const { content: previewContent, setContent, announce } = useTypePreview();
  const content = library ?? previewContent;
  const [selected, setSelected] = useState("default");
  const [navigationError, setNavigationError] = useState("");
  const [reviewDialog, setReviewing] = usePropertyEditReview(property, review, onClose);
  const form = useForm({
    defaultValues: { contexts: property.contexts, activeContext: emptyContext() },
    onSubmitMeta: { action: "apply" as "apply" | "save" },
    validators: {
      onSubmit: ({ value }) => {
        if (selected === "default") return undefined;
        const active = value.activeContext;
        if (!active.name.trim()) return "Give this context a name.";
        if (!active.projects.length && !active.types.length)
          return "Choose a project or type. The Default context already covers everything.";
        if (value.contexts.some((context) => context.id !== active.id && contextOverlap(context, active)))
          return "This scope overlaps another context at the same specificity. Narrow the projects or types.";
        return undefined;
      },
    },
    onSubmit: async ({ value, meta }) => {
      if (meta.action === "save") {
        const baseline = savedContext(value.contexts, selected);
        if (selected === "new" || JSON.stringify(value.activeContext) !== JSON.stringify(baseline))
          throw new Error(UNAPPLIED_CONTEXT_ERROR);
        if (review) {
          setReviewing(propertyToSave({ ...property, contexts: value.contexts }));
          return;
        }
        if (onSave) await onSave({ ...property, contexts: value.contexts });
        else
          setContent((current) => ({
            ...current,
            properties: current.properties.map((item) =>
              item.id === property.id ? { ...item, contexts: value.contexts } : item
            ),
          }));
        announce(`Contexts for ${property.name} updated.`);
        onClose();
        return;
      }
      const active = value.activeContext;
      const saved = {
        ...active,
        id: active.id || crypto.randomUUID(),
      };
      form.setFieldValue(
        "contexts",
        value.contexts.some((context) => context.id === saved.id)
          ? value.contexts.map((context) => (context.id === saved.id ? saved : context))
          : [...value.contexts, saved]
      );
      setSelected(saved.id);
      form.setFieldValue("activeContext", saved);
      setNavigationError("");
    },
  });
  const formValues = useStore(form.store, (state) => state.values);
  const value = formValues.activeContext;
  const contexts = formValues.contexts;
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const baseline = savedContext(contexts, selected);
  const dirty = JSON.stringify(value) !== JSON.stringify(baseline);
  const submit = async (action: "apply" | "save") => {
    form.setErrorMap({ onSubmit: undefined });
    try {
      await form.handleSubmit({ action });
    } catch (cause) {
      form.setErrorMap({ onSubmit: { form: typeErrorMessage(cause), fields: {} } });
    }
  };
  const choose = (context?: PreviewContext) => {
    if (dirty) {
      setNavigationError("Apply this context before switching. Your changes are still here.");
      return;
    }
    const next = context ?? emptyContext();
    setSelected(context?.id ?? "default");
    form.setFieldValue("activeContext", next);
    setNavigationError("");
  };
  const toggleScope = (field: "projects" | "types", id: string) => {
    form.setFieldValue("activeContext", {
      ...value,
      [field]: value[field].includes(id) ? value[field].filter((item) => item !== id) : [...value[field], id],
    });
  };
  const renderContextForm = () => (
    <div className="space-y-5">
      <Field label="Context name">
        {(id) => (
          <input
            id={id}
            className={inputStyle}
            placeholder="e.g. Mobile bugs"
            value={value.name}
            onChange={(event) => form.setFieldValue("activeContext", { ...value, name: event.target.value })}
          />
        )}
      </Field>
      <fieldset>
        <legend className="mb-2 text-13 font-medium">
          Projects{" "}
          <span className="font-normal text-tertiary">
            {value.projects.length ? `(${value.projects.length})` : "(All projects)"}
          </span>
        </legend>
        <button
          type="button"
          className="mb-2 text-12 text-accent-primary"
          onClick={() => form.setFieldValue("activeContext", { ...value, projects: [] })}
        >
          All projects
        </button>
        <div className="flex flex-wrap gap-3">
          {content.projects.map((project) => (
            <label key={project.id} className="flex items-center gap-2 text-13">
              <input
                type="checkbox"
                checked={value.projects.includes(project.id)}
                onChange={() => toggleScope("projects", project.id)}
              />
              {project.name}
            </label>
          ))}
        </div>
      </fieldset>
      <fieldset>
        <legend className="mb-2 text-13 font-medium">
          Work item types{" "}
          <span className="font-normal text-tertiary">
            {value.types.length ? `(${value.types.length})` : "(All types)"}
          </span>
        </legend>
        <button
          type="button"
          className="mb-2 text-12 text-accent-primary"
          onClick={() => form.setFieldValue("activeContext", { ...value, types: [] })}
        >
          All types
        </button>
        <div className="flex flex-wrap gap-3">
          {content.types.map((type) => (
            <label key={type.id} className="flex items-center gap-2 text-13">
              <input
                type="checkbox"
                checked={value.types.includes(type.id)}
                onChange={() => toggleScope("types", type.id)}
              />
              {type.name}
            </label>
          ))}
        </div>
      </fieldset>
      <div className="grid grid-cols-2 gap-4">
        {(["required", "active"] as const).map((setting) => (
          <Field key={setting} label={setting === "required" ? "Mandatory" : "Active"}>
            {() => (
              <Choice
                label={setting}
                disabled={
                  setting === "required" &&
                  (["Boolean", "Formula", "Cascading"].includes(property.kind) || property.format === "Read-only")
                }
                value={value[setting] === null ? "Use base setting" : value[setting] ? "Yes" : "No"}
                options={["Use base setting", "Yes", "No"]}
                onChange={(next) => {
                  form.setFieldValue("activeContext", {
                    ...value,
                    [setting]: next === "Use base setting" ? null : next === "Yes",
                    ...(setting === "required" && next === "Yes" ? { defaultValue: "" } : {}),
                  });
                }}
              />
            )}
          </Field>
        ))}
      </div>
      {["Number", "Dropdown", "Member picker"].includes(property.kind) && (
        <Field label="Default value" hint="Leave inheritance on to use the base default.">
          {(id) => (
            <div className="space-y-2">
              <label className="flex items-center gap-2 text-12">
                <input
                  type="checkbox"
                  checked={value.defaultValue == null}
                  onChange={(event) =>
                    form.setFieldValue("activeContext", { ...value, defaultValue: event.target.checked ? null : "" })
                  }
                />
                Use base setting
              </label>
              {value.defaultValue != null && (
                <PropertyDefaultInput
                  id={id}
                  property={{ ...property, required: value.required ?? property.required }}
                  members={members}
                  value={value.defaultValue}
                  onChange={(defaultValue) => form.setFieldValue("activeContext", { ...value, defaultValue })}
                />
              )}
            </div>
          )}
        </Field>
      )}
      {property.kind === "Date" && (
        <Field label="Date format">
          {() => (
            <Choice
              label="Date format"
              value={value.format ?? "Use base setting"}
              options={["Use base setting", "Jan 15, 2025", "15/01/2025", "01/15/2025", "2025/01/15"]}
              onChange={(next) => {
                form.setFieldValue("activeContext", { ...value, format: next === "Use base setting" ? null : next });
              }}
            />
          )}
        </Field>
      )}
      {property.kind === "Cascading" && (
        <p className="rounded bg-layer-1 p-3 text-12 text-secondary">
          Manage cascading options for this context in Manage levels and options. Until overridden, it inherits the base
          options.
        </p>
      )}
      <div className="rounded-md border border-subtle p-3 text-12 text-secondary">
        <strong>Effective settings</strong>
        <p className="mt-1">
          {(value.required ?? property.required) ? "Required" : "Optional"} ·{" "}
          {(value.active ?? property.active) ? "Visible" : "Hidden"} ·{" "}
          {value.projects.length
            ? value.projects.map((id) => content.projects.find((project) => project.id === id)?.name).join(", ")
            : "All projects"}
        </p>
      </div>
      <div className="flex justify-between gap-3">
        {value.id && (
          <Button
            variant="error-outline"
            onClick={() => {
              form.setFieldValue(
                "contexts",
                contexts.filter((context) => context.id !== value.id)
              );
              setSelected("default");
              form.setFieldValue("activeContext", emptyContext());
              setNavigationError("");
            }}
          >
            Remove context
          </Button>
        )}
        <Button type="button" variant="secondary" disabled={isSubmitting} onClick={() => void submit("apply")}>
          Apply context
        </Button>
      </div>
    </div>
  );
  if (reviewDialog) return reviewDialog;
  return (
    <PreviewDialog
      wide
      title={`Contexts · ${property.name}`}
      description="Change how this property behaves in specific projects and types."
      onClose={() => {
        if (!isSubmitting) onClose();
      }}
      footer={
        <>
          <Button variant="secondary" disabled={isSubmitting} onClick={onClose}>
            Cancel
          </Button>
          <Button
            type="button"
            disabled={dirty || selected === "new" || isSubmitting}
            onClick={() => void submit("save")}
          >
            Save changes
          </Button>
        </>
      }
    >
      <Form
        id="property-context-form"
        onSubmit={async () => {
          form.setErrorMap({ onSubmit: undefined });
          await form.handleSubmit();
        }}
      >
        <fieldset disabled={isSubmitting} className="grid gap-6 sm:grid-cols-[180px_1fr]">
          <nav aria-label="Property contexts" className="space-y-1">
            <button
              type="button"
              className={`flex w-full items-center gap-2 rounded-md px-3 py-2 text-left text-13 ${selected === "default" ? "bg-layer-2 font-medium" : "hover:bg-layer-1"}`}
              onClick={() => choose()}
            >
              <Globe2 className="size-4" />
              Default
            </button>
            {contexts.map((context) => (
              <div
                key={context.id}
                className={`rounded-md ${selected === context.id ? "bg-layer-2" : "hover:bg-layer-1"}`}
              >
                <button
                  type="button"
                  className="w-full px-3 pt-2 text-left text-13 font-medium"
                  aria-describedby={`context-summary-${context.id}`}
                  onClick={() => choose(context)}
                >
                  {context.name}
                </button>
                <p id={`context-summary-${context.id}`} className="px-3 pb-2 text-11 text-tertiary">
                  {contextSummary(context, content)}
                </p>
              </div>
            ))}
            <Button
              type="button"
              className="mt-3"
              variant="ghost"
              disabled={dirty}
              prependIcon={<Plus />}
              onClick={() => {
                choose();
                setSelected("new");
              }}
            >
              Add context
            </Button>
          </nav>
          {selected === "default" ? (
            <div className="space-y-5 text-13">
              <h3 className="font-medium">Default context</h3>
              <p className="text-secondary">All projects · All work item types</p>
              <p className="text-tertiary">
                The default context uses the property's base settings. More specific contexts override only the settings
                you choose.
              </p>
              <dl className="grid grid-cols-2 gap-3 rounded-md border border-subtle p-4">
                <dt>Required</dt>
                <dd>{property.required ? "Yes" : "No"}</dd>
                <dt>Active</dt>
                <dd>{property.active ? "Yes" : "No"}</dd>
                <dt>Default value</dt>
                <dd>{propertyDefaultLabel(property, members)}</dd>
              </dl>
              <p className="text-12 text-tertiary">
                Edit the property to change its base settings. Default cannot be deleted.
              </p>
            </div>
          ) : (
            renderContextForm()
          )}
        </fieldset>
        <ContextResolutionPreview
          property={{ ...property, contexts }}
          projects={content.projects}
          types={content.types}
          members={members}
        />
        {(formError || navigationError) && (
          <p role="alert" className="text-12 text-danger-primary">
            {formError || navigationError}
          </p>
        )}
      </Form>
    </PreviewDialog>
  );
}

function contextSummary(context: PreviewContext, content: Pick<PreviewContent, "projects" | "types">) {
  const overrides = (
    [
      ["required", "Mandatory"],
      ["active", "Active"],
      ["defaultValue", "Default"],
      ["format", "Date format"],
      ["options", "Options"],
    ] as const
  )
    .filter(([key]) => context[key] != null)
    .map(([, label]) => label);
  return `${scopeNames(context.projects, content.projects, "All projects")} · ${scopeNames(context.types, content.types, "All types")} · ${overrides.length ? overrides.join(", ") : "Inherits all settings"}`;
}

function scopeNames(ids: string[], choices: { id: string; name: string }[], all: string) {
  return ids.length
    ? ids.map((id) => choices.find((choice) => choice.id === id)?.name ?? "Unavailable").join(", ")
    : all;
}

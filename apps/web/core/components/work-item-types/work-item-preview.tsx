import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { ArrowLeft, ArrowLeftRight, List, Table2, Plus } from "lucide-react";
import { Button } from "@plane/propel/button";
import { CustomSearchSelect, Form } from "@plane/ui";
import { Confirmation, Field, inputStyle, PreviewDialog, PropertyIcon, TypeIcon } from "./controls";
import { useTypePreview } from "./preview-provider";
import { defaultPropertyValues, effectiveProperty, preserveProperties } from "./preview-actions";
import type { PreviewItem, PreviewProperty, PreviewContent, PreviewType } from "./preview-content";
import { PropertyInput } from "./property-input";
type PropertyDraftsByItem = Record<string, PreviewItem["values"]>;
const emptyPropertyDrafts = (): PropertyDraftsByItem => ({});

function requiredErrors(properties: PreviewProperty[], values: PreviewItem["values"], isDraft: boolean) {
  if (isDraft) return [];
  return properties.filter((property) => {
    const value = values[property.id];
    if (property.kind === "Cascading") {
      const path = Array.isArray(value) ? value : [];
      return property.levels.some(
        (level, index) =>
          level.required &&
          !path[index] &&
          property.options.some((option) => option.level === index && option.parent === (path[index - 1] ?? ""))
      );
    }
    return property.required && (value === undefined || value === "" || (Array.isArray(value) && !value.length));
  });
}
function requiredValuesMessage(properties: PreviewProperty[], values: PreviewItem["values"], isDraft: boolean) {
  const missing = requiredErrors(properties, values, isDraft);
  return missing.length ? `Fill in ${missing.map((property) => property.name).join(", ")} before saving.` : undefined;
}
function getItemProperties(content: PreviewContent, item?: PreviewItem, type?: PreviewType) {
  if (!item || !type) return [];
  return type.properties.flatMap((id) => {
    const property = content.properties.find((candidate) => candidate.id === id);
    if (!property) return [];
    const effective = effectiveProperty(property, item.project, item.type);
    return effective.active ? [effective] : [];
  });
}
export function WorkItemsPreview({ readOnly = false }: { readOnly?: boolean }) {
  const { content, setContent, announce } = useTypePreview();
  const [selected, setSelected] = useState<string | null>(null);
  const [editing, setEditing] = useState<PreviewItem | "new" | null>(null);
  const selectedItem = content.items.find((item) => item.id === selected);
  const currentType = content.types.find((type) => type.id === selectedItem?.type);
  const properties = getItemProperties(content, selectedItem, currentType);
  const form = useForm({
    defaultValues: { byItem: emptyPropertyDrafts() },
    validators: {
      onSubmit: ({ value }) => {
        if (!selectedItem) return undefined;
        return requiredValuesMessage(
          properties,
          value.byItem[selectedItem.id] ?? selectedItem.values,
          !!selectedItem.draft
        );
      },
    },
    onSubmit: ({ value }) => {
      if (!selectedItem) return;
      const values = value.byItem[selectedItem.id] ?? selectedItem.values;
      setContent((current) => ({
        ...current,
        items: current.items.map((row) =>
          row.id === selectedItem.id
            ? {
                ...row,
                values,
                activity: [`Alex Morgan updated custom properties · ${new Date().toISOString()}`, ...row.activity],
              }
            : row
        ),
      }));
      announce("Sample property values saved.");
    },
  });
  const saveError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const isSaving = useStore(form.store, (state) => state.isSubmitting);
  return (
    <>
      <div className="mb-6 flex items-center justify-between gap-3">
        <Button
          variant="ghost"
          prependIcon={<ArrowLeft />}
          disabled={!selectedItem}
          onClick={() => {
            form.setErrorMap({ onSubmit: undefined });
            setSelected(null);
          }}
        >
          Sample work items
        </Button>
        <Button disabled={readOnly} prependIcon={<Plus />} onClick={() => setEditing("new")}>
          Add work item
        </Button>
      </div>
      {selectedItem && currentType ? (
        <div className="grid gap-7 lg:grid-cols-[minmax(0,1fr)_300px]">
          <div>
            <div className="mb-3 flex flex-wrap items-center gap-2 text-12 text-tertiary">
              {selectedItem.parent && <span>{selectedItem.parent} /</span>}
              <TypeIcon type={currentType} />
              <span>{currentType.name}</span>
              <span>{selectedItem.id}</span>
              {selectedItem.draft && <span className="rounded bg-layer-2 px-2 py-1">Private draft</span>}
              <Button
                variant="ghost"
                disabled={readOnly}
                prependIcon={<ArrowLeftRight />}
                onClick={() => setEditing(selectedItem)}
              >
                Switch type
              </Button>
            </div>
            <h2 className="text-24 leading-tight font-semibold">{selectedItem.name}</h2>
            <div className="my-5 flex flex-wrap gap-4 text-13 text-secondary">
              <span className="rounded bg-layer-2 px-2 py-1">In progress</span>
              <span className="py-1">High priority</span>
              <span className="py-1">Alex Morgan</span>
            </div>
            <p className="min-h-32 text-14 leading-relaxed whitespace-pre-wrap text-secondary">
              {selectedItem.description}
            </p>
            <div className="mt-8 border-t border-subtle pt-5">
              <h3 className="mb-4 text-14 font-medium">Activity</h3>
              {selectedItem.activity.length ? (
                selectedItem.activity.map((entry) => (
                  <p key={entry} className="border-b border-subtle py-3 text-12 text-tertiary">
                    {entry}
                  </p>
                ))
              ) : (
                <p className="text-13 text-tertiary">No changes yet.</p>
              )}
            </div>
          </div>
          <aside className="space-y-6 border-t border-subtle pt-5 lg:border-t-0 lg:border-l lg:pt-0 lg:pl-5">
            <h3 className="text-14 font-medium">Properties</h3>
            <div className="space-y-3 text-13">
              <p className="font-medium">Project structure</p>
              <div className="flex justify-between text-tertiary">
                <span>Project</span>
                <span>{content.projects.find((project) => project.id === selectedItem.project)?.name}</span>
              </div>
              <div className="flex justify-between text-tertiary">
                <span>Parent</span>
                <span>{selectedItem.parent || "None"}</span>
              </div>
            </div>
            <Form
              onSubmit={async () => {
                await form.handleSubmit();
                if (typeof form.store.state.errorMap.onSubmit === "string")
                  announce(form.store.state.errorMap.onSubmit);
              }}
              className="space-y-5 border-t border-subtle pt-5"
            >
              <form.Field name="byItem">
                {(field) => {
                  const values = field.state.value[selectedItem.id] ?? selectedItem.values;
                  return (
                    <>
                      <h4 className="text-13 font-medium">Custom properties</h4>
                      {properties.map((property) => (
                        <div key={`${selectedItem.id}-${property.id}`} className="space-y-2">
                          <div className="flex items-center gap-2">
                            <PropertyIcon kind={property.kind} icon={property.icon} />
                            <span className="text-12 text-secondary">
                              {property.name}
                              {property.required && <span className="ml-1 text-danger-primary">*</span>}
                            </span>
                          </div>
                          <PropertyInput
                            property={property}
                            value={values[property.id]}
                            draft={selectedItem.draft}
                            disabled={readOnly}
                            onChange={(value) =>
                              field.handleChange({
                                ...field.state.value,
                                [selectedItem.id]: {
                                  ...values,
                                  [property.id]: value,
                                },
                              })
                            }
                          />
                        </div>
                      ))}
                      {saveError && (
                        <p role="alert" className="text-12 text-danger-primary">
                          {saveError}
                        </p>
                      )}
                      <Button type="submit" variant="secondary" disabled={readOnly || isSaving}>
                        Save sample values
                      </Button>
                    </>
                  );
                }}
              </form.Field>
            </Form>
          </aside>
        </div>
      ) : (
        <WorkItemList
          content={content}
          onSelect={(id) => {
            form.setErrorMap({ onSubmit: undefined });
            setSelected(id);
          }}
        />
      )}
      {editing && (
        <WorkItemEditor
          item={editing === "new" ? undefined : editing}
          onClose={() => setEditing(null)}
          onSaved={(id) => {
            setSelected(id);
            form.setErrorMap({ onSubmit: undefined });
            form.setFieldValue("byItem", (current) => {
              const next = { ...current };
              delete next[id];
              return next;
            });
            setEditing(null);
          }}
        />
      )}
    </>
  );
}
function WorkItemEditor({
  item,
  onClose,
  onSaved,
}: {
  item?: PreviewItem;
  onClose: () => void;
  onSaved: (id: string) => void;
}) {
  const { content, setContent, announce } = useTypePreview();
  const [pending, setPending] = useState<PreviewItem | null>(null);
  const commit = (saved: PreviewItem) => {
    setContent((current) => ({
      ...current,
      items: item ? current.items.map((row) => (row.id === item.id ? saved : row)) : [...current.items, saved],
    }));
    announce(
      saved.draft ? "Draft preview saved locally. This is not a published work item." : "Sample work item saved."
    );
    onSaved(saved.id);
  };
  const form = useForm({
    defaultValues: item
      ? { ...item, draft: !!item.draft }
      : ({
          id: "",
          name: "",
          project: content.projects[0].id,
          type: "task",
          draft: false,
          parent: "",
          description: "",
          values: defaultPropertyValues(content, content.projects[0].id, "task"),
          activity: [],
        } satisfies PreviewItem),
    validators: {
      onSubmit: ({ value }) => {
        const selectedType = content.types.find((type) => type.id === value.type);
        const selectedProject = content.projects.find((project) => project.id === value.project);
        if (!selectedType || !selectedProject) return "Choose a type and project.";
        const properties = selectedType.properties
          .map((id) => content.properties.find((property) => property.id === id))
          .filter((property): property is PreviewProperty => !!property)
          .map((property) => effectiveProperty(property, value.project, value.type))
          .filter((property) => property.active);
        const missing = requiredErrors(properties, value.values, value.draft).map(
          (property) => `${property.name} is required.`
        );
        if (!value.name.trim()) missing.unshift("Enter a work item title.");
        const parent = content.items.find((row) => row.id === value.parent);
        const children = content.items.filter((row) => row.parent === item?.id);
        const allowed = content.types.some(
          (type) =>
            type.id === value.type &&
            type.active &&
            (selectedProject.types.includes(type.id) || type.mandatory) &&
            (!content.hierarchyEnabled ||
              ((!parent || content.types.find((candidate) => candidate.id === parent.type)?.level === type.level + 1) &&
                children.every(
                  (child) => content.types.find((candidate) => candidate.id === child.type)?.level === type.level - 1
                )))
        );
        if (!allowed) missing.push("Choose a type available in this project and hierarchy.");
        return missing.length ? missing.join("\n") : undefined;
      },
    },
    onSubmit: ({ value }) => {
      const selectedType = content.types.find((type) => type.id === value.type)!;
      const oldType = content.types.find((type) => type.id === item?.type);
      const exclusive = oldType
        ? content.properties.filter(
            (property) => oldType.properties.includes(property.id) && !selectedType.properties.includes(property.id)
          )
        : [];
      const saved = {
        ...value,
        draft: value.draft,
        id: item?.id ?? `WEB-${200 + content.items.length}`,
        values: Object.fromEntries(Object.entries(value.values).filter(([id]) => selectedType.properties.includes(id))),
      };
      if (oldType && oldType.id !== selectedType.id) {
        const preserved = preserveProperties(
          value,
          exclusive,
          `Type changed from ${oldType.name} to ${selectedType.name}`
        );
        setPending({
          ...saved,
          description: preserved.description,
          activity: preserved.activity,
        });
      } else commit(saved);
    },
  });
  const value = useStore(form.store, (state) => state.values);
  const formError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const selectedType = content.types.find((type) => type.id === value.type)!;
  const selectedProject = content.projects.find((project) => project.id === value.project)!;
  const parent = content.items.find((row) => row.id === value.parent);
  const children = content.items.filter((row) => row.parent === item?.id);
  const allowed = content.types.filter(
    (type) =>
      type.active &&
      (selectedProject.types.includes(type.id) || type.mandatory) &&
      (!content.hierarchyEnabled ||
        ((!parent || content.types.find((candidate) => candidate.id === parent.type)?.level === type.level + 1) &&
          children.every(
            (child) => content.types.find((candidate) => candidate.id === child.type)?.level === type.level - 1
          )))
  );
  const fields = selectedType.properties
    .map((id) => content.properties.find((property) => property.id === id))
    .filter((property): property is PreviewProperty => !!property)
    .map((property) => effectiveProperty(property, value.project, value.type))
    .filter((property) => property.active);
  if (pending)
    return (
      <Confirmation
        title="Change work item type?"
        action="Change type"
        onClose={() => setPending(null)}
        onConfirm={() => commit(pending)}
      >
        <p>
          Values exclusive to the old type will move into the description. Shared properties keep their values.
          Switching back will not automatically restore removed fields.
        </p>
        <div className="max-h-48 overflow-auto rounded border border-subtle p-3 text-12 whitespace-pre-wrap">
          {pending.description}
        </div>
      </Confirmation>
    );
  return (
    <PreviewDialog
      wide
      title={item ? "Update work item" : "Create work item"}
      onClose={onClose}
      footer={
        <>
          <label className="mr-auto flex items-center gap-2 text-12">
            <input
              type="checkbox"
              aria-label="Save as draft"
              checked={value.draft}
              disabled={!!item}
              onChange={(event) => form.setFieldValue("draft", event.target.checked)}
            />
            Save as draft
          </label>
          <Button variant="secondary" onClick={onClose}>
            Discard
          </Button>
          <Button type="submit" form="preview-work-item">
            {item ? "Update" : "Save"}
          </Button>
        </>
      }
    >
      <Form
        id="preview-work-item"
        onSubmit={async () => {
          form.setErrorMap({ onSubmit: undefined });
          await form.handleSubmit();
        }}
        className="space-y-5"
      >
        <div className="flex flex-wrap items-center gap-3">
          <select
            aria-label="Project"
            className="rounded border border-subtle bg-surface-1 px-2 py-1 text-13"
            value={value.project}
            disabled={!!item && !item.draft}
            onChange={(event) => {
              const project = content.projects.find((candidate) => candidate.id === event.target.value)!;
              const type = project.types.includes(value.type) || selectedType.mandatory ? value.type : "task";
              form.setFieldValue("project", project.id);
              form.setFieldValue("type", type);
              form.setFieldValue("values", { ...defaultPropertyValues(content, project.id, type), ...value.values });
            }}
          >
            {content.projects.map((project) => (
              <option key={project.id} value={project.id}>
                {project.name}
              </option>
            ))}
          </select>
          <CustomSearchSelect
            value={value.type}
            onChange={(id: string) => {
              form.setFieldValue("type", id);
              form.setFieldValue("values", { ...defaultPropertyValues(content, value.project, id), ...value.values });
            }}
            label={
              <span className="flex items-center gap-2">
                <TypeIcon type={selectedType} />
                {selectedType.name}
              </span>
            }
            options={allowed.map((type) => ({
              value: type.id,
              query: type.name,
              content: (
                <span className="flex items-center gap-2">
                  <TypeIcon type={type} />
                  {type.name}
                </span>
              ),
            }))}
          />
        </div>
        <Field label="Title">
          {(id) => (
            <input
              id={id}
              className={`${inputStyle} text-18 font-medium`}
              placeholder="What needs to be done?"
              value={value.name}
              onChange={(event) => form.setFieldValue("name", event.target.value)}
            />
          )}
        </Field>
        <Field label="Description">
          {(id) => (
            <textarea
              id={id}
              rows={3}
              className={inputStyle}
              placeholder="Add a description…"
              value={value.description}
              onChange={(event) => form.setFieldValue("description", event.target.value)}
            />
          )}
        </Field>
        <div className="flex flex-wrap gap-2 border-b border-subtle pb-4 text-12 text-secondary">
          <span className="rounded border border-subtle px-2 py-1">Backlog</span>
          <span className="rounded border border-subtle px-2 py-1">No priority</span>
          <span className="rounded border border-subtle px-2 py-1">Unassigned</span>
        </div>
        <div className="space-y-5">
          {fields.map((property) => (
            <div key={property.id}>
              <div className="mb-2 flex items-center gap-2">
                <PropertyIcon kind={property.kind} icon={property.icon} />
                <span className="text-13 font-medium">
                  {property.name}
                  {property.required ? " *" : ""}
                </span>
              </div>
              <PropertyInput
                property={property}
                value={value.values[property.id]}
                draft={value.draft}
                onChange={(next) =>
                  form.setFieldValue("values", {
                    ...value.values,
                    [property.id]: next,
                  })
                }
              />
              {property.description && <p className="mt-1 text-12 text-tertiary">{property.description}</p>}
            </div>
          ))}
        </div>
        {formError && (
          <ul role="alert" className="space-y-1 rounded border border-danger-strong p-3 text-13 text-danger-primary">
            {formError.split("\n").map((error) => (
              <li key={error}>{error}</li>
            ))}
          </ul>
        )}
      </Form>
    </PreviewDialog>
  );
}

function WorkItemList({ content, onSelect }: { content: PreviewContent; onSelect: (id: string) => void }) {
  const [layout, setLayout] = useState("List");
  const [filter, setFilter] = useState("");
  const [operator, setOperator] = useState("is");
  const [showDrafts, setShowDrafts] = useState(false);
  const visible = content.items.filter(
    (item) =>
      !!item.draft === showDrafts && (!filter || (operator === "is" ? item.type === filter : item.type !== filter))
  );
  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center gap-2 rounded-lg bg-layer-1 p-3">
        <select
          aria-label="Work item collection"
          className="rounded border border-subtle bg-surface-1 p-2 text-12"
          value={showDrafts ? "drafts" : "published"}
          onChange={(event) => setShowDrafts(event.target.value === "drafts")}
        >
          <option value="published">Published work items</option>
          <option value="drafts">My drafts</option>
        </select>
        <span className="text-12 text-tertiary">Type</span>
        <select
          aria-label="Type filter operator"
          className="rounded border border-subtle bg-surface-1 p-2 text-12"
          value={operator}
          onChange={(event) => setOperator(event.target.value)}
        >
          <option>is</option>
          <option>is not</option>
        </select>
        <select
          aria-label="Filter by type"
          className="rounded border border-subtle bg-surface-1 p-2 text-12"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
        >
          <option value="">All types</option>
          {content.types.map((type) => (
            <option key={type.id} value={type.id}>
              {type.name}
            </option>
          ))}
        </select>
        <div className="ml-auto flex gap-1">
          <Button
            variant={layout === "List" ? "tertiary" : "ghost"}
            aria-label="List layout"
            onClick={() => setLayout("List")}
          >
            <List className="size-4" />
          </Button>
          <Button
            variant={layout === "Table" ? "tertiary" : "ghost"}
            aria-label="Table layout"
            onClick={() => setLayout("Table")}
          >
            <Table2 className="size-4" />
          </Button>
        </div>
      </div>
      <div className="overflow-x-auto rounded-lg border border-subtle">
        <table className="w-full text-left text-13">
          <thead className="bg-layer-1 text-12 text-tertiary">
            <tr>
              <th className="font-normal p-3">Work item</th>
              <th className="font-normal p-3">Type</th>
              {layout === "Table" && (
                <>
                  <th className="font-normal p-3">Severity</th>
                  <th className="font-normal p-3">Effort</th>
                </>
              )}
            </tr>
          </thead>
          <tbody>
            {visible.map((item) => {
              const rowType = content.types.find((type) => type.id === item.type)!;
              return (
                <tr key={item.id} className="border-t border-subtle hover:bg-layer-1">
                  <td className="p-3">
                    <button className="text-left" onClick={() => onSelect(item.id)}>
                      <span className="mr-3 text-11 text-tertiary">{item.id}</span>
                      {item.name}
                    </button>
                  </td>
                  <td className="p-3">
                    <span className="flex items-center gap-2">
                      <TypeIcon type={rowType} />
                      {rowType.name}
                    </span>
                  </td>
                  {layout === "Table" && (
                    <>
                      <td className="p-3">{String(item.values.severity ?? "—")}</td>
                      <td className="p-3">{String(item.values.effort ?? "—")}</td>
                    </>
                  )}
                </tr>
              );
            })}
          </tbody>
        </table>
        {!visible.length && (
          <p className="p-8 text-center text-13 text-tertiary">No sample work items match this filter.</p>
        )}
      </div>
    </div>
  );
}

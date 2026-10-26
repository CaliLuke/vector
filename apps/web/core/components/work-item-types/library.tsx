import { WorkspaceTypeDeletion } from "./workspace-type-deletion";
import { capturePropertyEditors, type TypePersistence } from "./library-persistence";
import { useId, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { ArrowDown, ArrowUp, ChevronDown, ChevronRight, Plus } from "lucide-react";
import { Button } from "@plane/propel/button";
import { CustomMenu, Form, ToggleSwitch } from "@plane/ui";
import { Confirmation, EmptyState, PreviewDialog, PropertyIcon, TypeIcon } from "./controls";
import type { PreviewProperty, PreviewType } from "./preview-content";
import { useTypePreview } from "./preview-provider";
import { AttachProperties } from "./attach-properties";
import { TypeEditor } from "./type-editor";
import { PropertyEditor } from "./property-editor";
import { ContextEditor } from "./context-editor";
import { CascadingEditor } from "./cascading-editor";
import { preserveProperties } from "./preview-actions";
import { PropertyRemovalDialog } from "./property-removal-dialog";

export function TypeLibrary({ readOnly = false, persistence }: { readOnly?: boolean; persistence?: TypePersistence }) {
  const { content: previewContent, setContent, announce } = useTypePreview();
  const content = persistence
    ? {
        ...previewContent,
        properties: persistence.properties,
        projects: persistence.projects.map((project) => ({ ...project, types: [] })),
        items: [],
        types: persistence.types.map((type) => ({
          ...type,
          properties: persistence.attachments.find((link) => link.type_id === type.id)?.property_ids ?? [],
        })),
      }
    : previewContent;
  const isDefault = (type: PreviewType) =>
    persistence ? persistence.types.some((item) => item.id === type.id && item.default) : type.id === "task";
  const [actionRevision, setActionRevision] = useState<number>();
  const [tab, setTab] = useState("types");
  const [expanded, setExpanded] = useState<string[]>([]);
  const [editType, setEditType] = useState<PreviewType | "new" | null>(null);
  const [editProperty, setEditProperty] = useState<PreviewProperty | "new" | null>(null);
  const [attach, setAttach] = useState<PreviewType | null>(null);
  const [context, setContext] = useState<PreviewProperty | null>(null);
  const [cascade, setCascade] = useState<PreviewProperty | null>(null);
  const [remove, setRemove] = useState<{
    property: PreviewProperty;
    type?: PreviewType;
  } | null>(null);
  const [mandatory, setMandatory] = useState<PreviewType | null>(null);
  const [deleteType, setDeleteType] = useState<PreviewType | null>(null);
  const updateType = (type: PreviewType) => {
    if (persistence) {
      void persistence.save(type).catch(() => {});
      return;
    }
    setContent((current) => ({
      ...current,
      types: current.types.map((item) => (item.id === type.id ? type : item)),
    }));
  };
  const reorder = (type: PreviewType, index: number, offset: number) => {
    const properties = [...type.properties];
    [properties[index], properties[index + offset]] = [properties[index + offset], properties[index]];
    if (persistence) void persistence.attachProperties(type.id, properties).catch(() => {});
    else
      setContent((current) => ({
        ...current,
        types: current.types.map((item) => (item.id === type.id ? { ...item, properties } : item)),
      }));
  };
  const affectedItems = remove
    ? content.items.filter(
        (item) => (!remove.type || item.type === remove.type.id) && item.values[remove.property.id] !== undefined
      )
    : [];
  const propertyToEdit = editProperty === "new" || editProperty === null ? undefined : editProperty;
  const renderTypeDialogs = () => (
    <>
      {editType && (
        <TypeEditor
          type={editType === "new" ? undefined : editType}
          types={content.types}
          onSave={persistence ? (type, creating) => persistence.save(type, creating, actionRevision) : undefined}
          onClose={() => setEditType(null)}
        />
      )}
      {mandatory && (
        <Confirmation
          title={`Make ${mandatory.name} mandatory?`}
          action="Make mandatory"
          onClose={() => setMandatory(null)}
          onConfirm={async () => {
            const target = mandatory;
            if (persistence) {
              await persistence.save({ ...target, active: true, mandatory: true }, false, actionRevision);
              setMandatory(null);
              announce(`${target.name} is now available in every project.`);
              return;
            }
            setContent((current) => ({
              ...current,
              types: current.types.map((type) =>
                type.id === target.id
                  ? {
                      ...type,
                      mandatory: true,
                    }
                  : type
              ),
              projects: current.projects.map((project) => ({
                ...project,
                types: [...new Set([...project.types, target.id])],
              })),
            }));
            setMandatory(null);
            announce(`${target.name} is now available in every project.`);
          }}
        >
          <p>
            This adds the type to all {persistence?.projectCount ?? content.projects.length} projects. Project
            administrators cannot remove or pause it. Switching back to Import later keeps these assignments.
          </p>
        </Confirmation>
      )}
      {deleteType &&
        (persistence ? (
          <WorkspaceTypeDeletion
            key={`${persistence.propertyRemoval.scope}:${persistence.propertyRemoval.accountId}:${deleteType.id}`}
            type={deleteType}
            persistence={persistence}
            revision={actionRevision ?? persistence.revision}
            onClose={() => setDeleteType(null)}
          />
        ) : (
          <DeleteType type={deleteType} onClose={() => setDeleteType(null)} />
        ))}
    </>
  );
  const editors = capturePropertyEditors(persistence, actionRevision);
  const renderConfigurationDialogs = () => (
    <>
      {editProperty && (
        <PropertyEditor
          property={propertyToEdit}
          library={content}
          formula={editors.formula}
          members={persistence?.members}
          onSave={
            persistence
              ? (property, creating) => persistence.saveProperty(property, creating, actionRevision)
              : undefined
          }
          onClose={() => setEditProperty(null)}
        />
      )}
      {attach && (
        <AttachProperties
          type={attach}
          properties={content.properties}
          onClose={() => setAttach(null)}
          onAttach={
            persistence
              ? (properties) => persistence.attachProperties(attach.id, properties, actionRevision)
              : undefined
          }
        />
      )}
      {context && (
        <ContextEditor
          property={context}
          library={content}
          members={persistence?.members}
          review={editors.review}
          onSave={persistence ? (property) => persistence.saveProperty(property, false, actionRevision) : undefined}
          onClose={() => setContext(null)}
        />
      )}
      {cascade && (
        <CascadingEditor
          property={cascade}
          review={editors.review}
          onSave={persistence ? (property) => persistence.saveProperty(property, false, actionRevision) : undefined}
          onClose={() => setCascade(null)}
        />
      )}
    </>
  );
  const renderPropertyDialogs = () => (
    <>
      {remove && persistence && (
        <PropertyRemovalDialog
          id={remove.property.id}
          name={remove.property.name}
          typeId={remove.type?.id}
          revision={actionRevision ?? persistence.revision}
          persistence={persistence.propertyRemoval}
          onClose={() => setRemove(null)}
        />
      )}
      {remove && !persistence && (
        <Confirmation
          title={`${remove.type ? "Detach" : "Delete"} ${remove.property.name}?`}
          action={remove.type ? "Detach property" : "Delete property"}
          token="Confirm"
          tokenRequiredWhen={(values) => !remove.type && values.permanent}
          initialValues={{ permanent: false }}
          renderFields={(form) =>
            remove.type ? null : (
              <form.Field name="permanent">
                {(field) => (
                  <fieldset className="space-y-3">
                    <legend className="sr-only">How should property values be removed?</legend>
                    <label className="flex gap-2">
                      <input
                        type="radio"
                        name="permanent"
                        aria-label="Save values to work item descriptions"
                        checked={!field.state.value}
                        onChange={() => field.handleChange(false)}
                      />
                      Save values to work item descriptions (recommended)
                    </label>
                    <label className="flex gap-2">
                      <input
                        type="radio"
                        name="permanent"
                        aria-label="Remove values permanently"
                        checked={field.state.value}
                        onChange={() => field.handleChange(true)}
                      />
                      Remove values permanently
                    </label>
                  </fieldset>
                )}
              </form.Field>
            )
          }
          onClose={() => setRemove(null)}
          onConfirm={async ({ permanent }) => {
            const target = remove;
            setContent((current) => ({
              ...current,
              properties: target.type
                ? current.properties
                : current.properties.filter((property) => property.id !== target.property.id),
              types: current.types.map((type) =>
                !target.type || type.id === target.type.id
                  ? {
                      ...type,
                      properties: type.properties.filter((id) => id !== target.property.id),
                    }
                  : type
              ),
              items: current.items.map((item) => {
                if (target.type && item.type !== target.type.id) return item;
                const next = preserveProperties(item, [target.property], `${target.property.name} removed`);
                return permanent
                  ? {
                      ...next,
                      description: item.description,
                    }
                  : next;
              }),
            }));
            setRemove(null);
            announce("Property removed.");
          }}
        >
          <p>
            {affectedItems.length} work items,{" "}
            {remove.type ? 1 : content.types.filter((type) => type.properties.includes(remove.property.id)).length}{" "}
            types, and {new Set(affectedItems.map((item) => item.project)).size} projects are affected.
          </p>
          {remove.type && <p>Values are saved to work item descriptions when a property is detached.</p>}
          {!remove.type && (
            <p>Choose whether to preserve values in work item descriptions or remove them permanently.</p>
          )}
        </Confirmation>
      )}
    </>
  );
  return (
    <>
      <div role="tablist" aria-label="Work item type settings" className="mb-8 flex gap-6 border-b border-subtle">
        {[
          ["types", "Work item Types"],
          ["properties", "Properties"],
        ].map(([id, label]) => (
          <button
            key={id}
            role="tab"
            aria-selected={tab === id}
            onClick={() => setTab(id)}
            className={`border-b-2 px-1 pb-3 text-13 font-medium ${tab === id ? "border-accent-strong text-primary" : "border-transparent text-tertiary hover:text-primary"}`}
          >
            {label}
          </button>
        ))}
      </div>
      <div className="mb-6 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h2 className="text-16 font-medium">
            {tab === "types" ? "Work item Types" : "Custom properties"}{" "}
            <span className="font-normal ml-2 text-13 text-tertiary">
              {tab === "types" ? content.types.length : content.properties.length}
            </span>
          </h2>
          <p className="mt-1 text-13 text-tertiary">
            {tab === "types"
              ? "Define the kinds of work your teams track across projects."
              : "Create reusable properties and attach them to work item types."}
          </p>
        </div>
        <Button
          size="lg"
          disabled={readOnly}
          prependIcon={<Plus />}
          onClick={() => {
            setActionRevision(persistence?.revision);
            if (tab === "types") setEditType("new");
            else setEditProperty("new");
          }}
        >
          {tab === "types" ? "Add work item type" : "Add new property"}
        </Button>
      </div>
      {tab === "types" ? (
        <div className="space-y-3">
          {content.types.map((type) => (
            <div key={type.id} className="overflow-hidden rounded-lg border border-subtle">
              <div className="flex items-center gap-3 px-4 py-4">
                <button
                  aria-label={`${expanded.includes(type.id) ? "Collapse" : "Expand"} ${type.name}`}
                  aria-expanded={expanded.includes(type.id)}
                  onClick={() =>
                    setExpanded((current) =>
                      current.includes(type.id) ? current.filter((id) => id !== type.id) : [...current, type.id]
                    )
                  }
                  className="rounded p-1 text-tertiary hover:bg-layer-2"
                >
                  {expanded.includes(type.id) ? (
                    <ChevronDown className="size-4" />
                  ) : (
                    <ChevronRight className="size-4" />
                  )}
                </button>
                <TypeIcon type={type} />
                <div className="min-w-0 flex-1">
                  <h3 className="text-14 font-medium text-primary">{type.name}</h3>
                  <p className="mt-0.5 text-12 text-tertiary">{type.description}</p>
                </div>
                {isDefault(type) ? (
                  <span className="rounded bg-layer-2 px-2 py-1 text-11 text-secondary">Default</span>
                ) : (
                  <>
                    <span className="hidden rounded border border-subtle px-2 py-0.5 text-11 text-tertiary sm:inline">
                      {type.mandatory ? "Mandatory" : "Import"}
                    </span>
                    <ToggleSwitch
                      label={`${type.name} active`}
                      value={type.active}
                      disabled={readOnly || type.mandatory}
                      onChange={(active) =>
                        updateType({
                          ...type,
                          active,
                        })
                      }
                    />
                  </>
                )}
                <CustomMenu ellipsis placement="bottom-end" ariaLabel={`${type.name} actions`} disabled={readOnly}>
                  <CustomMenu.MenuItem
                    onClick={() => {
                      setActionRevision(persistence?.revision);
                      setEditType(type);
                    }}
                  >
                    Edit type
                  </CustomMenu.MenuItem>
                  {!isDefault(type) && (
                    <>
                      <CustomMenu.MenuItem
                        onClick={() =>
                          type.mandatory
                            ? updateType({
                                ...type,
                                mandatory: false,
                              })
                            : (setActionRevision(persistence?.revision), setMandatory(type))
                        }
                      >
                        {type.mandatory ? "Change rollout to Import" : "Make mandatory for all projects"}
                      </CustomMenu.MenuItem>
                      <CustomMenu.MenuItem
                        onClick={() => {
                          setActionRevision(persistence?.revision);
                          setDeleteType(type);
                        }}
                      >
                        Delete type
                      </CustomMenu.MenuItem>
                    </>
                  )}
                </CustomMenu>
              </div>
              {expanded.includes(type.id) && (
                <div className="border-t border-subtle bg-layer-1/40 px-5 py-3">
                  {type.properties.length === 0 && (
                    <p className="py-3 text-13 text-tertiary">No custom properties attached yet.</p>
                  )}
                  {type.properties.map((id, index) => {
                    const property = content.properties.find((item) => item.id === id);
                    return (
                      property && (
                        <div key={id} className="flex items-center gap-3 border-b border-subtle py-3 last:border-0">
                          <PropertyIcon kind={property.kind} />
                          <span className="min-w-0 flex-1 truncate text-13">{property.name}</span>
                          {property.required && <span className="text-11 text-tertiary">Required</span>}
                          <span className="rounded bg-layer-2 px-2 py-0.5 text-11 text-tertiary">{property.kind}</span>
                          <button
                            aria-label={`Move ${property.name} up`}
                            disabled={readOnly || index === 0}
                            className="text-tertiary disabled:opacity-25"
                            onClick={() => reorder(type, index, -1)}
                          >
                            <ArrowUp className="size-3.5" />
                          </button>
                          <button
                            aria-label={`Move ${property.name} down`}
                            disabled={readOnly || index === type.properties.length - 1}
                            className="text-tertiary disabled:opacity-25"
                            onClick={() => reorder(type, index, 1)}
                          >
                            <ArrowDown className="size-3.5" />
                          </button>
                          <CustomMenu ellipsis ariaLabel={`${property.name} attachment actions`} disabled={readOnly}>
                            <CustomMenu.MenuItem
                              onClick={() => {
                                setActionRevision(persistence?.revision);
                                setEditProperty(property);
                              }}
                            >
                              Edit property
                            </CustomMenu.MenuItem>
                            <CustomMenu.MenuItem
                              onClick={() => {
                                setActionRevision(persistence?.revision);
                                setRemove({
                                  property,
                                  type,
                                });
                              }}
                            >
                              Detach property
                            </CustomMenu.MenuItem>
                          </CustomMenu>
                        </div>
                      )
                    );
                  })}
                  <Button
                    className="mt-2"
                    variant="ghost"
                    disabled={readOnly}
                    prependIcon={<Plus />}
                    onClick={() => {
                      setActionRevision(persistence?.revision);
                      setAttach(type);
                    }}
                  >
                    Add properties
                  </Button>
                </div>
              )}
            </div>
          ))}
        </div>
      ) : (
        <div className="overflow-hidden rounded-lg border border-subtle">
          {content.properties.map((property) => (
            <div key={property.id} className="flex items-center gap-3 border-b border-subtle px-4 py-4 last:border-0">
              <PropertyIcon kind={property.kind} />
              <button
                className="min-w-0 flex-1 text-left"
                disabled={readOnly}
                onClick={() => {
                  setActionRevision(persistence?.revision);
                  setEditProperty(property);
                }}
              >
                <span className="block text-13 font-medium">{property.name}</span>
                <span className="mt-0.5 block text-12 text-tertiary">
                  {content.types
                    .filter((type) => type.properties.includes(property.id))
                    .map((type) => type.name)
                    .join(", ") || "Not attached to any type"}
                  {property.contexts.length > 0 && ` · ${property.contexts.length + 1} contexts`}
                </span>
              </button>
              <span className="rounded bg-layer-2 px-2 py-0.5 text-11 text-secondary">{property.kind}</span>
              <ToggleSwitch
                label={`${property.name} active`}
                value={property.active}
                disabled={readOnly}
                onChange={(active) => {
                  if (persistence) {
                    void persistence.saveProperty({ ...property, active }).catch(() => {});
                    return;
                  }
                  setContent((current) => ({
                    ...current,
                    properties: current.properties.map((item) =>
                      item.id === property.id
                        ? {
                            ...item,
                            active,
                          }
                        : item
                    ),
                  }));
                }}
              />
              <CustomMenu ellipsis ariaLabel={`${property.name} actions`} disabled={readOnly}>
                <CustomMenu.MenuItem
                  onClick={() => {
                    setActionRevision(persistence?.revision);
                    setEditProperty(property);
                  }}
                >
                  Edit property
                </CustomMenu.MenuItem>
                <CustomMenu.MenuItem
                  onClick={() => {
                    setActionRevision(persistence?.revision);
                    setContext(property);
                  }}
                >
                  Manage contexts
                </CustomMenu.MenuItem>
                {property.kind === "Cascading" && (
                  <CustomMenu.MenuItem
                    onClick={() => {
                      setActionRevision(persistence?.revision);
                      setCascade(property);
                    }}
                  >
                    Manage levels and options
                  </CustomMenu.MenuItem>
                )}
                <CustomMenu.MenuItem
                  onClick={() => {
                    setActionRevision(persistence?.revision);
                    setRemove({
                      property,
                    });
                  }}
                >
                  Delete property
                </CustomMenu.MenuItem>
              </CustomMenu>
            </div>
          ))}
          {!content.properties.length && (
            <EmptyState title="No properties yet" description="Create your first property, then attach it to a type." />
          )}
        </div>
      )}
      {renderTypeDialogs()}
      {renderConfigurationDialogs()}
      {renderPropertyDialogs()}
    </>
  );
}
function DeleteType({ type, onClose }: { type: PreviewType; onClose: () => void }) {
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

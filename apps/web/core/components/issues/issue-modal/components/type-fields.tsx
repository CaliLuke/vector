import { useId } from "react";
import { CustomSearchSelect } from "@plane/ui";
import { ItemPropertyInput } from "@/components/work-item-types/item-property-input";
import { TypeIcon } from "@/components/work-item-types/controls";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import { WorkItemRichTextField } from "./item-rich-text-field";
import type { IssueDraft, IssueDraftForm } from "../form-state";
import type { ItemTypeFields } from "../use-item-type-fields";

export function IssueTypeSelect({
  fields,
  value,
  onChange,
  beforeChange,
}: {
  fields: ItemTypeFields;
  value: string | null;
  onChange: (id: string) => void;
  beforeChange: () => boolean;
}) {
  if (!fields.choicesQuery.data?.enabled) return null;
  const { selected, unavailable } = fields;
  return (
    <div className="min-w-0">
      <CustomSearchSelect
        value={value}
        onChange={(id: string) => {
          if (beforeChange()) onChange(id);
        }}
        customButtonAsChild
        customButton={
          <button
            type="button"
            aria-label="Work item type"
            className="flex items-center gap-2 rounded-md border border-subtle px-2 py-1 text-13"
          >
            {selected && <TypeIcon type={selected} />}
            {selected?.name ?? (value ? "Unavailable type" : "Select type")}
          </button>
        }
        options={fields.choicesQuery.data.types
          .filter((type) => type.available && fields.schema?.eligible_type_ids.includes(type.id))
          .map((type) => ({
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
      {!value && fields.schema?.selection_reason && (
        <p className="mt-1 text-12 text-secondary">{fields.schema.selection_reason}</p>
      )}
      {fields.hierarchyMismatch && (
        <p className="mt-1 text-12 text-danger-primary">Choose a type from the level immediately below this parent.</p>
      )}
      {unavailable && (
        <p className="mt-1 text-12 text-secondary">This type is not available for new work items in this project.</p>
      )}
    </div>
  );
}

export function IssueCustomFields({
  form,
  fields,
  values,
  workspaceSlug,
  isDraft,
  projectId,
  disabled,
  onAssetUpload,
}: {
  form: IssueDraftForm;
  fields: ItemTypeFields;
  values: IssueDraft;
  workspaceSlug: string;
  isDraft: boolean;
  projectId?: string | null;
  disabled: boolean;
  onAssetUpload: (id: string) => void;
}) {
  const instanceId = useId();
  const { data: workspace } = useWorkspaceBySlug(workspaceSlug);
  const { schema } = fields;
  const schemaError = fields.schemaQuery.isError || fields.sourceSchemaError;
  if (!schemaError && (!schema?.enabled || !schema.properties.length)) return null;
  return (
    <div className="space-y-4 border-t border-subtle bg-surface-1 px-5 py-4">
      {schemaError && (
        <p role="alert" className="text-13 text-danger-primary">
          Could not load {fields.sourceSchemaError ? "the previous type's" : "work item"} fields.{" "}
          <button type="button" className="underline" onClick={() => void fields.retrySchemas()}>
            Retry
          </button>
        </p>
      )}
      {schema?.properties.map((property) => (
        <form.Field key={property.id} name={`custom_properties.${property.id}`}>
          {(field) => (
            <ItemPropertyInput
              property={property}
              value={field.state.value}
              schema={schema}
              disabled={disabled}
              error={fields.errors[property.id]}
              onChange={(value) => fields.setValue(property.id, value)}
              onControlMount={(element) => {
                if (element) fields.controls.current.set(property.id, element);
                else fields.controls.current.delete(property.id);
              }}
              formulaResult={
                fields.previewError
                  ? {
                      kind: property.configuration.result,
                      status: "error",
                      error: "Could not calculate this value. Try again.",
                      value: null,
                    }
                  : fields.computed?.[property.id]
              }
              renderRichText={(_property, attributes) => (
                <div
                  ref={(element) => {
                    if (element) fields.controls.current.set(property.id, element);
                    else fields.controls.current.delete(property.id);
                  }}
                  className="rounded-md border border-subtle"
                >
                  <WorkItemRichTextField
                    id={`${instanceId}-${property.id}`}
                    propertyId={property.id}
                    value={richTextValue(field.state.value)}
                    attributes={attributes}
                    editors={fields.editors}
                    onEditorStateChange={fields.onEditorStateChange}
                    workspaceSlug={workspaceSlug}
                    workspaceId={workspace?.id ?? ""}
                    projectId={projectId ?? undefined}
                    itemId={values.id}
                    isDraft={isDraft}
                    onChange={(html) => fields.setValue(property.id, html)}
                    onAssetUpload={onAssetUpload}
                  />
                </div>
              )}
            />
          )}
        </form.Field>
      ))}
    </div>
  );
}

function richTextValue(value: unknown) {
  return typeof value === "string" ? value : "";
}

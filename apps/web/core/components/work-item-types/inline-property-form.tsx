import { useEffect, useLayoutEffect, useEffectEvent, useId, useState } from "react";
import type { ReactNode } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import type { EditorRefApi } from "@plane/editor";
import { Button } from "@plane/propel/button";
import { RichTextEditor } from "@/components/editor/rich-text";
import { WorkItemRichTextField } from "@/components/issues/issue-modal/components/item-rich-text-field";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import type { PublishedItemEditScope } from "@/queries/published-item-edit";
import { ItemPropertyInput } from "./item-property-input";
import { ItemPropertyValue } from "./item-property-value";
import { propertyFieldErrors, submittedPropertyValues } from "./item-property-draft";
import type { PropertyValues } from "./item-property-draft";
import { reloadPropertyDraft } from "./inline-property-values";
import { isPropertyHistoryAccessDenied } from "@/queries/work-item-property-history";
import { useInlinePropertyPreview } from "./use-inline-property-preview";
import type { usePublishedItemEdit } from "./use-published-item-edit";

type Edit = ReturnType<typeof usePublishedItemEdit>;
type Snapshot = NonNullable<Edit["source"]["data"]>;
type Props = {
  onCloseReadyChange?: (ready: boolean) => void;
  scope: PublishedItemEditScope;
  edit: Edit;
  snapshot: Snapshot;
  onClose: () => void;
  onAccessLost: () => void;
};
export function InlinePropertyForm({ scope, edit, snapshot, onClose, onAccessLost, onCloseReadyChange }: Props) {
  const [baseline, setBaseline] = useState(snapshot);
  const [defaults, setDefaults] = useState(() => ({
    values: (snapshot.source.item.custom_properties ?? {}) as PropertyValues,
  }));
  const [reloaded, setReloaded] = useState<string[]>([]);
  const [reloading, setReloading] = useState(false);
  const [reloadError, setReloadError] = useState<string>();
  const [editors] = useState(() => new Map<string, EditorRefApi>());
  const [editorsReady, setEditorsReady] = useState(true);
  const [controls] = useState(() => new Map<string, HTMLElement>());
  const form = useForm({
    defaultValues: defaults,
    validators: {
      onSubmit: () => (readyToSave(editors) ? undefined : "An editor is still processing changes. Wait before saving."),
    },
    onSubmit: async ({ value, formApi }) => {
      if (edit.conflict || preview.changed || reloadError) return;
      try {
        await edit.save({
          mode: "edit",
          item_id: scope.issueId,
          project_id: scope.projectId,
          revision: baseline.schema.revision,
          expected_source_version: baseline.source.source_version,
          changes: {
            custom_properties: submittedPropertyValues(
              value.values,
              baseline.schema,
              baseline.source.item.custom_properties ?? {}
            ),
          },
        });
      } catch (cause) {
        const serverErrors = Object.entries(propertyFieldErrors(cause)).map(([id, error]) => [`values.${id}`, error]);
        if (serverErrors.length > 0) {
          formApi.setErrorMap({ onSubmit: { fields: Object.fromEntries(serverErrors) } });
        }
        throw cause;
      }
    },
  });
  const values = useStore(form.store, (state) => state.values.values);
  const submitting = useStore(form.store, (state) => state.isSubmitting);
  const submissionError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const preview = useInlinePropertyPreview(scope, baseline.schema, baseline.source.item, values);
  const accessLost = useEffectEvent(onAccessLost);
  useEffect(() => {
    if (isPropertyHistoryAccessDenied(preview.cause)) accessLost();
  }, [preview.cause]);
  const fields = propertyFieldErrors(edit.cause);
  useEffect(() => {
    const first = Object.keys(fields)[0];
    if (first && !submitting) (editors.get(first) ?? controls.get(first))?.focus(undefined);
  }, [fields, editors, controls, submitting]);
  const busy = edit.busy || submitting || reloading;
  const canClose = !busy && editorsReady;
  useLayoutEffect(() => {
    onCloseReadyChange?.(canClose);
  }, [canClose, onCloseReadyChange]);
  const displayError = reloadError ?? (typeof submissionError === "string" ? submissionError : undefined) ?? edit.error;
  const reload = async () => {
    if (!readyToSave(editors)) return;
    const previous = baseline.source.item.custom_properties ?? {};
    setReloading(true);
    try {
      const fresh = await edit.reload();
      if (!fresh || !readyToSave(editors)) return;
      const draft = reloadPropertyDraft(
        form.store.state.values.values,
        previous,
        baseline.schema,
        fresh.source.item.custom_properties ?? {},
        fresh.schema
      );
      if (draft.unavailable.length) {
        setReloadError(
          `Edited fields are no longer available: ${draft.unavailable.map((field) => field.title).join(", ")}. Your changes remain here. Copy any changes you need before cancelling.`
        );
        return;
      }
      setBaseline(fresh);
      const nextDefaults = { values: draft.values };
      setDefaults(nextDefaults);
      form.reset(nextDefaults);
      setReloaded(draft.changed);
      setReloadError(undefined);
    } finally {
      setReloading(false);
    }
  };
  if (isPropertyHistoryAccessDenied(preview.cause)) return null;
  if (!baseline.schema.enabled)
    return (
      <div>
        <p>Custom properties are no longer enabled.</p>
        <Button variant="secondary" onClick={onClose}>
          Close
        </Button>
      </div>
    );
  return (
    <form
      aria-label="Edit custom properties"
      onSubmit={(event) => {
        event.preventDefault();
        event.stopPropagation();
        void form.handleSubmit().catch(() => {});
      }}
      className="space-y-4"
    >
      {reloaded.length > 0 && (
        <output className="text-12 text-secondary">
          Current values reloaded. Your edited fields were kept; review them before saving.
        </output>
      )}
      <fieldset disabled={busy} inert={busy} className="min-w-0 space-y-4">
        <PropertyControls
          scope={scope}
          baseline={baseline}
          computed={preview.computed}
          previewError={preview.error}
          editors={editors}
          controls={controls}
          renderField={(id, render) => (
            <form.Field name={`values.${id}`}>
              {(field) =>
                render({
                  value: field.state.value,
                  errors: field.state.meta.errors,
                  onChange: field.handleChange,
                  clearSubmitError: () => field.setErrorMap({ onSubmit: undefined }),
                })
              }
            </form.Field>
          )}
          onEditorStateChange={() => {
            const ready = readyToSave(editors);
            setEditorsReady(ready);
            if (ready) form.setErrorMap({ onSubmit: undefined });
          }}
          reloaded={reloaded}
        />
      </fieldset>
      {displayError && (
        <p role="alert" className="text-12 text-danger-primary">
          {displayError}
        </p>
      )}
      <div className="flex flex-wrap justify-end gap-2">
        <Button type="button" variant="secondary" disabled={!canClose} onClick={onClose}>
          Cancel
        </Button>
        {edit.conflict || preview.changed || reloadError ? (
          <Button
            type="button"
            variant="primary"
            disabled={!canClose || edit.source.isFetching}
            onClick={() => void reload()}
          >
            Reload current values
          </Button>
        ) : (
          <Button type="submit" variant="primary" disabled={!canClose} loading={busy}>
            Save properties
          </Button>
        )}
      </div>
    </form>
  );
}
function readyToSave(editors: Map<string, EditorRefApi>) {
  return [...editors.values()].every((editor) => editor.isEditorReadyToDiscard());
}

type ControlsProps = {
  scope: PublishedItemEditScope;
  baseline: Snapshot;
  computed: Snapshot["schema"]["computed"] | undefined;
  previewError: boolean;
  editors: Map<string, EditorRefApi>;
  controls: Map<string, HTMLElement>;
  renderField: (id: string, render: (field: NativePropertyField) => ReactNode) => ReactNode;
  onEditorStateChange: () => void;
  reloaded: string[];
};
type NativePropertyField = {
  value: unknown;
  errors: unknown[];
  onChange: (value: unknown) => void;
  clearSubmitError: () => void;
};
function PropertyControls(props: ControlsProps) {
  const { scope, baseline, computed, previewError, editors, controls, renderField, onEditorStateChange, reloaded } =
    props;
  const instance = useId();
  const workspace = useWorkspaceBySlug(scope.workspaceSlug);
  return baseline.schema.properties.map((property) => (
    <div key={property.id}>
      {renderField(property.id, (field) => {
        const fieldError = field.errors.find((error): error is string => typeof error === "string");
        return (
          <>
            <ItemPropertyInput
              property={property}
              value={field.value}
              schema={baseline.schema}
              error={fieldError}
              onChange={(value) => {
                field.clearSubmitError();
                field.onChange(value);
              }}
              onControlMount={(element) => {
                if (element) controls.set(property.id, element);
                else controls.delete(property.id);
              }}
              formulaResult={
                previewError
                  ? { status: "error", value: null, error: "Could not calculate this value. Try again." }
                  : computed?.[property.id]
              }
              renderRichText={(_field, attributes) =>
                workspace.data ? (
                  <div className="rounded-md border border-subtle">
                    <WorkItemRichTextField
                      id={`${instance}-${property.id}`}
                      propertyId={property.id}
                      value={typeof field.value === "string" ? field.value : ""}
                      workspaceSlug={scope.workspaceSlug}
                      workspaceId={workspace.data.id}
                      projectId={scope.projectId}
                      itemId={scope.issueId}
                      isDraft={false}
                      attributes={attributes}
                      editors={editors}
                      onEditorStateChange={onEditorStateChange}
                      onChange={(value) => {
                        field.clearSubmitError();
                        field.onChange(value);
                      }}
                      onAssetUpload={() => undefined}
                    />
                  </div>
                ) : (
                  <output>
                    {workspace.isError ? "Could not load the rich-text editor." : "Loading rich-text editor…"}
                    <button type="button" onClick={() => void workspace.refetch()}>
                      Retry
                    </button>
                  </output>
                )
              }
            />
            {reloaded.includes(property.id) && (
              <div className="mt-2 rounded bg-layer-1 p-2 text-12 text-secondary">
                <p>Current saved value</p>
                <ItemPropertyValue
                  property={property}
                  schema={{ ...baseline.schema, stored_values: baseline.source.item.custom_properties ?? {} }}
                  renderRichText={(html) =>
                    workspace.data ? (
                      <RichTextEditor
                        key={`${baseline.source.source_version}-${property.id}`}
                        id={`${instance}-saved-${property.id}`}
                        initialValue={html}
                        workspaceSlug={scope.workspaceSlug}
                        workspaceId={workspace.data.id}
                        projectId={scope.projectId}
                        editable={false}
                        containerClassName="border-none"
                      />
                    ) : (
                      <span>Loading current value…</span>
                    )
                  }
                />
              </div>
            )}
          </>
        );
      })}
    </div>
  ));
}

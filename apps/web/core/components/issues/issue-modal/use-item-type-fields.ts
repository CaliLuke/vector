import { nextFormulaClockBoundary } from "@/queries/work-item-clock";
import { isAxiosError } from "axios";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery, skipToken } from "@tanstack/react-query";
import { useStore } from "@tanstack/react-form";
import type { EditorRefApi } from "@plane/editor";
import type { TIssue } from "@plane/types";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { useCurrentUserId } from "@/hooks/use-current-user";
import useDebounce from "@/hooks/use-debounce";
import { itemSchemaOptions, itemTypeChoicesOptions } from "@/queries/work-item-schema";
import { workItemSchemaService } from "@/services/work-item-schema.service";
import type { WorkItemType } from "@/services/work-item-types.service";
import type { ItemSchema, ItemSchemaInput } from "@/services/work-item-schema.service";
import {
  applyPropertyDefaults,
  propertyFieldErrors,
  submittedPropertyValues,
  submittedFormerPropertyValues,
} from "@/components/work-item-types/item-property-draft";
import type { IssueDraft, IssueDraftForm } from "./form-state";

export function useItemTypeFields(
  form: IssueDraftForm,
  workspaceSlug: string,
  initialData: Partial<TIssue> | undefined,
  mode: ItemSchemaInput["mode"],
  data: Partial<TIssue> | undefined = initialData
) {
  const userId = useCurrentUserId();
  const values = useStore(form.store, (state) => state.values);
  const controls = useRef(new Map<string, HTMLElement>());
  const pendingErrorFocus = useRef<string | undefined>(undefined);
  const [editors] = useState(() => new Map<string, EditorRefApi>());
  const saved = useSavedItemFields(form, editors, initialData, data);
  const fieldMeta = useStore(form.store, (state) => state.fieldMeta);
  const errors = useMemo(
    () =>
      Object.fromEntries(
        Object.entries(fieldMeta).flatMap(([name, meta]) => {
          const error = meta?.errorMap.onServer;
          return name.startsWith("custom_properties.") && typeof error === "string"
            ? [[name.slice("custom_properties.".length), error]]
            : [];
        })
      ),
    [fieldMeta]
  );
  const scope = itemScope(mode, values, data);
  const schemaQuery = useQuery(itemSchemaOptions({ userId, workspaceSlug, input: scope }));
  const { changingConfiguration, sourceSchemaQuery } = useSourceItemSchema(userId, workspaceSlug, values, data, mode);
  const { projectRole } = useProjectPermissions(workspaceSlug, values.project_id ?? undefined);
  const choicesQuery = useQuery(
    itemTypeChoicesOptions(userId, workspaceSlug, projectRole ? values.project_id || undefined : undefined)
  );
  const schema = schemaQuery.isError && !isTransientSchemaError(schemaQuery.error) ? undefined : schemaQuery.data;
  const customValues = useMemo(() => values.custom_properties ?? {}, [values.custom_properties]);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  useEffect(() => {
    if (isSubmitting) return;
    if (!pendingErrorFocus.current || !errors[pendingErrorFocus.current]) return;
    const control = editors.get(pendingErrorFocus.current) ?? controls.current.get(pendingErrorFocus.current);
    if (control) {
      control.focus(undefined);
      pendingErrorFocus.current = undefined;
    }
  }, [errors, isSubmitting, editors]);
  const freshSchema =
    schemaQuery.isFetchedAfterMount && !schemaQuery.isFetching && !schemaQuery.isError ? schema : undefined;
  useSchemaDefaults(form, freshSchema, customValues, !data?.id, values.type_id);
  const preview = useFormulaPreview(userId, workspaceSlug, scope, schema, values, data);
  const { selected, unavailable, hierarchyMismatch, keepsSavedType } = typeSelectionState(
    values,
    data,
    schema,
    choicesQuery.data
  );
  const validate = () => {
    if (!schema || schemaQuery.isError)
      return schemaQuery.isError
        ? "Could not load this work item's fields. Check the project and type, then retry."
        : "Work item fields are still loading.";
    if (changingConfiguration && (!sourceSchemaQuery.data || sourceSchemaQuery.isError))
      return "Could not load the previous type's fields. Retry before saving this change.";
    const selectionError = validateTypeSelection(
      schema,
      values.type_id,
      mode,
      unavailable,
      keepsSavedType,
      hierarchyMismatch
    );
    if (selectionError) return selectionError;
    if ([...editors.values()].some((editor) => !editor.isEditorReadyToDiscard()))
      return "An editor is still processing changes. Please wait before saving.";
    return undefined;
  };
  return {
    schema,
    schemaQuery,
    sourceSchemaError: changingConfiguration && sourceSchemaQuery.isError,
    retrySchemas: () =>
      Promise.all([schemaQuery.refetch(), ...(changingConfiguration ? [sourceSchemaQuery.refetch()] : [])]),
    choicesQuery,
    selected,
    unavailable,
    hierarchyMismatch,
    errors,
    controls,
    editors,
    onEditorStateChange: saved.refresh,
    computed: preview.computed,
    previewError: preview.error,
    validate,
    submitted: () =>
      schema
        ? {
            ...(changingConfiguration && sourceSchemaQuery.data
              ? submittedFormerPropertyValues(customValues, sourceSchemaQuery.data, schema, saved.baseline.current)
              : {}),
            ...submittedPropertyValues(customValues, schema, data?.id ? saved.baseline.current : undefined),
          }
        : {},
    setValue: (id: string, value: unknown) => {
      form.setFieldValue(`custom_properties.${id}`, value);
      form.setFieldMeta(`custom_properties.${id}`, (meta) => ({
        ...meta,
        errorMap: { ...meta.errorMap, onServer: undefined },
      }));
    },
    clearErrors: () => clearPropertyErrors(form),
    recordError: (cause: unknown) => {
      const fields = propertyFieldErrors(cause);
      pendingErrorFocus.current = schema?.properties.find((property) => fields[property.id])?.id;
      clearPropertyErrors(form);
      for (const [id, error] of Object.entries(fields)) {
        const name = `custom_properties.${id}` as const;
        if (form.getFieldMeta(name)) {
          form.setFieldMeta(name, (meta) => ({ ...meta, errorMap: { ...meta.errorMap, onServer: error } }));
        }
      }
    },
    syncEditorsAfterReset: () => {
      for (const [id, editor] of editors) {
        const value = form.state.values.custom_properties?.[id];
        editor.setEditorValue(typeof value === "string" ? value : "", false);
      }
    },
  };
}
function clearPropertyErrors(form: IssueDraftForm) {
  for (const name of Object.keys(form.state.fieldMeta)) {
    if (!name.startsWith("custom_properties.")) continue;
    const id = name.slice("custom_properties.".length);
    form.setFieldMeta(`custom_properties.${id}`, (meta) => ({
      ...meta,
      errorMap: { ...meta.errorMap, onServer: undefined },
    }));
  }
}

export type ItemTypeFields = ReturnType<typeof useItemTypeFields>;

function configurationChanged(values: IssueDraft, data?: Partial<TIssue>) {
  return (
    !!data?.id &&
    ((values.type_id ?? "") !== (data.type_id ?? "") || (values.project_id ?? "") !== (data.project_id ?? ""))
  );
}
function useSourceItemSchema(
  userId: string | undefined,
  workspaceSlug: string,
  values: IssueDraft,
  data: Partial<TIssue> | undefined,
  mode: ItemSchemaInput["mode"]
) {
  const changingConfiguration = configurationChanged(values, data);
  const sourceSchemaQuery = useQuery({
    ...itemSchemaOptions({
      userId,
      workspaceSlug,
      input: {
        mode: mode === "publish" ? "draft" : mode,
        project_id: data?.project_id || undefined,
        type_id: data?.type_id ?? "",
        ...(mode === "edit" ? { issue_id: data?.id } : { draft_id: data?.id }),
      },
    }),
    enabled: changingConfiguration,
  });
  return { changingConfiguration, sourceSchemaQuery };
}
function validateTypeSelection(
  schema: ItemSchema,
  typeId: IssueDraft["type_id"],
  mode: ItemSchemaInput["mode"],
  unavailable: boolean,
  keepsSavedType: boolean,
  hierarchyMismatch: boolean
) {
  if (unavailable && !(keepsSavedType && (mode === "edit" || mode === "draft")))
    return "Choose an available work item type for this project.";
  if (schema.enabled && !typeId && mode !== "draft") return schema.selection_reason || "Choose a work item type.";
  if (hierarchyMismatch) return "Choose a type from the level immediately below this parent.";
  return undefined;
}

function typeSelectionState(
  values: IssueDraft,
  data: Partial<TIssue> | undefined,
  schema: ItemSchema | undefined,
  catalog: { enabled: boolean; types: (WorkItemType & { available: boolean })[] } | undefined
) {
  const selected = catalog?.types.find((type) => type.id === values.type_id);
  const unavailable = Boolean(values.type_id && catalog?.enabled && !selected?.available);
  const hierarchyMismatch = Boolean(
    values.type_id && selected?.available && schema?.enabled && !schema.eligible_type_ids.includes(values.type_id)
  );
  const keepsSavedType = !!data?.id && data.type_id === values.type_id && data.project_id === values.project_id;
  return { selected, unavailable, hierarchyMismatch, keepsSavedType };
}

function itemScope(mode: ItemSchemaInput["mode"], values: IssueDraft, data?: Partial<TIssue>): ItemSchemaInput {
  return {
    mode,
    project_id: values.project_id || undefined,
    ...(mode === "edit" ? { issue_id: data?.id } : { draft_id: data?.id }),
    type_id: values.type_id ?? "",
    ...(!data?.id || (values.parent_id ?? null) !== (data.parent_id ?? null)
      ? { parent_id: values.parent_id ?? "" }
      : {}),
  };
}
function useSchemaDefaults(
  form: IssueDraftForm,
  schema: ItemSchema | undefined,
  customValues: Record<string, unknown>,
  creating: boolean,
  typeId: string | null
) {
  useEffect(() => {
    if (!schema) return;
    if (creating && !typeId && schema.type_id && !form.getFieldMeta("type_id")?.isDirty) {
      form.setFieldValue("type_id", schema.type_id, { dontUpdateMeta: true });
    }
    const next = applyPropertyDefaults(customValues, schema, creating);
    if (next !== customValues) form.setFieldValue("custom_properties", next, { dontUpdateMeta: true });
  }, [schema, customValues, creating, form, typeId]);
}
function useFormulaPreview(
  userId: string | undefined,
  workspaceSlug: string,
  scope: ItemSchemaInput,
  schema: ItemSchema | undefined,
  values: IssueDraft,
  data?: Partial<TIssue>
) {
  const previewBody = useMemo(
    () => ({
      name: values.name,
      priority: values.priority ?? "none",
      start_date: values.start_date ?? "",
      target_date: values.target_date ?? "",
      ...(values.estimate_point === data?.estimate_point ? {} : { estimate_point: values.estimate_point ?? "" }),
      custom_properties: schema ? submittedPropertyValues(values.custom_properties ?? {}, schema) : {},
    }),
    [
      values.name,
      values.priority,
      values.start_date,
      values.target_date,
      values.estimate_point,
      data?.estimate_point,
      values.custom_properties,
      schema,
    ]
  );
  const debounced = useDebounce(previewBody, 250);
  const hasFormulas = schema?.properties.some((property) => property.kind === "Formula") ?? false;
  const previewQuery = useQuery({
    queryKey: ["workspace", workspaceSlug, "item-formula-preview", userId, scope, debounced] as const,
    queryFn:
      userId && schema && hasFormulas && previewBody === debounced
        ? ({ signal }) => workItemSchemaService.preview(workspaceSlug, scope, debounced, signal)
        : skipToken,
    retry: false,
    refetchInterval: hasFormulas ? nextFormulaClockBoundary : false,
    refetchOnWindowFocus: hasFormulas ? "always" : false,
    refetchOnMount: hasFormulas ? "always" : true,
  });
  return {
    computed: previewBody === debounced ? previewQuery.data?.computed : undefined,
    error: previewBody === debounced && previewQuery.isError,
  };
}

function useSavedItemFields(
  form: IssueDraftForm,
  editors: Map<string, EditorRefApi>,
  data?: Partial<TIssue>,
  persistedData = data
) {
  const customProperties = data?.custom_properties;
  const typeId = data?.type_id;
  const baseline = useRef(persistedData?.custom_properties ?? {});
  const refresh = useCallback(() => {
    // Recheck local edits when an upload finishes; a newer local draft wins.
    if ([...editors.values()].some((editor) => !editor.isEditorReadyToDiscard())) return;
    const propertiesDirty = Object.entries(form.state.fieldMeta).some(
      ([name, meta]) => (name === "custom_properties" || name.startsWith("custom_properties.")) && meta?.isDirty
    );
    if (customProperties && !propertiesDirty) {
      baseline.current = persistedData?.custom_properties ?? {};
      if (form.state.values.custom_properties !== customProperties)
        form.setFieldValue("custom_properties", customProperties, { dontUpdateMeta: true });
    }
    if (typeId !== undefined && !form.getFieldMeta("type_id")?.isDirty && form.state.values.type_id !== typeId) {
      form.setFieldValue("type_id", typeId, { dontUpdateMeta: true });
    }
  }, [customProperties, typeId, editors, form, persistedData?.custom_properties]);
  useEffect(refresh, [refresh]);
  return { baseline, refresh };
}

function isTransientSchemaError(error: unknown) {
  return (
    isAxiosError(error) &&
    (!error.response || error.response.status === 408 || error.response.status === 429 || error.response.status >= 500)
  );
}

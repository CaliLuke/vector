/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useEffect, useState, useRef } from "react";

import { useMutation } from "@tanstack/react-query";
import { useForm, useStore } from "@tanstack/react-form";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
// plane imports
import type { EditorRefApi, TExtensions } from "@plane/editor";
import { useTranslation } from "@plane/i18n";
import type { EFileAssetType, TNameDescriptionLoader } from "@plane/types";
import { getDescriptionPlaceholderI18n } from "@plane/utils/work-item";
// components
import { RichTextEditor } from "@/components/editor/rich-text";
// hooks
import { useAutosaveFormSubmit } from "@/hooks/use-autosave-form-submit";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
// plane web services
import { WorkspaceService } from "@/services/workspace.service";
// local imports
// services init
const workspaceService = new WorkspaceService();

type DescriptionValues = {
  description_html: string;
  description_json: object | undefined;
  isMigrationUpdate: boolean;
};
type Draft = DescriptionValues & { revision: number };
const normalize = (value: string | null | undefined) => (value?.trim() ? value : "<p></p>");

type Props = {
  mutationScope?: string;
  canEdit?: () => boolean;
  canSave?: () => boolean;
  /**
   * @description Container class name, this will be used to add custom styles to the editor container
   */
  containerClassName?: string;
  /**
   * @description Disabled, this will be used to disable the editor
   */
  disabled?: boolean;
  /**
   * @description Disabled extensions, this will be used to disable the extensions in the editor
   */
  disabledExtensions?: TExtensions[];
  /**
   * @description Editor ref, this will be used to imperatively attach editor related helper functions
   */
  editorRef?: React.RefObject<EditorRefApi | null>;
  /**
   * @description Entity ID, this will be used for file uploads and as the unique identifier for the entity
   */
  entityId: string;
  /**
   * @description File asset type, this will be used to upload the file to the editor
   */
  fileAssetType: EFileAssetType;
  /**
   * @description Initial value, pass the actual description to initialize the editor
   */
  initialValue: string | undefined;
  /**
   * @description Key, to ensure the editor is re-rendered when the key changes
   */
  /**
   * @description Submit handler, the actual function which will be called when the form is submitted
   */
  onSubmit: (
    value: {
      description_html: string;
      description_json: object | undefined;
    },
    isMigrationUpdate?: boolean
  ) => Promise<void>;
  /**
   * @description Placeholder, if not provided, the placeholder will be the default placeholder
   */
  placeholder?: string | ((isFocused: boolean, value: string) => string);
  /**
   * @description projectId, if not provided, the entity will be considered as a workspace entity
   */
  projectId?: string;
  /**
   * @description Set is submitting, use it to set the loading state of the form
   */
  setIsSubmitting: (initialValue: TNameDescriptionLoader) => void;
  /**
   * @description Description received from the server; updates the editor when it changes.
   */
  syncedDescription?: string | null | undefined;
  /**
   * @description Workspace slug, this will be used to get the workspace details
   */
  workspaceSlug: string;
  /**
   * @description Issue sequence id, this will be used to get the issue sequence id
   */
  issueSequenceId?: number;
};

/**
 * @description DescriptionInput component for rich text editor with autosave functionality using debounce
 * The component also makes an API call to save the description on unmount
 */
export function DescriptionInput(props: Props) {
  return <DescriptionEditor key={`${props.workspaceSlug}/${props.projectId}/${props.entityId}`} {...props} />;
}
const DescriptionEditor = function DescriptionEditor(props: Props) {
  const {
    mutationScope,
    canEdit,
    canSave,
    containerClassName,
    disabled,
    disabledExtensions,
    editorRef,
    entityId,
    fileAssetType,
    initialValue,
    issueSequenceId,
    onSubmit,
    placeholder,
    projectId,
    setIsSubmitting,
    syncedDescription,
    workspaceSlug,
  } = props;
  const [initial] = useState(() => normalize(initialValue));
  const [defaults, setDefaults] = useState<DescriptionValues>({
    description_html: initial,
    description_json: undefined,
    isMigrationUpdate: false,
  });
  const revision = useRef(0);
  const queuedRevision = useRef(0);
  const failedRevision = useRef<number | undefined>(undefined);
  const mounted = useRef(false);
  const saveTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const form = useForm({
    defaultValues: defaults,
    onSubmit: async ({ value, formApi }) => {
      clearTimeout(saveTimer.current);
      saveTimer.current = undefined;
      const snapshot = { ...value, revision: revision.current };
      queuedRevision.current = snapshot.revision;
      failedRevision.current = undefined;
      try {
        await save.mutateAsync(snapshot);
      } catch (cause) {
        if (revision.current === snapshot.revision) {
          formApi.setErrorMap({ onSubmit: { form: "Could not save the description.", fields: {} } });
        }
        throw cause;
      }
    },
  });
  const submissionError = useStore(form.store, (state) =>
    typeof state.errorMap.onSubmit === "string" ? state.errorMap.onSubmit : undefined
  );
  const submitting = useStore(form.store, (state) => state.isSubmitting);
  const { submit } = useAutosaveFormSubmit(form);
  const dirty = useStore(form.store, (state) => state.isDirty);
  const values = useStore(form.store, (state) => state.values);
  const receivedValue = normalize(syncedDescription ?? initialValue);
  const [previousValue, setPreviousValue] = useState(receivedValue);
  if (previousValue !== receivedValue) {
    setPreviousValue(receivedValue);
    // Refetches update pristine defaults, without replacing an unsaved draft.
    if (!dirty) setDefaults({ description_html: receivedValue, description_json: undefined, isMigrationUpdate: false });
  }
  const save = useMutation({
    scope: { id: mutationScope ?? `description/${workspaceSlug}/${projectId}/${entityId}` },
    retry: false,
    mutationFn: (snapshot: Draft) => {
      if (canSave?.() === false) throw new Error("The account session changed.");
      return onSubmit(
        { description_html: snapshot.description_html, description_json: snapshot.description_json },
        snapshot.isMigrationUpdate
      );
    },
    onSuccess: (_result, snapshot) => {
      if (failedRevision.current === snapshot.revision) failedRevision.current = undefined;
      if (mounted.current && revision.current === snapshot.revision) {
        const accepted = {
          description_html: snapshot.description_html,
          description_json: snapshot.description_json,
          isMigrationUpdate: snapshot.isMigrationUpdate,
        };
        setDefaults(accepted);
        form.reset(accepted);
        setIsSubmitting("submitted");
      }
    },
    onError: (_error, snapshot) => {
      failedRevision.current = snapshot.revision;
      if (mounted.current && revision.current === snapshot.revision) setIsSubmitting("error");
      if (!mounted.current && revision.current === snapshot.revision)
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Description not saved",
          message: "The last description edit could not be saved. Please reopen the work item and try again.",
        });
    },
  });
  const cancelQueuedSave = useCallback(() => {
    clearTimeout(saveTimer.current);
    saveTimer.current = undefined;
  }, []);
  const queueSave = useCallback(
    (snapshot: Draft) => {
      cancelQueuedSave();
      saveTimer.current = setTimeout(() => {
        saveTimer.current = undefined;
        queuedRevision.current = snapshot.revision;
        failedRevision.current = undefined;
        submit();
      }, 1500);
    },
    [cancelQueuedSave, submit]
  );
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      cancelQueuedSave();
      if (canSave?.() === false) return;
      if (revision.current > queuedRevision.current || failedRevision.current === revision.current) {
        queuedRevision.current = revision.current;
        failedRevision.current = undefined;
        submit();
      }
    };
  }, [cancelQueuedSave, submit, canSave]);
  useEffect(() => {
    if (!form.state.isDirty) form.reset(defaults);
  }, [defaults, form]);
  // store hooks
  const { data: workspaceDetails } = useWorkspaceBySlug(workspaceSlug);
  // derived values

  // translation
  const { t } = useTranslation();
  if (!workspaceDetails) return null;

  return (
    <>
      <RichTextEditor
        key={entityId}
        editable={!disabled}
        ref={editorRef}
        id={entityId}
        issueSequenceId={issueSequenceId}
        disabledExtensions={disabledExtensions}
        initialValue={initial}
        value={dirty ? null : values.description_html}
        workspaceSlug={workspaceSlug}
        workspaceId={workspaceDetails.id}
        projectId={projectId}
        dragDropEnabled
        onChange={(description_json, description_html, options) => {
          if (disabled || canEdit?.() === false || description_html === form.state.values.description_html) return;
          form.setErrorMap({ onSubmit: undefined });
          form.setFieldValue("description_html", description_html);
          form.setFieldValue("description_json", description_json);
          form.setFieldValue("isMigrationUpdate", !!options?.isMigrationUpdate);
          revision.current += 1;
          setIsSubmitting("submitting");
          queueSave({ ...form.state.values, revision: revision.current });
        }}
        placeholder={placeholder ?? ((isFocused, value) => t(getDescriptionPlaceholderI18n(isFocused, value)))}
        searchMentionCallback={async (payload) =>
          await workspaceService.searchEntity(workspaceSlug, {
            ...payload,
            project_id: projectId,
          })
        }
        containerClassName={containerClassName}
        fileAsset={{ entity_identifier: entityId, entity_type: fileAssetType }}
      />
      {submissionError && dirty && (
        <div role="alert" className="flex items-center gap-2 text-13 text-danger-primary">
          {submissionError}
          <button
            type="button"
            disabled={disabled || submitting}
            onClick={() => {
              cancelQueuedSave();
              queuedRevision.current = revision.current;
              failedRevision.current = undefined;
              setIsSubmitting("submitting");
              submit();
            }}
          >
            Retry
          </button>
        </div>
      )}
    </>
  );
};

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useStore } from "@tanstack/react-form";
import { useParams } from "@tanstack/react-router";
import React, { useState, useRef, useEffect, useEffectEvent } from "react";

import { ETabIndices } from "@plane/constants";
import type { EditorRefApi } from "@plane/editor";
import { useTranslation } from "@plane/i18n";
import { IssueFormActions } from "./form-actions";
import { FormErrors } from "@plane/ui";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { TIssue } from "@plane/types";
import { convertWorkItemDataToSearchResponse } from "@plane/utils/work-item";
import { getTabIndex } from "@plane/utils/tab-indices";
import {
  IssueDefaultProperties,
  IssueDescriptionEditor,
  IssueParentTag,
  IssueProjectSelect,
  IssueTitleInput,
} from "./components";
import { IssueTypeSelect, IssueCustomFields } from "./components/type-fields";
import { useItemTypeFields } from "./use-item-type-fields";
import type { IssueDraftForm } from "./form-state";
import { issueDraftDefaults, issueSubmissionPayload, useIssueDraftForm } from "./form-state";
import { useIssueModal } from "@/hooks/context/use-issue-modal";
import { useWorkItemDetail } from "@/hooks/use-work-item-detail";
import { useWorkspaceProjectDetails } from "@/hooks/use-projects";
import { useProjectStates } from "@/hooks/use-project-states";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { issueFieldErrors, issueSaveError, issueSourceChanged } from "./issue-field-errors";
import { ItemEditCancelled } from "./use-reviewed-item-edit";
import { itemAssetReferences } from "./item-asset-references";
import { usePlatformOS } from "@/hooks/use-platform-os";
import useKeypress from "@/hooks/use-keypress";

export interface IssueFormProps {
  data?: Partial<TIssue>;
  persistedData?: Partial<TIssue>;
  sourceVersion?: string;
  onReloadSource?: () => Promise<void>;
  issueTitleRef: React.MutableRefObject<HTMLInputElement | null>;
  isCreateMoreToggleEnabled: boolean;
  onAssetUpload: (assetId: string) => void;
  onCreateMoreToggleChange: (value: boolean) => void;
  onChange?: (formData: Partial<TIssue> | null) => void;
  onClose: () => void;
  onSubmit: (
    values: Partial<TIssue>,
    is_draft_issue?: boolean,
    assetReferences?: string[],
    revision?: number,
    editorDraft?: Partial<TIssue>,
    sourceVersion?: string
  ) => Promise<void>;
  onPublish?: (
    values: Partial<TIssue>,
    assetReferences: string[],
    revision?: number,
    editorDraft?: Partial<TIssue>,
    sourceVersion?: string
  ) => Promise<void>;
  projectId: string | null;
  isDraft: boolean;
  moveToIssue?: boolean;
  modalTitle?: string;
  primaryButtonText?: { default: string; loading: string };
  isProjectSelectionDisabled?: boolean;
  showActionButtons?: boolean;
  readOnly?: boolean;
}

export const IssueFormRoot = function IssueFormRoot(props: IssueFormProps) {
  const { t } = useTranslation();
  const { data, persistedData, version: sourceVersion } = useFormSource(props);
  const [sourceConflict, setSourceConflict] = useState(false);
  const [reloading, setReloading] = useState(false);
  const {
    issueTitleRef,
    onAssetUpload,
    onChange,
    onClose,
    onSubmit,
    onPublish,
    projectId: defaultProjectId,
    isCreateMoreToggleEnabled,
    onCreateMoreToggleChange,
    isDraft,
    moveToIssue = false,
    isProjectSelectionDisabled = false,
    showActionButtons = true,
    readOnly = false,
  } = props;
  const { modalTitle, primaryButtonText } = issueFormPresentation(props, t);
  const [publishing, setPublishing] = useState(false);
  const [gptAssistantModal, setGptAssistantModal] = useState(false);
  const [defaults] = useState(() => issueDraftDefaults(defaultProjectId, data ?? {}));
  const editorRef = useRef<EditorRefApi>(null);
  const submitBtnRef = useRef<HTMLButtonElement>(null);
  const pendingFieldFocus = useRef<string | undefined>(undefined);
  const formElement = useRef<HTMLFormElement>(null);
  const fieldsRef = useRef<ReturnType<typeof useItemTypeFields> | undefined>(undefined);
  const { workspaceSlug } = useParams({ from: "/_authenticated/$workspaceSlug" });
  const projectCatalogue = useWorkspaceProjectDetails();
  const projectRows = projectCatalogue.projects;
  const { selectedParentIssue, setSelectedParentIssue } = useIssueModal();
  const { isMobile } = usePlatformOS();
  const { getIndex } = getTabIndex(ETabIndices.ISSUE_FORM, isMobile);

  const editorReady = () => {
    if (
      editorRef.current?.isEditorReadyToDiscard() &&
      [...fields.editors.values()].every((editor) => editor.isEditorReadyToDiscard())
    )
      return true;
    setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message: t("editor_is_not_ready_to_discard_changes") });
    return false;
  };
  const form: IssueDraftForm = useIssueDraftForm(
    defaults,
    async (value, action, formApi, fieldMeta) => {
      formApi.setErrorMap({ onSubmit: undefined });
      const activeFields = fieldsRef.current;
      if (!activeFields) throw new Error("Work item fields are not ready.");
      activeFields.clearErrors();
      try {
        const dirtyFields = Object.fromEntries(
          Object.entries(fieldMeta).map(([key, meta]) => [key, isDirtyFieldMeta(meta)])
        );
        const payload = issueSubmissionPayload(value, dirtyFields, data?.id, isDraft, activeFields.submitted());
        const references = itemAssetReferences(value, activeFields.schema);
        const outcome = await submitIssueDraftAction(action, {
          data,
          onPublish,
          onSubmit,
          payload,
          isDraft,
          references,
          revision: activeFields.schema?.revision,
          draft: value,
          sourceVersion,
        });
        if (outcome === "published") return;
        setGptAssistantModal(false);
        formApi.reset(
          issueDraftDefaults(value.project_id ?? defaultProjectId, {
            ...(isCreateMoreToggleEnabled ? data : {}),
            project_id: value.project_id,
            type_id: value.type_id,
          })
        );
        editorRef.current?.setEditorValue(value.description_html ?? "", false);
        activeFields.syncEditorsAfterReset();
        if (isCreateMoreToggleEnabled) issueTitleRef.current?.focus();
      } catch (cause) {
        if (cause instanceof ItemEditCancelled) return;
        if (issueSourceChanged(cause)) setSourceConflict(true);
        activeFields.recordError(cause);
        const errors = issueFieldErrors(cause);
        pendingFieldFocus.current = Object.keys(errors)[0];
        formApi.setErrorMap({
          onSubmit: {
            form: issueSaveError(
              cause,
              action === "move"
                ? "Failed to move work item to project. Please try again."
                : "Failed to save work item. Your changes are still here; please try again."
            ),
            fields: errors,
          },
        });
        throw cause;
      }
    },
    () =>
      fieldsRef.current?.validate() ??
      (editorRef.current?.isEditorReadyToDiscard() ? undefined : t("editor_is_not_ready_to_discard_changes"))
  );
  const reloadSource = async () => {
    if (!editorReady()) return;
    setReloading(true);
    try {
      await props.onReloadSource?.();
    } catch (cause) {
      form.setErrorMap({
        onSubmit: {
          form: issueSaveError(cause, "Could not reload the saved work item. Your changes are still here."),
          fields: {},
        },
      });
    } finally {
      setReloading(false);
    }
  };
  const fields = useItemTypeFields(
    form,
    workspaceSlug,
    data,
    itemSchemaMode(publishing, isDraft, !!data?.id),
    persistedData
  );
  useEffect(() => {
    fieldsRef.current = fields;
    return () => {
      fieldsRef.current = undefined;
    };
  }, [fields]);
  const values = useStore(form.store, (state) => state.values);
  const parentDetail = useWorkItemDetail(
    workspaceSlug,
    parentDetailProjectId(values.project_id, defaultProjectId),
    values.parent_id ?? undefined
  );
  const isDirty = useStore(form.store, (state) => state.isDirty);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const submitError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const formLocked = isSubmitting || reloading;
  useEffect(() => {
    if (isSubmitting || !pendingFieldFocus.current) return;
    const field = pendingFieldFocus.current;
    const container = formElement.current?.querySelector<HTMLElement>(`[data-issue-field="${field}"]`);
    const control = container?.querySelector<HTMLElement>("input,button,[contenteditable=true],textarea") ?? container;
    control?.focus();
    pendingFieldFocus.current = undefined;
  }, [isSubmitting]);
  const projectId = values.project_id;
  const { projectRole } = useProjectPermissions(workspaceSlug, projectId ?? undefined);
  const { getStateById } = useProjectStates(workspaceSlug, projectId);
  const notifyChange = useEffectEvent((value: typeof values, dirty: boolean) => {
    onChange?.(dirty ? value : null);
  });
  useEffect(() => {
    notifyChange(values, isDirty);
  }, [values, isDirty]);
  useEffect(() => {
    issueTitleRef.current?.focus();
  }, [issueTitleRef]);
  useEffect(() => {
    if (data?.description_html !== undefined && !form.getFieldMeta("description_html")?.isDirty) {
      form.setFieldValue("description_html", data.description_html, { dontUpdateMeta: true });
    }
  }, [data?.description_html, form]);
  useEffect(() => {
    if (!values.parent_id || values.parent_id === selectedParentIssue?.id) return;
    const issue = parentDetail.issue;
    if (!issue) return;
    const parentProject = projectRows?.find((projectRow) => projectRow.id === issue.project_id);
    if (parentProject)
      setSelectedParentIssue(
        convertWorkItemDataToSearchResponse(workspaceSlug, issue, parentProject, getStateById(issue.state_id))
      );
  }, [
    values.parent_id,
    selectedParentIssue?.id,
    parentDetail.issue,
    projectRows,
    getStateById,
    setSelectedParentIssue,
    workspaceSlug,
  ]);

  const changeProject = () => {
    if (isDraft) return;
    form.setFieldValue("state_id", "");
    form.setFieldValue("assignee_ids", []);
    form.setFieldValue("label_ids", []);
    form.setFieldValue("cycle_id", null);
    form.setFieldValue("module_ids", []);
    form.setFieldValue("estimate_point", null);
    form.setFieldValue("parent_id", null);
    setSelectedParentIssue(null);
  };
  const close = () => {
    if (!isSubmitting && editorReady()) onClose();
  };
  useKeypress(
    "Escape",
    (event) => {
      if (isSubmitting || readOnly || gptAssistantModal) return;
      event.preventDefault();
      close();
    },
    formElement
  );

  return (
    <form
      ref={formElement}
      onSubmit={(event) => {
        event.preventDefault();
        event.stopPropagation();
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit({ action: publishing ? "move" : "save" }).catch(() => {});
      }}
      className="flex w-full flex-col"
    >
      <fieldset disabled={formLocked} inert={formLocked} className="min-w-0">
        <fieldset disabled={readOnly} inert={readOnly} className="min-w-0">
          <div className="rounded-t-lg bg-surface-1 p-5">
            <h3 className="pb-2 text-h4-medium text-secondary">{modalTitle}</h3>
            <div className="flex flex-wrap items-start gap-3 pt-2 pb-4">
              <IssueProjectSelect
                form={form}
                onProjectChange={changeProject}
                beforeChange={editorReady}
                disabled={projectSelectionDisabled(isSubmitting, isDraft, data, isProjectSelectionDisabled)}
                allowEmpty={isDraft && !publishing}
              />
              <form.Field name="type_id">
                {(field) => (
                  <div data-issue-field="type_id" tabIndex={-1}>
                    <IssueTypeSelect
                      fields={fields}
                      beforeChange={editorReady}
                      value={field.state.value}
                      onChange={field.handleChange}
                    />
                    <FormErrors errors={field.state.meta.errors} className="mt-1 text-12 text-danger-primary" />
                  </div>
                )}
              </form.Field>
            </div>
            {values.parent_id && selectedParentIssue && (
              <div className="pb-4">
                <IssueParentTag
                  form={form}
                  selectedParentIssue={selectedParentIssue}
                  setSelectedParentIssue={setSelectedParentIssue}
                />
              </div>
            )}
            <IssueTitleInput form={form} issueTitleRef={issueTitleRef} required={!isDraft || publishing} />
          </div>
          <div className="space-y-3 bg-surface-1 px-5 pb-4">
            <IssueDescriptionEditor
              form={form}
              isDraft={isDraft}
              disabled={isSubmitting || readOnly}
              issueName={values.name}
              issueId={data?.id}
              descriptionHtmlData={data?.description_html}
              editorRef={editorRef}
              submitBtnRef={submitBtnRef}
              gptAssistantModal={gptAssistantModal}
              workspaceSlug={workspaceSlug}
              projectId={projectRole ? projectId : null}
              setGptAssistantModal={setGptAssistantModal}
              onAssetUpload={onAssetUpload}
            />
          </div>
          <IssueCustomFields
            form={form}
            fields={fields}
            values={values}
            workspaceSlug={workspaceSlug}
            isDraft={isDraft}
            projectId={projectRole ? projectId : null}
            disabled={isSubmitting || readOnly}
            onAssetUpload={onAssetUpload}
          />
          <div className="border-t-[0.5px] border-subtle bg-surface-1 px-4 py-3">
            <div className="pb-3">
              <IssueDefaultProperties
                form={form}
                id={data?.id}
                projectId={projectRole ? projectId : null}
                workspaceSlug={workspaceSlug}
                selectedParentIssue={selectedParentIssue}
                startDate={values.start_date}
                targetDate={values.target_date}
                parentId={values.parent_id}
                isDraft={isDraft}
                setSelectedParentIssue={setSelectedParentIssue}
              />
            </div>
          </div>
        </fieldset>
        {sourceConflict && props.onReloadSource && (
          <div className="bg-surface-1 px-4 pb-3">
            <button type="button" className="text-13 text-accent-primary" onClick={() => void reloadSource()}>
              {reloading ? "Reloading…" : "Discard changes and reload saved work item"}
            </button>
          </div>
        )}
        <IssueFormActions
          form={form}
          isNew={!data?.id}
          submitError={submitError}
          showActionButtons={showActionButtons}
          isCreateMoreToggleEnabled={isCreateMoreToggleEnabled}
          onCreateMoreToggleChange={onCreateMoreToggleChange}
          close={close}
          discardTabIndex={getIndex("discard_button")}
          submitTabIndex={getIndex("submit_button")}
          moveToIssue={moveToIssue}
          publishing={publishing}
          setPublishing={(next) => {
            if (editorReady()) setPublishing(next);
          }}
          submitBtnRef={submitBtnRef}
          isSubmitting={isSubmitting}
          primaryButtonText={primaryButtonText}
        />
      </fieldset>
    </form>
  );
};

function projectSelectionDisabled(
  submitting: boolean,
  draft: boolean,
  data: Partial<TIssue> | undefined,
  disabled: boolean
) {
  return submitting || (!draft && (!!data?.id || !!data?.sourceIssueId)) || disabled;
}

function isDirtyFieldMeta(value: unknown): boolean {
  return typeof value === "object" && value !== null && "isDirty" in value && value.isDirty === true;
}

function parentDetailProjectId(projectId: string | null | undefined, defaultProjectId: string | null | undefined) {
  return projectId ?? defaultProjectId ?? undefined;
}

function useFormSource(props: IssueFormProps) {
  const [initial] = useState(() => ({
    data: props.data,
    persistedData: props.persistedData,
    version: props.sourceVersion,
  }));
  return initial.version ? initial : { data: props.data, persistedData: props.persistedData, version: initial.version };
}

async function publishIssueDraft(
  data: IssueFormProps["data"],
  publish: IssueFormProps["onPublish"],
  payload: Partial<TIssue>,
  references: string[],
  revision: number | undefined,
  draft: Partial<TIssue>,
  sourceVersion: string | undefined
) {
  if (!data?.id || !publish) throw new Error("Select a draft to publish.");
  await publish(payload, references, revision, draft, sourceVersion);
}

async function submitIssueDraftAction(
  action: "move" | "save",
  input: {
    data: IssueFormProps["data"];
    onPublish: IssueFormProps["onPublish"];
    onSubmit: IssueFormProps["onSubmit"];
    payload: Partial<TIssue>;
    isDraft: boolean;
    references: string[];
    revision: number | undefined;
    draft: Partial<TIssue>;
    sourceVersion: string | undefined;
  }
): Promise<"published" | "saved"> {
  const { data, onPublish, onSubmit, payload, isDraft, references, revision, draft, sourceVersion } = input;
  if (action === "move") {
    await publishIssueDraft(data, onPublish, payload, references, revision, draft, sourceVersion);
    return "published";
  }
  await onSubmit(payload, isDraft, references, revision, draft, sourceVersion);
  return "saved";
}

function issueFormPresentation(props: IssueFormProps, t: ReturnType<typeof useTranslation>["t"]) {
  const updating = !!props.data?.id;
  return {
    modalTitle:
      props.modalTitle ?? (updating ? t("update") : props.isDraft ? t("create_a_draft") : t("create_new_issue")),
    primaryButtonText: props.primaryButtonText ?? {
      default: updating ? t("update") : props.isDraft ? t("save_to_drafts") : t("save"),
      loading: updating ? t("updating") : t("saving"),
    },
  };
}
function itemSchemaMode(
  publishing: boolean,
  isDraft: boolean,
  existing: boolean
): Parameters<typeof useItemTypeFields>[3] {
  if (publishing) return "publish";
  if (isDraft) return "draft";
  return existing ? "edit" : "create";
}

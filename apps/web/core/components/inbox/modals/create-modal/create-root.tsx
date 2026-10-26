/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useRouter } from "@tanstack/react-router";
import { useStore } from "@tanstack/react-form";
import { useMutation } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { ETabIndices } from "@plane/constants";
import type { EditorRefApi } from "@plane/editor";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Input, ToggleSwitch } from "@plane/ui";
import { getTabIndex } from "@plane/utils/tab-indices";
import {
  issueDraftDefaults,
  issueTitleValidators,
  useIssueDraftForm,
} from "@/components/issues/issue-modal/form-state";
import { useIntakeMutation } from "@/hooks/use-intake";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import useKeypress from "@/hooks/use-keypress";
import { usePlatformOS } from "@/hooks/use-platform-os";
import { FileService } from "@/services/file.service";
import { InboxIssueDescription } from "./issue-description";
import { InboxIssueProperties } from "./issue-properties";
const fileService = new FileService();
type Props = { workspaceSlug: string; projectId: string; handleModalClose: () => void };
export const InboxIssueCreateRoot = function InboxIssueCreateRoot({
  workspaceSlug,
  projectId,
  handleModalClose,
}: Props) {
  const router = useRouter();
  const create = useIntakeMutation(workspaceSlug, projectId);
  const { data: workspaceDetails } = useWorkspaceBySlug(workspaceSlug);
  const workspaceId = workspaceDetails?.id;
  const { isMobile } = usePlatformOS();
  const { t } = useTranslation();
  const { getIndex } = getTabIndex(ETabIndices.INTAKE_ISSUE_FORM, isMobile);
  const editor = useRef<EditorRefApi>(null);
  const title = useRef<HTMLInputElement>(null);
  const submitButton = useRef<HTMLButtonElement>(null);
  const formElement = useRef<HTMLFormElement>(null);
  const [uploadedAssets, setUploadedAssets] = useState<string[]>([]);
  const [createMore, setCreateMore] = useState(false);
  const [initial] = useState(() => issueDraftDefaults(projectId, {}));
  const finalize = useMutation({
    mutationFn: (id: string) =>
      fileService.updateBulkProjectAssetsUploadStatus(workspaceSlug, projectId, id, { asset_ids: uploadedAssets }),
    retry: false,
  });
  const ready = () => {
    if (editor.current?.isEditorReadyToDiscard()) return true;
    setToast({ type: TOAST_TYPE.ERROR, title: t("error"), message: t("editor_is_not_ready_to_discard_changes") });
    return false;
  };
  const form = useIssueDraftForm(
    initial,
    async (value, _action, formApi) => {
      formApi.setErrorMap({ onSubmit: undefined });
      try {
        const created =
          create.data?.saved ??
          (
            await create.mutateAsync({
              kind: "create",
              data: {
                name: value.name,
                description_html: value.description_html,
                priority: value.priority,
                state_id: value.state_id,
                label_ids: value.label_ids,
                assignee_ids: value.assignee_ids,
                target_date: value.target_date,
              },
            })
          ).saved;
        if (!created) throw new Error("The server did not return the created intake work item.");
        if (uploadedAssets.length && !finalize.isSuccess) await finalize.mutateAsync(created.issue.id);
        setUploadedAssets([]);
        setToast({ type: TOAST_TYPE.SUCCESS, title: t("success"), message: t("issue_created_successfully") });
        if (createMore) {
          formApi.reset();
          editor.current?.clearEditor();
          create.reset();
          finalize.reset();
          title.current?.focus();
        } else {
          handleModalClose();
          void router.navigate({
            href: `/${workspaceSlug}/projects/${projectId}/intake/?currentTab=open&inboxIssueId=${created.issue.id}`,
          });
        }
      } catch (cause) {
        formApi.setErrorMap({
          onSubmit: {
            form: "Could not finish creating the work item. Your progress is saved; please try again.",
            fields: {},
          },
        });
        throw cause;
      }
    },
    () => (editor.current?.isEditorReadyToDiscard() ? undefined : t("editor_is_not_ready_to_discard_changes"))
  );
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const locked = busy || create.isSuccess;
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const close = () => {
    if (!busy && ready()) handleModalClose();
  };
  useKeypress(
    "Escape",
    (event) => {
      if (busy) return;
      event.preventDefault();
      close();
    },
    formElement
  );
  useEffect(() => {
    title.current?.focus();
  }, []);
  if (!workspaceId) return null;
  return (
    <form
      ref={formElement}
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        form.setErrorMap({ onSubmit: undefined });
        void form.handleSubmit().catch(() => {});
      }}
      className="flex w-full flex-col rounded-lg"
    >
      <fieldset disabled={locked} inert={locked} className="space-y-5 rounded-t-lg bg-surface-1 p-5">
        <h3 className="text-18 font-medium text-secondary">{t("inbox_issue.modal.title")}</h3>
        <form.Field
          name="name"
          validators={issueTitleValidators(t("title_is_required"), t("title_should_be_less_than_255_characters"))}
        >
          {(field) => (
            <div className="space-y-1">
              <Input
                ref={title}
                name={field.name}
                value={field.state.value}
                onChange={(event) => field.handleChange(event.target.value)}
                onBlur={field.handleBlur}
                hasError={!field.state.meta.isValid}
                placeholder={t("title")}
                className="w-full text-14"
                tabIndex={getIndex("name")}
              />
              {[...new Set(field.state.meta.errors)].map((message) => (
                <p key={message} role="alert" className="text-11 text-danger-primary">
                  {message}
                </p>
              ))}
            </div>
          )}
        </form.Field>
        <form.Field name="description_html">
          {(field) => (
            <InboxIssueDescription
              value={field.state.value ?? "<p></p>"}
              onChange={field.handleChange}
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              workspaceId={workspaceId}
              editorRef={editor}
              containerClassName="bg-layer-2 border-[0.5px] border-subtle-1 py-3 min-h-[150px]"
              onEnterKeyPress={() => submitButton.current?.click()}
              onAssetUpload={(id) => setUploadedAssets((ids) => [...ids, id])}
            />
          )}
        </form.Field>
        <InboxIssueProperties projectId={projectId} form={form} />
      </fieldset>
      <div className="rounded-b-lg border-t-[0.5px] border-subtle bg-surface-1 px-5 py-4">
        {error && (
          <p role="alert" className="mb-3 text-13 text-danger-primary">
            {error}
          </p>
        )}
        <div className="flex items-center justify-between gap-2">
          <label className="inline-flex cursor-pointer items-center gap-1.5">
            <ToggleSwitch value={createMore} onChange={setCreateMore} disabled={busy} size="sm" />
            <span className="text-11">{t("create_more")}</span>
          </label>
          <div className="flex items-center gap-3">
            <Button
              variant="secondary"
              size="lg"
              type="button"
              onClick={close}
              disabled={busy}
              tabIndex={getIndex("discard_button")}
            >
              {t("discard")}
            </Button>
            <Button
              variant="primary"
              ref={submitButton}
              type="submit"
              loading={busy}
              disabled={busy}
              tabIndex={getIndex("submit_button")}
              size="lg"
            >
              {busy ? t("creating") : t("create_work_item")}
            </Button>
          </div>
        </div>
      </div>
    </form>
  );
};

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IssueDraftForm } from "../form-state";
import React from "react";
import { useQuery, useMutation } from "@tanstack/react-query";

import { Sparkle } from "lucide-react";
// plane imports
import { ETabIndices } from "@plane/constants";
import type { EditorRefApi } from "@plane/editor";
import { useTranslation } from "@plane/i18n";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EFileAssetType } from "@plane/types";
import { Loader, FormErrors } from "@plane/ui";
import { getDescriptionPlaceholderI18n } from "@plane/utils/work-item";
import { getTabIndex } from "@plane/utils/tab-indices";
// components
import { GptAssistantPopover } from "@/components/core/modals/gpt-assistant-popover";
import { RichTextEditor } from "@/components/editor/rich-text";
// helpers
// hooks
import { instanceConfigOptions } from "@/queries/instance";
import { useWorkspaceBySlug } from "@/hooks/use-workspaces";
import { usePlatformOS } from "@/hooks/use-platform-os";
// plane web services
import { WorkspaceService } from "@/services/workspace.service";
// services
import { getAiErrorMessage } from "@/helpers/ai-error.helper";
import { AIService } from "@/services/ai.service";
const workspaceService = new WorkspaceService();
const aiService = new AIService();

type TIssueDescriptionEditorProps = {
  form: IssueDraftForm;
  isDraft: boolean;
  disabled: boolean;
  issueName: string;
  issueId: string | undefined;
  descriptionHtmlData: string | undefined;
  editorRef: React.MutableRefObject<EditorRefApi | null>;
  submitBtnRef: React.MutableRefObject<HTMLButtonElement | null>;
  gptAssistantModal: boolean;
  workspaceSlug: string;
  projectId: string | null;
  setGptAssistantModal: React.Dispatch<React.SetStateAction<boolean>>;
  onAssetUpload: (assetId: string) => void;
};

export const IssueDescriptionEditor = function IssueDescriptionEditor(props: TIssueDescriptionEditorProps) {
  const {
    form,
    isDraft,
    disabled,
    issueName,
    issueId,
    descriptionHtmlData,
    editorRef,
    submitBtnRef,
    gptAssistantModal,
    workspaceSlug,
    projectId,
    setGptAssistantModal,
    onAssetUpload,
  } = props;
  // i18n
  const { t } = useTranslation();
  // states
  // store hooks
  const { data: workspaceDetails } = useWorkspaceBySlug(workspaceSlug);
  const workspaceId = workspaceDetails?.id ?? "";
  const { data: config } = useQuery(instanceConfigOptions);
  // platform
  const { isMobile } = usePlatformOS();

  const { getIndex } = getTabIndex(ETabIndices.ISSUE_FORM, isMobile);

  // handlers
  const handleAiAssistance = (response: string) => {
    if (!workspaceSlug || !projectId) return;

    editorRef.current?.setEditorValueAtCursorPosition(response);
  };

  const generate = useMutation({
    mutationFn: () =>
      aiService.createGptTask(workspaceSlug, {
        prompt: issueName,
        task: "Generate a proper description for this work item.",
      }),
    onSuccess: (response) => {
      if (response.response === "")
        setToast({
          type: TOAST_TYPE.ERROR,
          title: "Error!",
          message:
            "Work item title isn't informative enough to generate the description. Please try with a different title.",
        });
      else editorRef.current?.setEditorValueAtCursorPosition(response.response_html);
    },
    onError: (error) => setToast({ type: TOAST_TYPE.ERROR, title: "Error!", message: getAiErrorMessage(error) }),
  });

  return (
    <div data-issue-field="description_html" className="relative rounded-lg border-[0.5px] border-subtle-1 bg-layer-2">
      {descriptionHtmlData === undefined ? (
        <Loader className="max-h-64 min-h-[120px] space-y-2 overflow-hidden rounded-md border border-subtle p-3 py-2 pt-3">
          <Loader.Item width="100%" height="26px" />
          <div className="flex items-center gap-2">
            <Loader.Item width="26px" height="26px" />
            <Loader.Item width="400px" height="26px" />
          </div>
          <div className="flex items-center gap-2">
            <Loader.Item width="26px" height="26px" />
            <Loader.Item width="400px" height="26px" />
          </div>
          <Loader.Item width="80%" height="26px" />
          <div className="flex items-center gap-2">
            <Loader.Item width="50%" height="26px" />
          </div>
          <div className="border-0.5 absolute right-3.5 bottom-2 z-10 flex items-center gap-2">
            <Loader.Item width="100px" height="26px" />
            <Loader.Item width="50px" height="26px" />
          </div>
        </Loader>
      ) : (
        <>
          <form.Field name="description_html">
            {({ state: { value, meta }, handleChange: onChange }) => (
              <>
                <RichTextEditor
                  editable
                  id="issue-modal-editor"
                  initialValue={value ?? ""}
                  value={null}
                  workspaceSlug={workspaceSlug}
                  workspaceId={workspaceId}
                  projectId={projectId ?? undefined}
                  onChange={(_description: object, description_html: string) => {
                    onChange(description_html);
                  }}
                  onEnterKeyPress={() => submitBtnRef?.current?.click()}
                  ref={editorRef}
                  tabIndex={getIndex("description_html")}
                  placeholder={(isFocused, description) => t(getDescriptionPlaceholderI18n(isFocused, description))}
                  searchMentionCallback={(payload) =>
                    workspaceService.searchEntity(workspaceSlug, {
                      ...payload,
                      ...(projectId ? { project_id: projectId } : {}),
                    })
                  }
                  containerClassName="pt-3 min-h-[120px]"
                  fileAsset={{
                    entity_identifier: issueId ?? "",
                    entity_type: isDraft ? EFileAssetType.DRAFT_ISSUE_DESCRIPTION : EFileAssetType.ISSUE_DESCRIPTION,
                  }}
                  onAssetUpload={onAssetUpload}
                />
                <FormErrors errors={meta.errors} className="px-3 text-12 text-danger-primary" />
              </>
            )}
          </form.Field>
          <div className="border-0.5 z-10 flex items-center justify-end gap-2 p-3">
            {issueName && issueName.trim() !== "" && config?.has_llm_configured && (
              <button
                type="button"
                className={`flex items-center gap-1 rounded-sm bg-surface-2 px-1.5 py-1 text-caption-sm-regular hover:bg-layer-1 ${
                  generate.isPending ? "cursor-wait" : ""
                }`}
                onClick={() => generate.mutate()}
                disabled={disabled || generate.isPending}
                tabIndex={getIndex("feeling_lucky")}
              >
                {generate.isPending ? (
                  "Generating response"
                ) : (
                  <>
                    <Sparkle className="h-3.5 w-3.5" />I{"'"}m feeling lucky
                  </>
                )}
              </button>
            )}
            {config?.has_llm_configured && projectId && (
              <GptAssistantPopover
                isOpen={gptAssistantModal}
                handleClose={() => setGptAssistantModal(false)}
                onOpen={() => setGptAssistantModal(true)}
                disabled={disabled}
                onResponse={(response) => {
                  handleAiAssistance(response);
                }}
                placement="top-end"
                button={
                  <>
                    <Sparkle className="h-4 w-4" />
                    AI
                  </>
                }
                workspaceId={workspaceId}
                workspaceSlug={workspaceSlug}
                projectId={projectId}
              />
            )}
          </div>
        </>
      )}
    </div>
  );
};

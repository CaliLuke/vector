/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { executeToolbarCommand } from "@plane/editor";
import { useEditorHandlers } from "@/hooks/editor/use-editor-handlers";
import React, { useState } from "react";
// plane constants
import type { EIssueCommentAccessSpecifier } from "@plane/constants";
// plane imports
import { LiteTextEditorWithRef } from "@plane/editor";
import type { EditorRefApi, ILiteTextEditorProps } from "@plane/editor";
import { useTranslation } from "@plane/i18n";
import type { TFileEntityInfo, MakeOptional } from "@plane/types";
import { cn } from "@plane/utils/common";
import { isCommentEmpty } from "@plane/utils/string";
// components

import { IssueCommentToolbar } from "@/components/editor/lite-text/toolbar";
// hooks

// plane web hooks
import { DISABLED_EDITOR_EXTENSIONS } from "@/components/editor/extensions";
// plane web service
import { WorkspaceService } from "@/services/workspace.service";
import { LiteToolbar } from "./lite-toolbar";
const workspaceService = new WorkspaceService();

type LiteTextEditorWrapperProps = MakeOptional<
  Omit<ILiteTextEditorProps, "fileHandler" | "mentionHandler" | "extendedEditorProps">,
  "disabledExtensions" | "flaggedExtensions" | "getEditorMetaData"
> & {
  workspaceSlug: string;
  workspaceId: string;
  projectId?: string;
  accessSpecifier?: EIssueCommentAccessSpecifier;
  handleAccessChange?: (accessKey: EIssueCommentAccessSpecifier) => void;
  showAccessSpecifier?: boolean;
  showSubmitButton?: boolean;
  isSubmitting?: boolean;
  showToolbarInitially?: boolean;
  variant?: "full" | "lite" | "none";
  issue_id?: string;
  parentClassName?: string;
  editorClassName?: string;
  submitButtonText?: string;
} & (
    | {
        editable: false;
      }
    | {
        editable: true;
        fileAsset: TFileEntityInfo;
        onAssetUpload?: (assetId: string) => void;
      }
  );

export const LiteTextEditor = React.forwardRef(function LiteTextEditor(
  props: LiteTextEditorWrapperProps,
  ref: React.ForwardedRef<EditorRefApi>
) {
  const { t } = useTranslation();
  const {
    containerClassName,
    editable,
    workspaceSlug,
    workspaceId,
    projectId,
    issue_id,
    accessSpecifier,
    handleAccessChange,
    showAccessSpecifier,
    showSubmitButton,
    isSubmitting,
    showToolbarInitially = true,
    variant = "full",
    parentClassName,
    placeholder = t("issue.comments.placeholder"),
    disabledExtensions: additionalDisabledExtensions = [],
    editorClassName = "",
    showPlaceholderOnEmpty = true,
    submitButtonText,
    ...rest
  } = props;
  // states
  const isLiteVariant = variant === "lite";
  const isFullVariant = variant === "full";
  const [isFocused, setIsFocused] = useState(isFullVariant ? showToolbarInitially : true);
  const [editorRef, setEditorRef] = useState<EditorRefApi | null>(null);
  const { fileHandler, getEditorMetaData, mentionHandler } = useEditorHandlers({
    ...props,
    searchMentionCallback: async (payload) =>
      workspaceService.searchEntity(workspaceSlug, { ...payload, project_id: projectId, issue_id }),
  });
  // derived values
  const isEmpty = isCommentEmpty(props.initialValue);

  const inlineEditable = isLiteVariant && editable;
  const toolbar = (
    <CommentEditorToolbar
      variant={variant}
      editable={editable}
      editorRef={editorRef}
      isFocused={isFocused}
      isEmpty={isEmpty}
      onEnterKeyPress={rest.onEnterKeyPress}
      isSubmitting={isSubmitting}
      accessSpecifier={accessSpecifier}
      handleAccessChange={handleAccessChange}
      showAccessSpecifier={showAccessSpecifier}
      showSubmitButton={showSubmitButton}
      submitButtonText={submitButtonText}
    />
  );

  return (
    <div
      className={cn(
        "relative rounded-sm border border-subtle",
        {
          "p-3": editable && !isLiteVariant,
        },
        parentClassName
      )}
      onFocus={() => isFullVariant && !showToolbarInitially && setIsFocused(true)}
      onBlur={() => isFullVariant && !showToolbarInitially && setIsFocused(false)}
    >
      {/* Wrapper for lite toolbar layout */}
      <div className={cn(inlineEditable ? "flex items-end gap-1" : "")}>
        {/* Main Editor - always rendered once */}
        <div className={cn(inlineEditable ? "min-w-0 flex-1" : "")}>
          <LiteTextEditorWithRef
            ref={ref}
            disabledExtensions={[...DISABLED_EDITOR_EXTENSIONS, ...additionalDisabledExtensions]}
            editable={editable}
            flaggedExtensions={[]}
            fileHandler={fileHandler}
            getEditorMetaData={getEditorMetaData}
            handleEditorReady={(ready) => {
              if (ready) {
                setEditorRef(ref && typeof ref === "object" ? ref.current : null);
              }
            }}
            mentionHandler={mentionHandler}
            placeholder={placeholder}
            showPlaceholderOnEmpty={showPlaceholderOnEmpty}
            containerClassName={cn(containerClassName, "relative", {
              "p-2": !editable,
            })}
            extendedEditorProps={{}}
            editorClassName={editorClassName}
            {...rest}
          />
        </div>

        {isLiteVariant && toolbar}
      </div>

      {!isLiteVariant && toolbar}
    </div>
  );
});

LiteTextEditor.displayName = "LiteTextEditor";

type ToolbarProps = Pick<
  LiteTextEditorWrapperProps,
  | "variant"
  | "editable"
  | "onEnterKeyPress"
  | "isSubmitting"
  | "accessSpecifier"
  | "handleAccessChange"
  | "showAccessSpecifier"
  | "showSubmitButton"
  | "submitButtonText"
> & { editorRef: EditorRefApi | null; isFocused: boolean; isEmpty: boolean };

function CommentEditorToolbar({
  variant,
  editable,
  editorRef,
  isFocused,
  isEmpty,
  onEnterKeyPress,
  isSubmitting = false,
  accessSpecifier,
  handleAccessChange,
  showAccessSpecifier = false,
  showSubmitButton = true,
  submitButtonText = "common.comment",
}: ToolbarProps) {
  if (!editable || variant === "none") return null;
  const executeCommand = (item: Parameters<typeof executeToolbarCommand>[1]) => {
    if (editorRef) executeToolbarCommand(editorRef.executeMenuItemCommand, item);
  };
  if (variant === "lite") {
    return (
      <LiteToolbar
        executeCommand={executeCommand}
        onSubmit={(event) => onEnterKeyPress?.(event)}
        isSubmitting={isSubmitting}
        isEmpty={isEmpty}
      />
    );
  }
  return (
    <div
      className={cn(
        "origin-top overflow-hidden transition-all duration-300 ease-out",
        isFocused ? "mt-3 max-h-[200px] scale-y-100 opacity-100" : "invisible max-h-0 scale-y-0 opacity-0"
      )}
    >
      <IssueCommentToolbar
        accessSpecifier={accessSpecifier}
        executeCommand={executeCommand}
        handleAccessChange={handleAccessChange}
        handleSubmit={(event) => onEnterKeyPress?.(event)}
        isCommentEmpty={isEmpty}
        isSubmitting={isSubmitting}
        showAccessSpecifier={showAccessSpecifier}
        editorRef={editorRef}
        showSubmitButton={showSubmitButton}
        submitButtonText={submitButtonText}
      />
    </div>
  );
}

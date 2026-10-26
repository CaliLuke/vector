/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

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
    showAccessSpecifier = false,
    showSubmitButton = true,
    isSubmitting = false,
    showToolbarInitially = true,
    variant = "full",
    parentClassName = "",
    placeholder = t("issue.comments.placeholder"),
    disabledExtensions: additionalDisabledExtensions = [],
    editorClassName = "",
    showPlaceholderOnEmpty = true,
    submitButtonText = "common.comment",
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
  function isMutableRefObject<T>(ref: React.ForwardedRef<T>): ref is React.MutableRefObject<T | null> {
    return !!ref && typeof ref === "object" && "current" in ref;
  }
  // derived values
  const isEmpty = isCommentEmpty(props.initialValue);

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
      <div className={cn(isLiteVariant && editable ? "flex items-end gap-1" : "")}>
        {/* Main Editor - always rendered once */}
        <div className={cn(isLiteVariant && editable ? "min-w-0 flex-1" : "")}>
          <LiteTextEditorWithRef
            ref={ref}
            disabledExtensions={[...DISABLED_EDITOR_EXTENSIONS, ...additionalDisabledExtensions]}
            editable={editable}
            flaggedExtensions={[]}
            fileHandler={fileHandler}
            getEditorMetaData={getEditorMetaData}
            handleEditorReady={(ready) => {
              if (ready) {
                setEditorRef(isMutableRefObject<EditorRefApi>(ref) ? ref.current : null);
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

        {/* Lite Toolbar - conditionally rendered */}
        {isLiteVariant && editable && (
          <LiteToolbar
            executeCommand={(item) => {
              // TODO: update this while toolbar homogenization
              // @ts-expect-error type mismatch here
              editorRef?.executeMenuItemCommand({
                itemKey: item.itemKey,
                ...item.extraProps,
              });
            }}
            onSubmit={(e) => rest.onEnterKeyPress?.(e)}
            isSubmitting={isSubmitting}
            isEmpty={isEmpty}
          />
        )}
      </div>

      {/* Full Toolbar - conditionally rendered */}
      {isFullVariant && editable && (
        <div
          className={cn(
            "origin-top overflow-hidden transition-all duration-300 ease-out",
            isFocused ? "mt-3 max-h-[200px] scale-y-100 opacity-100" : "invisible max-h-0 scale-y-0 opacity-0"
          )}
        >
          <IssueCommentToolbar
            accessSpecifier={accessSpecifier}
            executeCommand={(item) => {
              // TODO: update this while toolbar homogenization
              // @ts-expect-error type mismatch here
              editorRef?.executeMenuItemCommand({
                itemKey: item.itemKey,
                ...item.extraProps,
              });
            }}
            handleAccessChange={handleAccessChange}
            handleSubmit={(e) => rest.onEnterKeyPress?.(e)}
            isCommentEmpty={isEmpty}
            isSubmitting={isSubmitting}
            showAccessSpecifier={showAccessSpecifier}
            editorRef={editorRef}
            showSubmitButton={showSubmitButton}
            submitButtonText={submitButtonText}
          />
        </div>
      )}
    </div>
  );
});

LiteTextEditor.displayName = "LiteTextEditor";

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React from "react";
// plane imports
import { LiteTextEditorWithRef } from "@plane/editor";
import type { EditorRefApi, ILiteTextEditorProps, TFileHandler } from "@plane/editor";
import type { MakeOptional } from "@plane/types";
import { cn } from "@plane/utils/common";
import { isCommentEmpty } from "@plane/utils/string";
// helpers
import { getEditorFileHandlers } from "@/helpers/editor.helper";
// hooks
import { useParseEditorContent } from "@/hooks/use-parse-editor-content";
// local imports
import { EditorMentionsRoot } from "./embeds/mentions";
import { IssueCommentToolbar } from "./toolbar";

type LiteTextEditorWrapperProps = MakeOptional<
  Omit<ILiteTextEditorProps, "fileHandler" | "mentionHandler" | "extendedEditorProps">,
  "disabledExtensions" | "flaggedExtensions" | "getEditorMetaData"
> & {
  anchor: string;
  isSubmitting?: boolean;
  showSubmitButton?: boolean;
  workspaceId: string;
} & (
    | {
        editable: false;
      }
    | {
        editable: true;
        uploadFile: TFileHandler["upload"];
      }
  );

export const LiteTextEditor = React.forwardRef(function LiteTextEditor(
  props: LiteTextEditorWrapperProps,
  ref: React.ForwardedRef<EditorRefApi>
) {
  const {
    anchor,
    containerClassName,
    disabledExtensions: additionalDisabledExtensions = [],
    editable,
    isSubmitting = false,
    showSubmitButton = true,
    workspaceId,
    ...rest
  } = props;
  function isMutableRefObject<T>(ref: React.ForwardedRef<T>): ref is React.MutableRefObject<T | null> {
    return !!ref && typeof ref === "object" && "current" in ref;
  }
  // derived values
  const isEmpty = isCommentEmpty(props.initialValue);
  const editorRef = isMutableRefObject<EditorRefApi>(ref) ? ref.current : null;
  // parse content
  const { getEditorMetaData } = useParseEditorContent({
    anchor,
  });

  return (
    <div className="space-y-3 rounded-sm border border-subtle p-3">
      <LiteTextEditorWithRef
        ref={ref}
        disabledExtensions={additionalDisabledExtensions}
        flaggedExtensions={[]}
        editable={editable}
        fileHandler={getEditorFileHandlers({
          anchor,
          uploadFile: editable ? props.uploadFile : async () => "",
          workspaceId,
        })}
        getEditorMetaData={getEditorMetaData}
        mentionHandler={{
          renderComponent: (props) => <EditorMentionsRoot {...props} />,
        }}
        extendedEditorProps={{}}
        {...rest}
        // overriding the containerClassName to add relative class passed
        containerClassName={cn(containerClassName, "relative")}
      />
      {editable && (
        <IssueCommentToolbar
          executeCommand={(item) => {
            // TODO: update this while toolbar homogenization
            // @ts-expect-error type mismatch here
            editorRef?.executeMenuItemCommand({
              itemKey: item.itemKey,
              ...item.extraProps,
            });
          }}
          isSubmitting={isSubmitting}
          showSubmitButton={showSubmitButton}
          handleSubmit={(e) => rest.onEnterKeyPress?.(e)}
          isCommentEmpty={isEmpty}
          editorRef={editorRef}
        />
      )}
    </div>
  );
});

LiteTextEditor.displayName = "LiteTextEditor";

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TEditableEditorProps } from "@/hooks/editor/use-editor-handlers";
import { useEditorHandlers } from "@/hooks/editor/use-editor-handlers";
import { forwardRef } from "react";
// plane imports
import { RichTextEditorWithRef } from "@plane/editor";
import type { EditorRefApi, IRichTextEditorProps } from "@plane/editor";
import type { MakeOptional } from "@plane/types";
import { cn } from "@plane/utils/common";
// components

// hooks

// plane web hooks
import { DISABLED_EDITOR_EXTENSIONS } from "@/components/editor/extensions";

type RichTextEditorWrapperProps = MakeOptional<
  Omit<IRichTextEditorProps, "fileHandler" | "mentionHandler" | "extendedEditorProps">,
  "disabledExtensions" | "editable" | "flaggedExtensions" | "getEditorMetaData"
> & {
  workspaceSlug: string;
  workspaceId: string;
  projectId?: string;
  issueSequenceId?: number;
} & TEditableEditorProps;

export const RichTextEditor = forwardRef(function RichTextEditor(
  props: RichTextEditorWrapperProps,
  ref: React.ForwardedRef<EditorRefApi>
) {
  const {
    containerClassName,
    editable,
    workspaceSlug,
    workspaceId,
    projectId,
    disabledExtensions: additionalDisabledExtensions = [],
    ...rest
  } = props;
  const { fileHandler, getEditorMetaData, mentionHandler } = useEditorHandlers(props);
  return (
    <RichTextEditorWithRef
      ref={ref}
      disabledExtensions={[...DISABLED_EDITOR_EXTENSIONS, ...(additionalDisabledExtensions ?? [])]}
      editable={editable}
      flaggedExtensions={[]}
      fileHandler={fileHandler}
      getEditorMetaData={getEditorMetaData}
      mentionHandler={mentionHandler}
      extendedEditorProps={{}}
      {...rest}
      containerClassName={cn("relative pb-3 pl-3", containerClassName)}
    />
  );
});

RichTextEditor.displayName = "RichTextEditor";

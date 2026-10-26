/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { TEditableEditorProps } from "@/hooks/editor/use-editor-handlers";
import { useEditorHandlers } from "@/hooks/editor/use-editor-handlers";
import { forwardRef } from "react";
// plane imports
import { DocumentEditorWithRef } from "@plane/editor";
import type { IEditorPropsExtended, EditorRefApi, IDocumentEditorProps } from "@plane/editor";
import type { MakeOptional } from "@plane/types";
import { cn } from "@plane/utils/common";
// hooks

import { DISABLED_EDITOR_EXTENSIONS } from "@/components/editor/extensions";
// local imports

type DocumentEditorWrapperProps = MakeOptional<
  Omit<IDocumentEditorProps, "fileHandler" | "mentionHandler" | "user" | "extendedEditorProps">,
  "disabledExtensions" | "editable" | "flaggedExtensions" | "getEditorMetaData"
> & {
  extendedEditorProps?: Partial<IEditorPropsExtended>;
  workspaceSlug: string;
  workspaceId: string;
  projectId?: string;
} & TEditableEditorProps;

export const DocumentEditor = forwardRef(function DocumentEditor(
  props: DocumentEditorWrapperProps,
  ref: React.ForwardedRef<EditorRefApi>
) {
  const {
    containerClassName,
    editable,
    extendedEditorProps,
    workspaceSlug,
    workspaceId,
    projectId,
    disabledExtensions: additionalDisabledExtensions = [],
    ...rest
  } = props;
  const { fileHandler, getEditorMetaData, mentionHandler } = useEditorHandlers({
    ...props,
    enableAdvancedMentions: true,
  });
  return (
    <DocumentEditorWithRef
      ref={ref}
      disabledExtensions={[...DISABLED_EDITOR_EXTENSIONS, ...(additionalDisabledExtensions ?? [])]}
      editable={editable}
      flaggedExtensions={[]}
      fileHandler={fileHandler}
      getEditorMetaData={getEditorMetaData}
      mentionHandler={mentionHandler}
      extendedEditorProps={extendedEditorProps}
      {...rest}
      containerClassName={cn("relative pb-3 pl-3", containerClassName)}
    />
  );
});

DocumentEditor.displayName = "DocumentEditor";

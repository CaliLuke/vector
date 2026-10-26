/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { forwardRef } from "react";
// plane imports
import { RichTextEditorWithRef } from "@plane/editor";
import type { EditorRefApi, IRichTextEditorProps, TFileHandler } from "@plane/editor";
import type { MakeOptional } from "@plane/types";
// helpers
import { getEditorFileHandlers } from "@/helpers/editor.helper";
// hooks
import { membersOptions } from "@/queries/lookups";
import { useParseEditorContent } from "@/hooks/use-parse-editor-content";
// local imports
import { EditorMentionsRoot } from "./embeds/mentions";

type RichTextEditorWrapperProps = MakeOptional<
  Omit<IRichTextEditorProps, "editable" | "fileHandler" | "mentionHandler" | "extendedEditorProps">,
  "disabledExtensions" | "flaggedExtensions" | "getEditorMetaData"
> & {
  anchor: string;
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

export const RichTextEditor = forwardRef(function RichTextEditor(
  props: RichTextEditorWrapperProps,
  ref: React.ForwardedRef<EditorRefApi>
) {
  const {
    anchor,
    containerClassName,
    editable,
    workspaceId,
    disabledExtensions: additionalDisabledExtensions = [],
    ...rest
  } = props;
  const { data: members = [] } = useQuery(membersOptions(anchor));
  // parse content
  const { getEditorMetaData } = useParseEditorContent({
    anchor,
  });

  return (
    <RichTextEditorWithRef
      mentionHandler={{
        renderComponent: (props) => <EditorMentionsRoot {...props} />,
        getMentionedEntityDetails: (id: string) => ({
          display_name: members.find((item) => item.member === id)?.member__display_name ?? "",
        }),
      }}
      ref={ref}
      disabledExtensions={additionalDisabledExtensions}
      editable={editable}
      fileHandler={getEditorFileHandlers({
        anchor,
        uploadFile: editable ? props.uploadFile : async () => "",
        workspaceId,
      })}
      getEditorMetaData={getEditorMetaData}
      flaggedExtensions={[]}
      extendedEditorProps={{}}
      {...rest}
      containerClassName={containerClassName}
      editorClassName="min-h-[100px] py-2 overflow-hidden"
      displayConfig={{ fontSize: "large-font" }}
    />
  );
});

RichTextEditor.displayName = "RichTextEditor";

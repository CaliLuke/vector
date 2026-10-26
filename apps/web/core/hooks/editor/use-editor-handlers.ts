/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import type { IRichTextEditorProps } from "@plane/editor";
import type { TFileEntityInfo, TSearchEntityRequestPayload, TSearchResponse } from "@plane/types";
import { EditorMentionsRoot } from "@/components/editor/embeds/mentions";
import { useWorkspaceMemberRoster } from "@/hooks/use-member-rosters";
import { useParseEditorContent } from "@/hooks/use-parse-editor-content";
import { useEditorConfig } from "./use-editor-config";
import { useEditorMention } from "./use-editor-mention";

type SearchMentions = (payload: TSearchEntityRequestPayload) => Promise<TSearchResponse>;

export type TEditableEditorProps =
  | { editable: false; searchMentionCallback?: SearchMentions }
  | {
      editable: true;
      fileAsset: TFileEntityInfo;
      onAssetUpload?: (assetId: string) => void;
      searchMentionCallback: SearchMentions;
    };

type Props = TEditableEditorProps & {
  workspaceSlug: string;
  workspaceId: string;
  projectId?: string;
  enableAdvancedMentions?: boolean;
};

export function useEditorHandlers(props: Props) {
  const { workspaceSlug, workspaceId, projectId, enableAdvancedMentions } = props;
  const { membersById } = useWorkspaceMemberRoster(workspaceSlug);
  const fileHandler = useEditorConfig({
    workspaceSlug,
    workspaceId,
    projectId,
    fileAsset: props.editable ? props.fileAsset : undefined,
    onAssetUpload: props.editable ? props.onAssetUpload : undefined,
  });
  const { getEditorMetaData } = useParseEditorContent({ workspaceSlug, projectId });
  const { fetchMentions } = useEditorMention({
    enableAdvancedMentions,
    searchEntity: props.editable ? props.searchMentionCallback : async () => ({}),
  });
  const mentionHandler: IRichTextEditorProps["mentionHandler"] = {
    searchCallback: async (query) => {
      const result = await fetchMentions(query);
      if (!result) throw new Error("Failed in fetching mentions");
      return result;
    },
    renderComponent: EditorMentionsRoot,
    getMentionedEntityDetails: (id) => {
      const member = membersById?.get(id)?.member;
      return member ? { display_name: member.display_name } : undefined;
    },
  };
  return { fileHandler, getEditorMetaData, mentionHandler };
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mergeAttributes } from "@tiptap/core";
import type { Editor } from "@tiptap/core";
import Mention from "@tiptap/extension-mention";
import type { MarkdownSerializerState } from "prosemirror-markdown";
import type { Node as NodeType } from "@tiptap/pm/model";
// types
import type { TMentionHandler } from "@/types";
// local types
import type { TMentionComponentAttributes } from "./types";
import { EMentionComponentAttributeNames } from "./types";

export const CustomMentionExtensionConfig = Mention.extend({
  addAttributes() {
    return {
      [EMentionComponentAttributeNames.ID]: {
        default: null,
      },
      [EMentionComponentAttributeNames.ENTITY_IDENTIFIER]: {
        default: null,
      },
      [EMentionComponentAttributeNames.ENTITY_NAME]: {
        default: null,
      },
    };
  },

  parseHTML() {
    return [
      {
        tag: "mention-component",
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return ["mention-component", mergeAttributes(HTMLAttributes)];
  },

  renderText({ node }) {
    const getMentionedEntityDetails =
      this.editor?.storage.utility.identityHandlers.mentionHandler.getMentionedEntityDetails;
    return getMentionDisplayText(getMentionedEntityDetails, node);
  },

  addStorage() {
    return {
      markdown: {
        serialize(this: { editor: Editor }, state: MarkdownSerializerState, node: NodeType) {
          const getMentionedEntityDetails =
            this.editor.storage.utility.identityHandlers.mentionHandler.getMentionedEntityDetails;
          state.write(getMentionDisplayText(getMentionedEntityDetails, node));
        },
      },
    };
  },
});

function getMentionDisplayText(
  getMentionedEntityDetails: TMentionHandler["getMentionedEntityDetails"],
  node: NodeType
): string {
  const attrs = node.attrs as TMentionComponentAttributes;
  const mentionEntityId = attrs[EMentionComponentAttributeNames.ENTITY_IDENTIFIER];
  const mentionEntityDetails = getMentionedEntityDetails?.(mentionEntityId ?? "");
  return `@${mentionEntityDetails?.display_name ?? attrs[EMentionComponentAttributeNames.ID] ?? mentionEntityId}`;
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import type { NodeType } from "@tiptap/pm/model";
import { Selection } from "@tiptap/pm/state";
import ts from "highlight.js/lib/languages/typescript";
import { common, createLowlight } from "lowlight";
// components
import { CodeBlockLowlight } from "./code-block-lowlight";

const lowlight = createLowlight(common);
lowlight.register("ts", ts);

function codeCursor(editor: Editor, type: NodeType) {
  const { selection } = editor.state;
  if (!selection.empty || selection.$from.parent.type !== type) return null;
  return selection.$from;
}

export const CustomCodeBlockExtensionWithoutProps = CodeBlockLowlight.extend({
  addKeyboardShortcuts() {
    return {
      Tab: ({ editor }) => {
        try {
          const $from = codeCursor(editor, this.type);
          if (!$from) return false;

          // Use ProseMirror's insertText transaction to insert the tab character
          const tr = editor.state.tr.insertText("\t", $from.pos, $from.pos);
          editor.view.dispatch(tr);

          return true;
        } catch (error) {
          console.error("Error handling Tab in CustomCodeBlockExtension:", error);
          return false;
        }
      },
      ArrowUp: ({ editor }) => {
        try {
          const $from = codeCursor(editor, this.type);
          if (!$from) return false;

          const isAtStart = $from.parentOffset === 0;

          if (!isAtStart) {
            return false;
          }

          // Check if codeBlock is the first node
          const isFirstNode = $from.depth === 1 && $from.index($from.depth - 1) === 0;

          if (isFirstNode) {
            // Insert a new paragraph at the start of the document and move the cursor to it
            return editor.commands.command(({ tr }) => {
              const node = editor.schema.nodes.paragraph.create();
              tr.insert(0, node);
              tr.setSelection(Selection.near(tr.doc.resolve(1)));
              return true;
            });
          }

          return false;
        } catch (error) {
          console.error("Error handling ArrowUp in CustomCodeBlockExtension:", error);
          return false;
        }
      },
      ArrowDown: ({ editor }) => {
        try {
          if (!this.options.exitOnArrowDown) {
            return false;
          }

          const { doc } = editor.state;
          const $from = codeCursor(editor, this.type);
          if (!$from) return false;

          const isAtEnd = $from.parentOffset === $from.parent.nodeSize - 2;

          if (!isAtEnd) {
            return false;
          }

          const after = $from.after();

          if (after === undefined) {
            return false;
          }

          const nodeAfter = doc.nodeAt(after);

          if (nodeAfter) {
            return editor.commands.command(({ tr }) => {
              tr.setSelection(Selection.near(doc.resolve(after)));
              return true;
            });
          }

          return editor.commands.exitCode();
        } catch (error) {
          console.error("Error handling ArrowDown in CustomCodeBlockExtension:", error);
          return false;
        }
      },
    };
  },
}).configure({
  lowlight,
  defaultLanguage: "plaintext",
  exitOnTripleEnter: false,
});

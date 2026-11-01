/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor, KeyboardShortcutCommand } from "@tiptap/core";
import { Extension } from "@tiptap/core";
import { useState } from "react";

/**
 * Creates a title editor extension that enables keyboard navigation to the main editor
 *
 * @param getMainEditor Function to get the main editor instance
 * @returns A Tiptap extension with keyboard shortcuts
 */
const createTitleNavigationExtension = (getMainEditor: () => Editor | null) =>
  Extension.create({
    name: "titleEditorNavigation",
    priority: 10,

    addKeyboardShortcuts() {
      return {
        // Arrow down at end of title - Move to main editor
        ArrowDown: () => {
          const mainEditor = getMainEditor();
          if (!mainEditor) return false;

          // If cursor is at the end of the title
          mainEditor.commands.focus("start");
          return true;
        },

        // Right arrow at end of title - Move to main editor
        ArrowRight: ({ editor: titleEditor }) => {
          const mainEditor = getMainEditor();
          if (!mainEditor) return false;

          const { from, to } = titleEditor.state.selection;
          const documentLength = titleEditor.state.doc.content.size;

          // If cursor is at the end of the title
          if (from === to && to === documentLength - 1) {
            mainEditor.commands.focus("start");
            return true;
          }
          return false;
        },

        // Enter - Create new line in main editor and focus
        Enter: () => {
          const mainEditor = getMainEditor();
          if (!mainEditor) return false;

          // Focus at the start of the main editor
          mainEditor.chain().focus().insertContentAt(0, { type: "paragraph" }).run();
          return true;
        },
      };
    },
  });

/**
 * Creates a main editor extension that enables keyboard navigation to the title editor
 *
 * @param getTitleEditor Function to get the title editor instance
 * @returns A Tiptap extension with keyboard shortcuts
 */
const createMainNavigationExtension = (getTitleEditor: () => Editor | null) => {
  const focusTitleAtStart: KeyboardShortcutCommand = ({ editor }) => {
    const titleEditor = getTitleEditor();
    if (!titleEditor) return false;
    const { from, to } = editor.state.selection;
    if (from !== 1 || to !== 1) return false;
    titleEditor.commands.focus("end");
    return true;
  };
  return Extension.create({
    name: "mainEditorNavigation",
    priority: 10,

    addKeyboardShortcuts() {
      return {
        ArrowUp: focusTitleAtStart,
        ArrowLeft: focusTitleAtStart,

        // Backspace - Special handling for first paragraph
        Backspace: ({ editor }) => {
          const titleEditor = getTitleEditor();
          if (!titleEditor) return false;

          const { from, to, empty } = editor.state.selection;

          // Only handle when cursor is at position 1 with empty selection
          if (from === 1 && to === 1 && empty) {
            const firstNode = editor.state.doc.firstChild;

            // If first node is a paragraph
            if (firstNode && firstNode.type.name === "paragraph") {
              // If paragraph is already empty, delete it and focus title editor
              if (firstNode.content.size === 0) {
                editor.commands.deleteNode("paragraph");
                // Use setTimeout to ensure the node is deleted before changing focus
                setTimeout(() => titleEditor.commands.focus("end"), 0);
                return true;
              }
              // If paragraph is not empty, just move focus to title editor
              else {
                titleEditor.commands.focus("end");
                return true;
              }
            }
          }
          return false;
        },
      };
    },
  });
};

/** Own editor handles and extensions for the lifetime of one editor pair. */
function createEditorNavigation() {
  let titleEditor: Editor | null = null;
  let mainEditor: Editor | null = null;
  return {
    setTitleEditor: (editor: Editor | null) => {
      titleEditor = editor;
    },
    setMainEditor: (editor: Editor | null) => {
      mainEditor = editor;
    },
    titleNavigationExtension: createTitleNavigationExtension(() => mainEditor),
    mainNavigationExtension: createMainNavigationExtension(() => titleEditor),
  };
}

export function useEditorNavigation() {
  const [navigation] = useState(createEditorNavigation);
  return navigation;
}

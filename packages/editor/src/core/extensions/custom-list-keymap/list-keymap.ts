/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Extension } from "@tiptap/core";
import type { Editor } from "@tiptap/core";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";
// extensions
import { handleBackspace, handleDelete } from "@/extensions/custom-list-keymap/list-helpers";

export type ListKeymapOptions = {
  listTypes: Array<{
    itemName: string;
    wrapperNames: string[];
  }>;
};

export const ListKeymap = ({ tabIndex }: { tabIndex?: number }) =>
  Extension.create<ListKeymapOptions>({
    name: "listKeymap",

    addOptions() {
      return {
        listTypes: [
          {
            itemName: "listItem",
            wrapperNames: ["bulletList", "orderedList"],
          },
          {
            itemName: "taskItem",
            wrapperNames: ["taskList"],
          },
        ],
      };
    },

    addKeyboardShortcuts() {
      const handleItems = (editor: Editor, operation: "Delete" | "Backspace") => {
        let handled = false;
        for (const { itemName, wrapperNames } of this.options.listTypes) {
          if (editor.state.schema.nodes[itemName] === undefined) continue;
          const changed =
            operation === "Delete" ? handleDelete(editor, itemName) : handleBackspace(editor, itemName, wrapperNames);
          handled = changed || handled;
        }
        return handled;
      };
      const guarded = (editor: Editor, operation: "Delete" | "Backspace") => {
        try {
          return handleItems(editor, operation);
        } catch (error) {
          console.error(`Error in handling ${operation}:`, error);
          return false;
        }
      };
      return {
        Tab: () => {
          if (this.editor.isActive(CORE_EXTENSIONS.LIST_ITEM) || this.editor.isActive(CORE_EXTENSIONS.TASK_ITEM)) {
            if (this.editor.commands.sinkListItem(CORE_EXTENSIONS.LIST_ITEM)) {
              return true;
            } else if (this.editor.commands.sinkListItem(CORE_EXTENSIONS.TASK_ITEM)) {
              return true;
            }
            return true;
          }
          // if tabIndex is set, we don't want to handle Tab key
          if (tabIndex !== undefined && tabIndex !== null) {
            return false;
          }
          return true;
        },
        "Shift-Tab": () => {
          if (this.editor.commands.liftListItem(CORE_EXTENSIONS.LIST_ITEM)) {
            return true;
          } else if (this.editor.commands.liftListItem(CORE_EXTENSIONS.TASK_ITEM)) {
            return true;
          }
          // if tabIndex is set, we don't want to handle Tab key
          if (tabIndex !== undefined && tabIndex !== null) {
            return false;
          }
          return true;
        },
        Delete: ({ editor }) => guarded(editor, "Delete"),
        "Mod-Delete": ({ editor }) => handleItems(editor, "Delete"),
        Backspace: ({ editor }) => guarded(editor, "Backspace"),
        "Mod-Backspace": ({ editor }) => handleItems(editor, "Backspace"),
      };
    },
  });

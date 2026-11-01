/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor, KeyboardShortcutCommand } from "@tiptap/core";
import { CORE_EXTENSIONS } from "@/constants/extension";
import { findParentNodeOfType } from "@/helpers/common";

function leaveTable(editor: Editor, edge: "before" | "after") {
  if (!editor.isEditable || !editor.isActive(CORE_EXTENSIONS.TABLE)) return false;
  try {
    const { selection, doc } = editor.state;
    const table = findParentNodeOfType(selection, [CORE_EXTENSIONS.TABLE]);
    if (!table) return false;
    const selectedRow = findParentNodeOfType(selection, [CORE_EXTENSIONS.TABLE_ROW]);
    const boundaryRow = edge === "before" ? table.node.firstChild : table.node.lastChild;
    if (selectedRow?.node !== boundaryRow) return false;
    const position = edge === "before" ? table.pos : table.pos + table.node.nodeSize;
    const boundary = doc.resolve(position);
    const adjacent = edge === "before" ? boundary.nodeBefore : boundary.nodeAfter;
    if (adjacent) {
      if (adjacent.type.name !== CORE_EXTENSIONS.PARAGRAPH) return false;
      const cursor = edge === "before" ? position - 1 : position + adjacent.nodeSize - 1;
      editor.commands.setTextSelection(cursor);
    } else {
      editor.commands.insertContentAt(position, { type: CORE_EXTENSIONS.PARAGRAPH });
      editor.commands.setTextSelection(position + 1);
    }
    return true;
  } catch (error) {
    console.error("Failed to move the cursor outside the table", error);
    return false;
  }
}

export const insertLineAboveTableAction: KeyboardShortcutCommand = ({ editor }) => leaveTable(editor, "before");
export const insertLineBelowTableAction: KeyboardShortcutCommand = ({ editor }) => leaveTable(editor, "after");

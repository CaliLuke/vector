/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { ECustomImageStatus } from "@/extensions/custom-image/types";
import { CORE_EDITOR_META } from "@/constants/meta";
import type { TFileHandler } from "@/types";
import { addFileSource, collectFileSources, readFileNode } from "./nodes";
import type { FileNode } from "./nodes";

const DELETE_PLUGIN_KEY = new PluginKey("delete-utility");

async function deleteFile(
  editor: Editor,
  file: FileNode,
  deleteHandler: TFileHandler["delete"],
  deleteSource: boolean
) {
  try {
    editor.storage[file.name]?.[file.fileSetName]?.set(file.src, true);
    if (file.id) editor.commands.updateAssetsList?.({ idToRemove: file.id });
    if (deleteSource) await deleteHandler(file.src);
  } catch (error) {
    console.error("Error deleting file via delete utility plugin:", error);
  }
}

export const TrackFileDeletionPlugin = (editor: Editor, deleteHandler: TFileHandler["delete"]): Plugin =>
  new Plugin({
    key: DELETE_PLUGIN_KEY,
    appendTransaction: (transactions, oldState, newState) => {
      if (!transactions.some((tr) => tr.docChanged)) return null;
      if (transactions.some((tr) => tr.getMeta(CORE_EDITOR_META.SKIP_FILE_DELETION))) return null;
      const newFileSources = collectFileSources(newState.doc);
      const deleting = new Map<string, Set<string>>();
      oldState.doc.descendants((node) => {
        const file = readFileNode(node);
        if (!file) return;
        // A provisional copy still belongs to its source until duplication succeeds.
        if (
          node.attrs.status === ECustomImageStatus.DUPLICATING ||
          node.attrs.status === ECustomImageStatus.DUPLICATION_FAILED
        )
          return;
        if (!newFileSources.get(file.name)?.has(file.src))
          void deleteFile(editor, file, deleteHandler, addFileSource(deleting, file));
      });
      return null;
    },
  });

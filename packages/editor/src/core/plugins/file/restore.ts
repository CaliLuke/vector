/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { CORE_EXTENSIONS } from "@plane/utils/editor/common";
import { CORE_ASSETS_META_DATA_RECORD } from "@/helpers/assets";
import type { TFileHandler } from "@/types";
import { addFileSource, collectFileSources, readFileNode } from "./nodes";
import type { FileNode } from "./nodes";

const RESTORE_PLUGIN_KEY = new PluginKey("restore-utility");

async function restoreFile(editor: Editor, file: FileNode, restoreHandler: TFileHandler["restore"]) {
  const deletedFiles = editor.storage[file.name]?.[file.fileSetName];
  const wasDeleted = deletedFiles?.get(file.src);
  if (wasDeleted === undefined) {
    deletedFiles?.set(file.src, false);
  } else if (wasDeleted) {
    try {
      await restoreHandler(file.src);
      deletedFiles?.set(file.src, false);
    } catch (error) {
      console.error("Error restoring file via restore utility plugin:", error);
    }
  }
}

export const TrackFileRestorationPlugin = (editor: Editor, restoreHandler: TFileHandler["restore"]): Plugin =>
  new Plugin({
    key: RESTORE_PLUGIN_KEY,
    appendTransaction: (transactions, oldState, newState) => {
      if (!transactions.some((tr) => tr.docChanged)) return null;
      const oldFileSources = collectFileSources(oldState.doc);
      const restoring = new Map<string, Set<string>>();
      newState.doc.descendants((node) => {
        const file = readFileNode(node);
        if (!file || oldFileSources.get(file.name)?.has(file.src)) return;
        const asset = CORE_ASSETS_META_DATA_RECORD[file.name]?.(node.attrs);
        if (asset) editor.commands.updateAssetsList?.({ asset });
        // Private images restore on their failed load, rather than document insertion.
        if (file.name === CORE_EXTENSIONS.CUSTOM_IMAGE && !file.src.startsWith("http")) return;
        if (addFileSource(restoring, file)) void restoreFile(editor, file, restoreHandler);
      });
      return null;
    },
  });

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";
// types
import type { TFileHandler } from "@/types";

/**
 * Finds all public image nodes in the document and restores them using the provided restore function
 *
 * Never remove this onCreate hook, it's a hack to restore old public
 * images, since they don't give error if they've been deleted as they are
 * rendered directly from image source instead of going through the
 * apiserver
 */
export const restorePublicImages = (editor: Editor, restoreImageFn: TFileHandler["restore"]) => {
  const imageSources = new Set<string>();
  editor.state.doc.descendants((node) => {
    if ([CORE_EXTENSIONS.IMAGE, CORE_EXTENSIONS.CUSTOM_IMAGE].some((type) => type === node.type.name)) {
      if (typeof node.attrs.src !== "string" || !node.attrs.src.startsWith("http")) return;

      imageSources.add(node.attrs.src);
    }
  });

  for (const src of imageSources) void restorePublicImage(src, restoreImageFn);
};

async function restorePublicImage(src: string, restoreImageFn: TFileHandler["restore"]) {
  try {
    await restoreImageFn(src);
  } catch (error) {
    console.error("Error restoring image: ", error);
  }
}

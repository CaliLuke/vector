/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ImageOptions } from "@tiptap/extension-image";
import { mergeAttributes } from "@tiptap/core";
import { Image as BaseImageExtension } from "@tiptap/extension-image";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";
// local imports
import type { CustomImageExtensionType, CustomImageExtensionStorage, InsertImageComponentProps } from "./types";
import { DEFAULT_CUSTOM_IMAGE_ATTRIBUTES } from "./utils";

declare module "@tiptap/core" {
  interface Commands<ReturnType> {
    [CORE_EXTENSIONS.CUSTOM_IMAGE]: {
      insertImageComponent: ({ file, pos, event }: InsertImageComponentProps) => ReturnType;
    };
  }
  interface Storage {
    [CORE_EXTENSIONS.CUSTOM_IMAGE]: CustomImageExtensionStorage;
  }
}

export const CustomImageExtensionConfig: CustomImageExtensionType = BaseImageExtension.extend<
  ImageOptions,
  CustomImageExtensionStorage
>({
  name: CORE_EXTENSIONS.CUSTOM_IMAGE,
  group: "block",
  atom: true,

  addAttributes() {
    const attributes = {
      ...this.parent?.(),
      ...Object.fromEntries(
        Object.entries(DEFAULT_CUSTOM_IMAGE_ATTRIBUTES).map(([name, value]) => [name, { default: value }])
      ),
    };

    return attributes;
  },

  parseHTML() {
    return [
      {
        tag: "image-component",
      },
    ];
  },

  renderHTML({ HTMLAttributes }) {
    return ["image-component", mergeAttributes(HTMLAttributes)];
  },
});

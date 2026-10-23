/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ReactNodeViewRenderer } from "@tiptap/react";
import { CodeBlockComponent } from "./code-block-node-view";
import { CustomCodeBlockExtensionWithoutProps } from "./without-props";

export const CustomCodeBlockExtension = CustomCodeBlockExtensionWithoutProps.extend({
  addNodeView() {
    return ReactNodeViewRenderer(CodeBlockComponent);
  },
}).configure({
  HTMLAttributes: {
    class: "",
  },
});

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { TrailingNode as TiptapTrailingNode } from "@tiptap/extensions";
import { CORE_EXTENSIONS } from "@/constants/extension";

// Preserve the public paragraph defaults while sharing the supported plugin.
export const TrailingNode = TiptapTrailingNode.configure({
  node: CORE_EXTENSIONS.PARAGRAPH,
  notAfter: [CORE_EXTENSIONS.PARAGRAPH],
});

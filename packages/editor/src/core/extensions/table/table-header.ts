/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { mergeAttributes, Node } from "@tiptap/core";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";
// local imports
import { cellGeometryAttributes } from "./cell-attributes";

type TableHeaderOptions = {
  HTMLAttributes: Record<string, unknown>;
};

export const TableHeader = Node.create<TableHeaderOptions>({
  name: CORE_EXTENSIONS.TABLE_HEADER,

  addOptions() {
    return {
      HTMLAttributes: {},
    };
  },

  content: "block+",

  addAttributes() {
    return {
      ...cellGeometryAttributes(),
      background: {
        default: "none",
      },
    };
  },

  tableRole: "header_cell",

  isolating: true,

  parseHTML() {
    return [{ tag: "th" }];
  },

  renderHTML({ node, HTMLAttributes }) {
    return [
      "th",
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes, {
        style: `background-color: ${node.attrs.background};`,
      }),
      0,
    ];
  },
});

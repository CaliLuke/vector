/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { EditorView } from "@tiptap/pm/view";

const generalSelectors = [
  "li",
  "p.editor-paragraph-block:not(:first-child)",
  ".code-block",
  "blockquote",
  "h1.editor-heading-block, h2.editor-heading-block, h3.editor-heading-block, h4.editor-heading-block, h5.editor-heading-block, h6.editor-heading-block",
  "[data-type=horizontalRule]",
  "table:not(.table-drag-preview)",
  ".issue-embed",
  ".image-component",
  ".image-upload-component",
  ".editor-callout-component",
  ".editor-embed-component",
  ".editor-drawio-component",
].join(", ");

export const nodeDOMAtCoords = (coords: { x: number; y: number }) => {
  const elements = document.elementsFromPoint(coords.x, coords.y);

  for (const elem of elements) {
    // Check for table wrapper first
    if (elem.matches("table:not(.table-drag-preview)")) {
      return elem;
    }

    if (elem.matches("p:first-child") && elem.parentElement?.matches(".ProseMirror")) {
      return elem;
    }

    // Skip table cells
    if (elem.closest("table")) {
      continue;
    }

    // Skip elements inside .editor-embed-component
    if (elem.closest(".editor-embed-component") && !elem.matches(".editor-embed-component")) {
      continue;
    }

    // apply general selector
    if (elem.matches(generalSelectors)) {
      return elem;
    }
  }
  return null;
};

/** Resolve a block hit from an inset inside its rendered bounds. */
export function nodePosAtDOM(node: Element, view: EditorView, leftInset: number) {
  const bounds = node.getBoundingClientRect();
  return view.posAtCoords({ left: bounds.left + leftInset, top: bounds.top + 1 })?.inside;
}

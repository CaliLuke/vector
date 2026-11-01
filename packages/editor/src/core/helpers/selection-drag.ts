/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";

export function observeSelectionDrag(
  editor: Editor,
  getMenu: () => HTMLElement | null,
  onSelecting: (value: boolean) => void
) {
  function handleMouseMove() {
    if (!editor.state.selection.empty) {
      onSelecting(true);
      document.removeEventListener("mousemove", handleMouseMove);
    }
  }
  function removeDragListeners() {
    document.removeEventListener("mousemove", handleMouseMove);
    document.removeEventListener("mouseup", handleMouseUp);
  }
  function handleMouseUp() {
    onSelecting(false);
    removeDragListeners();
  }
  function handleMouseDown(event: MouseEvent) {
    if (event.target instanceof Node && getMenu()?.contains(event.target)) return;
    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
  }
  document.addEventListener("mousedown", handleMouseDown);
  return () => {
    document.removeEventListener("mousedown", handleMouseDown);
    removeDragListeners();
  };
}

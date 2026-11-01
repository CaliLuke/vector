/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import { findTable, isCellSelection } from "@/extensions/table/table/utilities/helpers";
import type { moveSelectedRows } from "./actions";
import { hideDragMarker, hideDropMarker } from "./marker-utils";
import { showCellContent } from "./utils";

export function finishTableDrag(
  editor: Editor,
  initialIndex: number,
  dropIndex: number,
  move: typeof moveSelectedRows
) {
  if (initialIndex === dropIndex || editor.isDestroyed) return;
  const selection = editor.state.selection;
  if (!isCellSelection(selection)) return;
  const table = findTable(selection);
  if (table) editor.view.dispatch(move(editor, table, selection, dropIndex, editor.state.tr));
}

/** Own one drag's listeners and decorations, including cancellation and failing callbacks. */
export function startTableDrag({
  editor,
  dropMarker,
  dragMarker,
  onMove,
  onDrop,
}: {
  editor: Editor;
  dropMarker: HTMLElement;
  dragMarker: HTMLElement;
  onMove: (event: MouseEvent) => void;
  onDrop: () => void;
}): () => void {
  let active = true;
  const cancel = () => {
    if (!active) return;
    active = false;
    window.removeEventListener("mouseup", finish);
    window.removeEventListener("mousemove", move);
    window.removeEventListener("blur", cancel);
    hideDropMarker(dropMarker);
    hideDragMarker(dragMarker);
    if (!editor.isDestroyed) showCellContent(editor);
  };
  const finish = () => {
    cancel();
    onDrop();
  };
  const move = (event: MouseEvent) => {
    try {
      onMove(event);
    } catch (error) {
      cancel();
      throw error;
    }
  };
  window.addEventListener("mouseup", finish);
  window.addEventListener("mousemove", move);
  window.addEventListener("blur", cancel);
  return cancel;
}

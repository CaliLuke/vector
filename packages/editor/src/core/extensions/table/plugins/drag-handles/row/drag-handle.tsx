/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import { TableDragHandle } from "../drag-handle";
import { finishTableDrag, startTableDrag } from "../drag-session";
import { findTable, getTableHeightPx, getTableWidthPx, selectRow } from "@/extensions/table/table/utilities/helpers";
import { moveSelectedRows } from "../actions";
import {
  DROP_MARKER_THICKNESS,
  getDropMarker,
  getRowDragMarker,
  updateRowDragMarker,
  updateRowDropMarker,
} from "../marker-utils";
import { RowOptionsDropdown } from "../axis-options";
import { calculateRowDropIndex, constructRowDragPreview, getTableRowNodesInfo } from "./utils";

export type RowDragHandleProps = { editor: Editor; row: number };

function startRowDrag(editor: Editor, row: number, e: MouseEvent) {
  const table = findTable(editor.state.selection);
  if (!table) return undefined;

  editor.view.dispatch(selectRow(table, row, editor.state.tr));

  // drag row
  const tableHeightPx = getTableHeightPx(table, editor);
  const rows = getTableRowNodesInfo(table, editor);

  let dropIndex = row;
  const initial = rows.at(row);
  if (!initial) return undefined;
  const startTop = initial.top;
  const startY = e.clientY;
  const tableElement = editor.view.nodeDOM(table.pos);

  const dropMarker = tableElement instanceof HTMLElement ? getDropMarker(tableElement) : null;
  const dragMarker = tableElement instanceof HTMLElement ? getRowDragMarker(tableElement) : null;
  if (!dropMarker || !dragMarker) return undefined;

  let pseudoRow: HTMLElement | undefined;

  const handleMove = (moveEvent: MouseEvent) => {
    const cursorTop = startTop + moveEvent.clientY - startY;
    dropIndex = calculateRowDropIndex(row, rows, cursorTop);

    if (!pseudoRow) {
      pseudoRow = constructRowDragPreview(editor, editor.state.selection, table);
      const tableWidthPx = getTableWidthPx(table, editor);
      if (pseudoRow) {
        pseudoRow.style.width = `${tableWidthPx}px`;
      }
    }

    const dragMarkerHeightPx = initial.height;
    const dragMarkerTopPx = Math.max(0, Math.min(cursorTop, tableHeightPx - dragMarkerHeightPx));
    const dropMarkerTopPx = dropIndex <= row ? rows[dropIndex].top : rows[dropIndex].top + rows[dropIndex].height;

    updateRowDropMarker({
      element: dropMarker,
      top: dropMarkerTopPx - DROP_MARKER_THICKNESS / 2,
      height: DROP_MARKER_THICKNESS,
    });
    updateRowDragMarker({
      element: dragMarker,
      top: dragMarkerTopPx,
      height: dragMarkerHeightPx,
      pseudoRow,
    });
  };

  return startTableDrag({
    editor,
    dropMarker,
    dragMarker,
    onMove: handleMove,
    onDrop: () => finishTableDrag(editor, row, dropIndex, moveSelectedRows),
  });
}

export function RowDragHandle({ editor, row }: RowDragHandleProps) {
  return (
    <TableDragHandle
      editor={editor}
      axis="row"
      Dropdown={RowOptionsDropdown}
      startDrag={(event) => startRowDrag(editor, row, event)}
    />
  );
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import { TableDragHandle } from "../drag-handle";
import { finishTableDrag, startTableDrag } from "../drag-session";
import { findTable, getTableHeightPx, getTableWidthPx, selectColumn } from "@/extensions/table/table/utilities/helpers";
import { moveSelectedColumns } from "../actions";
import {
  DROP_MARKER_THICKNESS,
  getColDragMarker,
  getDropMarker,
  updateColDragMarker,
  updateColDropMarker,
} from "../marker-utils";
import { ColumnOptionsDropdown } from "../axis-options";
import { calculateColumnDropIndex, constructColumnDragPreview, getTableColumnNodesInfo } from "./utils";

export type ColumnDragHandleProps = { editor: Editor; col: number };

function startColumnDrag(editor: Editor, col: number, e: MouseEvent) {
  const table = findTable(editor.state.selection);
  if (!table) return undefined;

  editor.view.dispatch(selectColumn(table, col, editor.state.tr));

  // drag column
  const tableWidthPx = getTableWidthPx(table, editor);
  const columns = getTableColumnNodesInfo(table, editor);

  let dropIndex = col;
  const initial = columns.at(col);
  if (!initial) return undefined;
  const startLeft = initial.left;
  const startX = e.clientX;
  const tableElement = editor.view.nodeDOM(table.pos);

  const dropMarker = tableElement instanceof HTMLElement ? getDropMarker(tableElement) : null;
  const dragMarker = tableElement instanceof HTMLElement ? getColDragMarker(tableElement) : null;
  if (!dropMarker || !dragMarker) return undefined;

  let pseudoColumn: HTMLElement | undefined;

  const handleMove = (moveEvent: MouseEvent) => {
    const currentLeft = startLeft + moveEvent.clientX - startX;
    dropIndex = calculateColumnDropIndex(col, columns, currentLeft);

    if (!pseudoColumn) {
      pseudoColumn = constructColumnDragPreview(editor, editor.state.selection, table);
      const tableHeightPx = getTableHeightPx(table, editor);
      if (pseudoColumn) {
        pseudoColumn.style.height = `${tableHeightPx}px`;
      }
    }

    const dragMarkerWidthPx = initial.width;
    const dragMarkerLeftPx = Math.max(0, Math.min(currentLeft, tableWidthPx - dragMarkerWidthPx));
    const dropMarkerLeftPx =
      dropIndex <= col ? columns[dropIndex].left : columns[dropIndex].left + columns[dropIndex].width;

    updateColDropMarker({
      element: dropMarker,
      left: dropMarkerLeftPx - Math.floor(DROP_MARKER_THICKNESS / 2) - 1,
      width: DROP_MARKER_THICKNESS,
    });
    updateColDragMarker({
      element: dragMarker,
      left: dragMarkerLeftPx,
      width: dragMarkerWidthPx,
      pseudoColumn,
    });
  };

  return startTableDrag({
    editor,
    dropMarker,
    dragMarker,
    onMove: handleMove,
    onDrop: () => finishTableDrag(editor, col, dropIndex, moveSelectedColumns),
  });
}

export function ColumnDragHandle({ editor, col }: ColumnDragHandleProps) {
  return (
    <TableDragHandle
      editor={editor}
      axis="column"
      Dropdown={ColumnOptionsDropdown}
      startDrag={(event) => startColumnDrag(editor, col, event)}
    />
  );
}

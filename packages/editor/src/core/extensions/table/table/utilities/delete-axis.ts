/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Command } from "@tiptap/core";
import { deleteColumn, deleteRow, deleteTable, TableMap } from "@tiptap/pm/tables";
import { findTable, isCellSelection } from "./helpers";

function deleteAxis(axis: "row" | "column"): Command {
  return ({ state, dispatch }) => {
    if (!isCellSelection(state.selection)) return false;
    const table = findTable(state.selection);
    if (!table?.node.firstChild) return false;
    const map = TableMap.get(table.node);
    const size = axis === "column" ? map.width : map.height;
    if (size === 1) return deleteTable(state, dispatch);
    return (axis === "column" ? deleteColumn : deleteRow)(state, dispatch);
  };
}
export const deleteColumnOrTable = (): Command => deleteAxis("column");
export const deleteRowOrTable = (): Command => deleteAxis("row");

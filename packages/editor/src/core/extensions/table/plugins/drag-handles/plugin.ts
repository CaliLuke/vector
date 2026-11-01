/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { TableMap } from "@tiptap/pm/tables";
import { Decoration, DecorationSet } from "@tiptap/pm/view";
import { ReactRenderer } from "@tiptap/react";
import { findTable, getTableCellWidgetDecorationPos } from "@/extensions/table/table/utilities/helpers";
import { ColumnDragHandle } from "./column/drag-handle";
import { RowDragHandle } from "./row/drag-handle";

type HandleState = { decorations: DecorationSet; count: number; tablePos: number };
const EMPTY_STATE: HandleState = { decorations: DecorationSet.empty, count: 0, tablePos: -1 };
const KEYS = {
  row: new PluginKey<HandleState>("tableRowDragHandlePlugin"),
  column: new PluginKey<HandleState>("tableColumnHandlerDecorationPlugin"),
};

function createHandleWidget(editor: Editor, axis: keyof typeof KEYS, index: number, pos: number) {
  const renderers = new WeakMap<Node, ReactRenderer>();
  return Decoration.widget(
    pos,
    () => {
      const renderer =
        axis === "row"
          ? new ReactRenderer(RowDragHandle, { editor, props: { editor, row: index } })
          : new ReactRenderer(ColumnDragHandle, { editor, props: { editor, col: index } });
      renderers.set(renderer.element, renderer);
      return renderer.element;
    },
    {
      destroy(node) {
        renderers.get(node)?.destroy();
        renderers.delete(node);
      },
    }
  );
}

export function createTableDragHandlePlugin(editor: Editor, axis: keyof typeof KEYS): Plugin<HandleState> {
  const key = KEYS[axis];
  return new Plugin<HandleState>({
    key,
    state: {
      init: () => EMPTY_STATE,
      apply(tr, previous, oldState, newState) {
        const table = findTable(newState.selection);
        if (!editor.isEditable || !table) return EMPTY_STATE;
        if (!tr.docChanged && newState.selection.eq(oldState.selection) && previous.tablePos === table.pos)
          return previous;

        const map = TableMap.get(table.node);
        const count = axis === "row" ? map.height : map.width;
        const positions = Array.from({ length: count }, (_, index) =>
          getTableCellWidgetDecorationPos(table, map, axis === "row" ? index * map.width : index)
        );
        const mapped = previous.decorations.map(tr.mapping, tr.doc);
        if (
          previous.count === count &&
          previous.tablePos === table.pos &&
          positions.every((pos) => mapped.find(pos, pos + 1).length === 1)
        )
          return { decorations: mapped, count, tablePos: table.pos };

        return {
          decorations: DecorationSet.create(
            newState.doc,
            positions.map((pos, index) => createHandleWidget(editor, axis, index, pos))
          ),
          count,
          tablePos: table.pos,
        };
      },
    },
    props: {
      decorations(state) {
        return key.getState(state)?.decorations;
      },
    },
  });
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import type { MouseEvent } from "react";
import { TableMap } from "@tiptap/pm/tables";
import { ArrowDown, ArrowLeft, ArrowRight, ArrowUp, ToggleRight } from "lucide-react";
import { CopyIcon, TrashIcon, CloseIcon } from "@plane/propel/icons";
import { findTable, getSelectedColumns, getSelectedRows } from "@/extensions/table/table/utilities/helpers";
import { duplicateColumns, duplicateRows } from "./actions";
import { TableDragHandleDropdownColorSelector } from "./color-selector";

type Props = { editor: Editor; onClose: () => void };
type Axis = "row" | "column";

function duplicateAxis(editor: Editor, axis: Axis) {
  const table = findTable(editor.state.selection);
  if (!table) return;
  const map = TableMap.get(table.node);
  const selected = axis === "column" ? getSelectedColumns : getSelectedRows;
  const duplicate = axis === "column" ? duplicateColumns : duplicateRows;
  editor.view.dispatch(duplicate(table, selected(editor.state.selection, map), editor.state.tr));
}

function AxisOptionsDropdown({ editor, onClose, axis }: Props & { axis: Axis }) {
  const column = axis === "column";
  const apply = (event: MouseEvent, action: () => void) => {
    event.preventDefault();
    event.stopPropagation();
    if (editor.isDestroyed || !editor.isEditable) return;
    action();
    onClose();
  };
  const items = [
    {
      key: "before",
      label: column ? "Insert left" : "Insert above",
      icon: column ? ArrowLeft : ArrowUp,
      action: () => {
        const chain = editor.chain().focus();
        (column ? chain.addColumnBefore() : chain.addRowBefore()).run();
      },
    },
    {
      key: "after",
      label: column ? "Insert right" : "Insert below",
      icon: column ? ArrowRight : ArrowDown,
      action: () => {
        const chain = editor.chain().focus();
        (column ? chain.addColumnAfter() : chain.addRowAfter()).run();
      },
    },
    { key: "duplicate", label: "Duplicate", icon: CopyIcon, action: () => duplicateAxis(editor, axis) },
    {
      key: "clear",
      label: "Clear contents",
      icon: CloseIcon,
      action: () => {
        editor.chain().focus().clearSelectedCells().run();
      },
    },
    {
      key: "delete",
      label: "Delete",
      icon: TrashIcon,
      action: () => {
        const chain = editor.chain().focus();
        (column ? chain.deleteColumn() : chain.deleteRow()).run();
      },
    },
  ];
  return (
    <>
      <button
        type="button"
        className="flex w-full items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 text-left text-11 text-secondary hover:bg-layer-1"
        onClick={(event) =>
          apply(event, () => {
            const chain = editor.chain().focus();
            (column ? chain.toggleHeaderColumn() : chain.toggleHeaderRow()).run();
          })
        }
      >
        <div className="flex-grow truncate">Header {axis}</div>
        <ToggleRight className="size-3 shrink-0" />
      </button>
      <hr className="my-2 border-subtle" />
      <TableDragHandleDropdownColorSelector editor={editor} onSelect={onClose} />
      {items.map((item) => (
        <button
          key={item.key}
          type="button"
          className="flex w-full items-center gap-2 truncate rounded-sm px-1 py-1.5 text-left text-11 text-secondary hover:bg-layer-1"
          onClick={(event) => apply(event, item.action)}
        >
          <item.icon className="size-3 shrink-0" />
          <div className="flex-grow truncate">{item.label}</div>
        </button>
      ))}
    </>
  );
}

export function ColumnOptionsDropdown(props: Props) {
  return <AxisOptionsDropdown {...props} axis="column" />;
}
export function RowOptionsDropdown(props: Props) {
  return <AxisOptionsDropdown {...props} axis="row" />;
}

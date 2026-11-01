/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Editor } from "@tiptap/core";
import type { Node as ProseMirrorNode } from "@tiptap/pm/model";
import { addColumn, removeColumn, addRow, removeRow, TableMap } from "@tiptap/pm/tables";
import type { TableRect } from "@tiptap/pm/tables";
import { isCellEmpty } from "../../table/utilities/helpers";

const addSvg = `<svg width="16" height="16" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
<path
  d="M8.5 7.49988V3.49988C8.5 3.22374 8.27614 2.99988 8 2.99988C7.72386 2.99988 7.5 3.22374 7.5 3.49988L7.5 7.49988L3.5 7.49988C3.22386 7.49988 3 7.72374 3 7.99988C3 8.27602 3.22386 8.49988 3.5 8.49988H7.5L7.5 12.4999C7.5 12.776 7.72386 12.9999 8 12.9999C8.27614 12.9999 8.5 12.776 8.5 12.4999L8.5 8.49988L12.5 8.49988C12.7761 8.49988 13 8.27602 13 7.99988C13 7.72374 12.7761 7.49988 12.5 7.49988L8.5 7.49988Z"
  fill="currentColor"
/>
</svg>`;

type InsertControl = { element: HTMLButtonElement; destroy: () => void };
export type TableInfo = {
  tableElement: HTMLElement;
  tableNode: ProseMirrorNode;
  tablePos: number;
  columnButton?: InsertControl;
  rowButton?: InsertControl;
  dragMarkerContainerElement?: HTMLElement;
};
type Axis = "row" | "column";
const preventDefault = (event: Event) => event.preventDefault();

export function createTableInsertButton(editor: Editor, tableInfo: TableInfo, axis: Axis): InsertControl {
  const button = document.createElement("button");
  button.type = "button";
  button.className = `table-${axis}-insert-button`;
  button.title = `Insert ${axis}s`;
  button.ariaLabel = button.title;
  const icon = document.createElement("span");
  icon.innerHTML = addSvg;
  button.appendChild(icon);

  const coordinate = axis === "column" ? "clientX" : "clientY";
  const actionThreshold = axis === "column" ? 150 : 40;
  let gesture: { origin: number; lastAction: number; dragging: boolean; userSelect: string } | null = null;
  let suppressClick = false;
  const change = (remove: boolean) => {
    if (editor.isDestroyed || !editor.isEditable || !button.isConnected) return false;
    return changeTableEnd(editor, tableInfo.tableElement, axis, remove);
  };
  const cancel = () => {
    document.removeEventListener("mousemove", onMouseMove);
    document.removeEventListener("mouseup", onMouseUp);
    window.removeEventListener("blur", cancel);
    button.classList.remove("dragging");
    if (gesture?.dragging) document.body.style.userSelect = gesture.userSelect;
    gesture = null;
  };
  const onMouseMove = (event: MouseEvent) => {
    const current = gesture;
    if (!current) return;
    const position = event[coordinate];
    if (!current.dragging && Math.abs(position - current.origin) > 5) {
      current.dragging = true;
      button.classList.add("dragging");
      document.body.style.userSelect = "none";
    }
    if (!current.dragging || Math.abs(position - current.lastAction) < actionThreshold) return;
    if (change(position < current.lastAction)) current.lastAction = position;
  };
  const onMouseUp = () => {
    suppressClick = gesture?.dragging ?? false;
    cancel();
  };
  const onMouseDown = (event: MouseEvent) => {
    if (event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();
    cancel();
    suppressClick = false;
    gesture = {
      origin: event[coordinate],
      lastAction: event[coordinate],
      dragging: false,
      userSelect: document.body.style.userSelect,
    };
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    window.addEventListener("blur", cancel);
  };
  const onClick = (event: MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    const wasDrag = suppressClick && event.detail > 0;
    suppressClick = false;
    if (!wasDrag) change(false);
  };
  button.addEventListener("mousedown", onMouseDown);
  button.addEventListener("click", onClick);
  button.addEventListener("contextmenu", preventDefault);
  button.addEventListener("selectstart", preventDefault);
  return {
    element: button,
    destroy: () => {
      cancel();
      button.removeEventListener("mousedown", onMouseDown);
      button.removeEventListener("click", onClick);
      button.removeEventListener("contextmenu", preventDefault);
      button.removeEventListener("selectstart", preventDefault);
      button.remove();
    },
  };
}

export function findAllTables(editor: Editor): TableInfo[] {
  const tables: TableInfo[] = [];
  editor.state.doc.descendants((node, pos) => {
    if (node.type.spec.tableRole !== "table") return;
    const dom = editor.view.nodeDOM(pos);
    if (!(dom instanceof HTMLElement)) return;
    const tableElement = dom.matches("table") ? dom : dom.querySelector("table");
    if (tableElement) tables.push({ tableElement, tableNode: node, tablePos: pos });
  });
  return tables;
}

function changeTableEnd(editor: Editor, element: HTMLElement, axis: Axis, remove: boolean): boolean {
  const current = findAllTables(editor).find((table) => table.tableElement === element);
  if (!current) return false;
  const { tableNode, tablePos } = current;
  const map = TableMap.get(tableNode);
  const size = axis === "column" ? map.width : map.height;
  const rect: TableRect = {
    map,
    tableStart: tablePos + 1,
    table: tableNode,
    top: 0,
    left: 0,
    bottom: map.height,
    right: map.width,
  };
  const tr = editor.state.tr;
  if (remove) {
    if (size <= 1 || !isLastAxisEmpty(tableNode, map, axis)) return false;
    const removeAxis = axis === "column" ? removeColumn : removeRow;
    removeAxis(tr, rect, size - 1);
  } else {
    const addAxis = axis === "column" ? addColumn : addRow;
    addAxis(tr, rect, size);
  }
  editor.view.dispatch(tr);
  return true;
}

function isLastAxisEmpty(table: ProseMirrorNode, map: TableMap, axis: Axis) {
  const positions =
    axis === "column"
      ? Array.from({ length: map.height }, (_, row) => map.map[row * map.width + map.width - 1])
      : map.map.slice((map.height - 1) * map.width);
  return positions.every((position) => isCellEmpty(table.nodeAt(position)));
}

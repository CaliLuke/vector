/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useCallback, useRef } from "react";
import type { FocusEvent, KeyboardEvent } from "react";

type GridFocus = { active: HTMLTableCellElement | null; tabs: Map<HTMLElement, string | null> };
const controlSelector = 'button, a[href], input, select, textarea, [contenteditable="true"], [tabindex]';

function gridRows(table: HTMLTableElement) {
  return Array.from(table.rows).filter((row) => row.parentElement !== table.tFoot);
}

function cellControls(cell: HTMLTableCellElement) {
  return Array.from(cell.querySelectorAll<HTMLElement>(controlSelector)).filter(
    (element) => !element.matches(":disabled") && !element.closest("[hidden], [inert]")
  );
}

function syncFocus(table: HTMLTableElement, state: GridFocus, editing: HTMLTableCellElement | null = null) {
  const cells = gridRows(table).flatMap((row) => Array.from(row.cells));
  if (!state.active || !cells.includes(state.active)) state.active = cells[0] ?? null;
  for (const cell of cells) {
    cell.tabIndex = cell === state.active && !editing ? 0 : -1;
    for (const control of cellControls(cell)) {
      if (!state.tabs.has(control)) state.tabs.set(control, control.getAttribute("tabindex"));
      const original = state.tabs.get(control);
      control.tabIndex = cell === editing ? Number(original ?? 0) : -1;
    }
  }
  for (const control of state.tabs.keys()) {
    if (!table.contains(control)) state.tabs.delete(control);
  }
}

function eventCell(table: HTMLTableElement, target: EventTarget | null) {
  if (!(target instanceof Element)) return null;
  const cell = target.closest("td, th");
  return cell instanceof HTMLTableCellElement && cell.closest("table") === table ? cell : null;
}

function adjacentCell(cell: HTMLTableCellElement, event: KeyboardEvent<HTMLTableElement>) {
  const rows = gridRows(event.currentTarget);
  const row = cell.closest("tr");
  if (!row) return null;
  const index = rows.indexOf(row);
  switch (event.key) {
    case "ArrowRight":
      return row.cells[cell.cellIndex + 1];
    case "ArrowLeft":
      return row.cells[cell.cellIndex - 1];
    case "ArrowUp":
      return rows[index - 1]?.cells[cell.cellIndex];
    case "ArrowDown":
      return rows[index + 1]?.cells[cell.cellIndex];
    case "Home":
      return (event.ctrlKey ? rows[0] : row)?.cells[0];
    case "End": {
      const last = event.ctrlKey ? rows.at(-1) : row;
      return last?.cells[last.cells.length - 1];
    }
    default:
      return null;
  }
}

function enterCell(cell: HTMLTableCellElement, key: string) {
  const control = cell.querySelector<HTMLElement>(".clickable") ?? cellControls(cell)[0];
  if (!control) return;
  control.focus();
  if (key !== "F2") control.click();
}

function handleGridKey(event: KeyboardEvent<HTMLTableElement>) {
  if (event.defaultPrevented) return;
  const cell = eventCell(event.currentTarget, event.target);
  if (!cell) return;
  if (event.target !== cell) {
    if (event.key === "Escape") {
      event.preventDefault();
      cell.focus();
    }
    return;
  }
  if (["Enter", " ", "Space", "F2"].includes(event.key)) {
    event.preventDefault();
    enterCell(cell, event.key);
    return;
  }
  if (!["ArrowRight", "ArrowLeft", "ArrowUp", "ArrowDown", "Home", "End"].includes(event.key)) return;
  event.preventDefault();
  const next = adjacentCell(cell, event);
  if (!next) return;
  next.focus();
  next.scrollIntoView({ behavior: "smooth", block: "center", inline: "end" });
}

export function useTableKeyboardNavigation() {
  const focus = useRef<GridFocus>({ active: null, tabs: new Map() });
  const ref = useCallback((table: HTMLTableElement | null) => {
    if (!table) return undefined;
    const state = focus.current;
    const refresh = () => {
      const cell = eventCell(table, document.activeElement);
      syncFocus(table, state, document.activeElement === cell ? null : cell);
    };
    refresh();
    const observer = new MutationObserver(refresh);
    observer.observe(table, { childList: true, subtree: true });
    return () => {
      observer.disconnect();
      for (const [control, original] of state.tabs) {
        if (original === null) control.removeAttribute("tabindex");
        else control.setAttribute("tabindex", original);
      }
      state.tabs.clear();
    };
  }, []);
  const onFocusCapture = (event: FocusEvent<HTMLTableElement>) => {
    const cell = eventCell(event.currentTarget, event.target);
    if (!cell) return;
    focus.current.active = cell;
    syncFocus(event.currentTarget, focus.current, event.target instanceof HTMLTableCellElement ? null : cell);
  };
  const onBlurCapture = (event: FocusEvent<HTMLTableElement>) => {
    if (!event.currentTarget.contains(event.relatedTarget)) syncFocus(event.currentTarget, focus.current);
  };
  return { ref, role: "grid" as const, onKeyDown: handleGridKey, onFocusCapture, onBlurCapture };
}

/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

function verticalCell(element: HTMLTableCellElement, offset: number) {
  const table = element.closest("table");
  const row = element.closest("tr");
  if (!table || !row) return null;
  const rowIndex = Array.from(table.rows).indexOf(row);
  return table.rows[rowIndex + offset]?.cells[element.cellIndex] ?? null;
}

function adjacentCell(element: HTMLTableCellElement, key: string) {
  switch (key) {
    case "ArrowRight":
      return element.nextElementSibling;
    case "ArrowLeft":
      return element.previousElementSibling;
    case "ArrowUp":
      return verticalCell(element, -1);
    case "ArrowDown":
      return verticalCell(element, 1);
    default:
      return null;
  }
}

export const useTableKeyboardNavigation = () => (event: React.KeyboardEvent<HTMLTableElement>) => {
  const element = event.target;
  if (!(element instanceof HTMLTableCellElement)) return;
  if (["Enter", " ", "Space"].includes(event.key)) {
    event.preventDefault();
    element.querySelector<HTMLElement>(".clickable")?.click();
    return;
  }
  const next = adjacentCell(element, event.key);
  if (!(next instanceof HTMLElement)) return;
  event.preventDefault();
  next.focus();
  next.scrollIntoView({ behavior: "smooth", block: "center", inline: "end" });
};

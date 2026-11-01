import { UtilityExtension } from "@/extensions/utility";
import { createEditorHookProps } from "../../../../../../tests/helpers/editor-hook-props";
import { Editor } from "@tiptap/core";
import { TableMap } from "@tiptap/pm/tables";
import { StarterKit } from "@tiptap/starter-kit";
import { EditorContent } from "@tiptap/react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, expect, test, vi } from "vitest";
import { Table } from "../../table/table";
import { TableCell } from "../../table-cell";
import { TableHeader } from "../../table-header";
import { TableRow } from "../../table-row";
import { TableDragStatePlugin } from "../drag-state";
import { ColumnDragHandle } from "./column/drag-handle";
import { RowDragHandle } from "./row/drag-handle";
import { TableDragHandle } from "./drag-handle";
import { TableDragHandleDropdownColorSelector } from "./color-selector";
import { finishTableDrag, startTableDrag } from "./drag-session";
import { moveSelectedRows } from "./actions";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

const cleanups: (() => void)[] = [];
afterEach(() => {
  for (const cleanup of cleanups.toReversed()) cleanup();
  cleanups.length = 0;
  vi.restoreAllMocks();
});

function fixture(withTablePlugins = false) {
  const menuCommands = UtilityExtension({ ...createEditorHookProps(), isEditable: true, isTouchDevice: false });
  const editor = new Editor({
    extensions: [
      StarterKit,
      menuCommands,
      withTablePlugins
        ? Table
        : Table.extend({
            addProseMirrorPlugins: () => [TableDragStatePlugin],
            addNodeView: () => null,
          }),
      TableCell,
      TableHeader,
      TableRow,
    ],
    content:
      "<table><tbody><tr><td><p>A</p></td><td><p>B</p></td></tr><tr><td><p>C</p></td><td><p>D</p></td></tr></tbody></table>",
  });
  const activeMenus = editor.storage.utility.activeDropbarExtensions;
  const editorElement = editor.view.dom;
  const editorHost = document.createElement("div");
  editorHost.append(editorElement);
  document.body.append(editorHost);
  if (!withTablePlugins) editor.commands.setTextSelection(4);
  const table = editor.view.dom.querySelector("table")!;
  const dropMarker = document.createElement("div");
  dropMarker.className = "table-drop-marker hidden";
  const dragMarker = document.createElement("div");
  dragMarker.className = "table-row-drag-marker table-col-drag-marker hidden";
  if (!withTablePlugins) table.append(dropMarker, dragMarker);
  for (const [index, cell] of [...table.querySelectorAll("td")].entries()) {
    Object.defineProperties(cell, {
      offsetWidth: { value: 100 },
      offsetHeight: { value: 40 },
      offsetLeft: { value: (index % 2) * 100 },
    });
  }
  for (const row of table.querySelectorAll("tr")) Object.defineProperty(row, "offsetHeight", { value: 40 });
  Object.defineProperties(table, { offsetWidth: { value: 200 }, offsetHeight: { value: 80 } });
  const container = document.createElement("div");
  document.body.append(container);
  const root = createRoot(container);
  cleanups.push(() => {
    editor.destroy();
    editorHost.remove();
    container.remove();
  });
  cleanups.push(() => act(() => root.unmount()));
  return { editor, root, container, dropMarker, dragMarker, activeMenus };
}

test("table handles release open menus when selection leaves the table and when its structure changes", async () => {
  const { editor, root, container, activeMenus } = fixture(true);
  await act(async () => root.render(<EditorContent editor={editor} />));
  await act(() => editor.commands.setTextSelection(5));
  expect(container.querySelectorAll('button[aria-label="Row options"]')).toHaveLength(2);
  mouse(container.querySelector('button[aria-label="Row options"]')!, "click");
  expect(activeMenus.length).toBe(1);
  await act(() => editor.commands.insertContentAt(editor.state.doc.content.size, "<p>Outside</p>"));
  await act(() => editor.commands.setTextSelection(editor.state.doc.content.size - 2));
  expect(container.querySelectorAll('button[aria-label="Row options"]')).toHaveLength(0);
  expect(activeMenus.length).toBe(0);
  await act(() => editor.commands.setTextSelection(4));
  mouse(container.querySelector('button[aria-label="Row options"]')!, "click");
  expect(activeMenus.length).toBe(1);
  await act(() => editor.commands.addRowAfter());
  expect(container.querySelectorAll('button[aria-label="Row options"]')).toHaveLength(3);
  expect(activeMenus.length).toBe(0);
});

test("destroying an editor cancels table setup before its deferred callback runs", () => {
  vi.useFakeTimers();
  try {
    const editor = new Editor({
      extensions: [StarterKit, Table, TableCell, TableHeader, TableRow],
      content: "<table><tbody><tr><td><p>Draft</p></td></tr></tbody></table>",
    });
    editor.destroy();
    expect(() => vi.runOnlyPendingTimers()).not.toThrow();
  } finally {
    vi.useRealTimers();
  }
});

function mouse(target: EventTarget, type: string, clientX = 0, clientY = 0) {
  act(() => {
    target.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true, clientX, clientY }));
  });
}

function cellText(editor: Editor) {
  return [...editor.view.dom.querySelectorAll(":scope > table > tbody > tr > td > p")].map((cell) => cell.textContent);
}

test.each(["row", "column"] as const)(
  "%s dragging reorders cells once and restores content and markers on release",
  (axis) => {
    const { editor, root, container, dropMarker, dragMarker } = fixture();
    act(() =>
      root.render(
        axis === "row" ? <RowDragHandle editor={editor} row={0} /> : <ColumnDragHandle editor={editor} col={0} />
      )
    );
    mouse(container.querySelector("button")!, "mousedown");
    mouse(window, "mousemove", 110, 50);
    expect(editor.view.dom.querySelector(".content-hidden")).not.toBeNull();
    expect(dragMarker.classList.contains("hidden")).toBe(false);
    mouse(window, "mouseup");
    expect(cellText(editor)).toEqual(axis === "row" ? ["C", "D", "A", "B"] : ["B", "A", "D", "C"]);
    expect(editor.view.dom.querySelector(".content-hidden")).toBeNull();
    expect(dropMarker.classList.contains("hidden")).toBe(true);
    expect(dragMarker.classList.contains("hidden")).toBe(true);
    const documentAfterDrop = editor.getJSON();
    mouse(window, "mousemove", 200, 200);
    mouse(window, "mouseup");
    expect(editor.getJSON()).toEqual(documentAfterDrop);
  }
);

test.each(["row", "column"] as const)("%s handles tolerate missing markers without attaching a drag", (axis) => {
  const { editor, root, container, dragMarker } = fixture();
  dragMarker.remove();
  act(() =>
    root.render(
      axis === "row" ? <RowDragHandle editor={editor} row={0} /> : <ColumnDragHandle editor={editor} col={0} />
    )
  );
  mouse(container.querySelector("button")!, "mousedown");
  mouse(window, "mousemove", 110, 50);
  mouse(window, "mouseup");
  expect(cellText(editor)).toEqual(["A", "B", "C", "D"]);
  expect(editor.view.dom.querySelector(".content-hidden")).toBeNull();
});

test("leaving the window cancels a drag without moving cells", () => {
  const { editor, root, container, dragMarker } = fixture();
  act(() => root.render(<RowDragHandle editor={editor} row={0} />));
  mouse(container.querySelector("button")!, "mousedown");
  mouse(window, "mousemove", 0, 50);
  act(() => {
    window.dispatchEvent(new Event("blur"));
  });
  mouse(window, "mouseup");
  expect(cellText(editor)).toEqual(["A", "B", "C", "D"]);
  expect(editor.view.dom.querySelector(".content-hidden")).toBeNull();
  expect(dragMarker.classList.contains("hidden")).toBe(true);
});

test("replacing and unmounting a handle cancels its previous drag and restores cell visibility", () => {
  const { editor, root, container } = fixture();
  act(() => root.render(<ColumnDragHandle editor={editor} col={0} />));
  const button = container.querySelector("button")!;
  mouse(button, "mousedown");
  mouse(window, "mousemove", 110);
  mouse(button, "mousedown");
  expect(editor.view.dom.querySelector(".content-hidden")).toBeNull();
  mouse(window, "mousemove", 110);
  act(() => root.render(null));
  mouse(window, "mouseup");
  expect(cellText(editor)).toEqual(["A", "B", "C", "D"]);
  expect(editor.view.dom.querySelector(".content-hidden")).toBeNull();
});

function TestDropdown({ onClose }: { editor: Editor; onClose: () => void }) {
  return <button onClick={onClose}>Close menu</button>;
}
const noDrag = () => undefined;

test("table menus release their active editor state on keyboard close, action close and unmount", async () => {
  const { editor, root, container, activeMenus } = fixture();
  act(() => root.render(<TableDragHandle editor={editor} axis="row" Dropdown={TestDropdown} startDrag={noDrag} />));
  const button = container.querySelector("button")!;
  await act(async () => {
    button.click();
  });
  expect(activeMenus.includes("table")).toBe(true);
  const tab = new KeyboardEvent("keydown", { key: "Tab", bubbles: true, cancelable: true });
  await act(async () => {
    document.dispatchEvent(tab);
  });
  expect(activeMenus.includes("table")).toBe(true);
  expect(tab.defaultPrevented).toBe(false);
  act(() => {
    document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
  });
  expect(activeMenus.length).toBe(0);
  await act(async () => {
    button.click();
  });
  const close = [...document.querySelectorAll("button")].find((candidate) => candidate.textContent === "Close menu")!;
  mouse(close, "click");
  expect(activeMenus.length).toBe(0);
  await act(async () => {
    button.click();
  });
  act(() => root.render(null));
  expect(activeMenus.length).toBe(0);
});

test("a click without movement and a cancelled drag after editor destruction do not apply a move", () => {
  const { editor, dropMarker, dragMarker } = fixture();
  finishTableDrag(editor, 0, 0, moveSelectedRows);
  finishTableDrag(editor, 0, 1, moveSelectedRows);
  expect(cellText(editor)).toEqual(["A", "B", "C", "D"]);
  const dropped = vi.fn();
  const cancel = startTableDrag({ editor, dropMarker, dragMarker, onMove: vi.fn(), onDrop: dropped });
  editor.destroy();
  cancel();
  cancel();
  mouse(window, "mouseup");
  expect(dropped).not.toHaveBeenCalled();
});

test("table colors update the selected cells and report both color selection and reset", async () => {
  const { editor, root, container } = fixture();
  editor.commands.setCellSelection({ anchorCell: 2 });
  const onSelect = vi.fn();
  await act(async () => root.render(<TableDragHandleDropdownColorSelector editor={editor} onSelect={onSelect} />));
  mouse(container.querySelector('button[aria-label="Green"]')!, "click");
  expect(editor.state.doc.nodeAt(2)?.attrs.background).toBe("var(--editor-colors-green-background)");
  expect(onSelect).toHaveBeenLastCalledWith("var(--editor-colors-green-background)");
  mouse(container.querySelector('button[aria-label="Reset background color"]')!, "click");
  expect(editor.state.doc.nodeAt(2)?.attrs.background).toBeNull();
  expect(onSelect).toHaveBeenLastCalledWith(null);
  mouse(container.querySelector("button")!, "click");
  expect(container.querySelector('button[aria-label="Green"]')).toBeNull();
});

test.each(["row", "column"] as const)(
  "%s menus apply insertion, duplication, clearing, header and deletion to the selected axis",
  async (axis) => {
    const { editor, root, container } = fixture();
    const { RowOptionsDropdown, ColumnOptionsDropdown } = await import("./axis-options");
    const Dropdown = axis === "row" ? RowOptionsDropdown : ColumnOptionsDropdown;
    const onClose = vi.fn();
    const selectFirst = () => {
      const table = editor.state.doc.firstChild!;
      const map = TableMap.get(table);
      const last = axis === "row" ? map.width - 1 : (map.height - 1) * map.width;
      editor.commands.setCellSelection({ anchorCell: 1 + map.map[0], headCell: 1 + map.map[last] });
    };
    const choose = (label: string) => {
      const button = [...container.querySelectorAll("button")].find(
        (candidate) => candidate.textContent?.trim() === label
      );
      if (!button) throw new Error(`Missing ${label} action`);
      mouse(button, "click");
    };
    await act(async () => root.render(<Dropdown editor={editor} onClose={onClose} />));
    act(selectFirst);
    choose("Duplicate");
    expect(cellText(editor)).toEqual(axis === "row" ? ["A", "B", "A", "B", "C", "D"] : ["A", "A", "B", "C", "C", "D"]);
    act(selectFirst);
    choose("Clear contents");
    expect(cellText(editor)).toEqual(axis === "row" ? ["", "", "A", "B", "C", "D"] : ["", "A", "B", "", "C", "D"]);
    act(selectFirst);
    choose(`Header ${axis}`);
    expect(editor.state.doc.firstChild!.firstChild!.firstChild!.type.name).toBe("tableHeader");
    choose(axis === "row" ? "Insert above" : "Insert left");
    expect(TableMap.get(editor.state.doc.firstChild!)[axis === "row" ? "height" : "width"]).toBe(4);
    act(selectFirst);
    choose(axis === "row" ? "Insert below" : "Insert right");
    expect(TableMap.get(editor.state.doc.firstChild!)[axis === "row" ? "height" : "width"]).toBe(5);
    act(selectFirst);
    choose("Delete");
    expect(TableMap.get(editor.state.doc.firstChild!)[axis === "row" ? "height" : "width"]).toBe(4);
    expect(onClose).toHaveBeenCalledTimes(6);
  }
);

test.each(["row", "column"] as const)(
  "%s deletion rejects text selections and removes a table containing only that axis",
  (axis) => {
    const { editor } = fixture();
    const remove = axis === "row" ? () => editor.commands.deleteRow() : () => editor.commands.deleteColumn();
    expect(remove()).toBe(false);
    const cells =
      axis === "row"
        ? "<tr><td><p>A</p></td><td><p>B</p></td></tr>"
        : "<tr><td><p>A</p></td></tr><tr><td><p>B</p></td></tr>";
    editor.commands.setContent(`<table>${cells}</table><p>After</p>`);
    editor.commands.setCellSelection({ anchorCell: 2 });
    expect(remove()).toBe(true);
    expect(editor.getHTML()).toBe("<p>After</p>");
  }
);

test("retained table menu actions cannot edit a read-only document", async () => {
  const { editor, root, container } = fixture();
  const { RowOptionsDropdown } = await import("./axis-options");
  const onClose = vi.fn();
  await act(async () => root.render(<RowOptionsDropdown editor={editor} onClose={onClose} />));
  editor.setEditable(false);
  const before = editor.getHTML();
  const insert = [...container.querySelectorAll("button")].find((button) => button.textContent === "Insert above")!;
  mouse(insert, "click");
  expect(editor.getHTML()).toBe(before);
  expect(onClose).not.toHaveBeenCalled();
});

test("table colors include header cells and ignore retained controls after editing is disabled", async () => {
  const { editor, root, container } = fixture();
  const onSelect = vi.fn();
  editor.commands.setCellSelection({ anchorCell: 2, headCell: 7 });
  editor.commands.toggleHeaderRow();
  await act(async () => root.render(<TableDragHandleDropdownColorSelector editor={editor} onSelect={onSelect} />));
  mouse(container.querySelector('button[aria-label="Green"]')!, "click");
  expect(editor.state.doc.nodeAt(2)?.attrs.background).toBe("var(--editor-colors-green-background)");
  expect(editor.state.doc.nodeAt(7)?.attrs.background).toBe("var(--editor-colors-green-background)");
  editor.setEditable(false);
  mouse(container.querySelector('button[aria-label="Reset background color"]')!, "click");
  expect(editor.state.doc.nodeAt(2)?.attrs.background).toBe("var(--editor-colors-green-background)");
  expect(onSelect).toHaveBeenCalledOnce();
});

test("a retained menu cannot duplicate a former table or mutate a destroyed editor", async () => {
  const { editor, root, container } = fixture();
  const { RowOptionsDropdown } = await import("./axis-options");
  const onClose = vi.fn();
  await act(async () => root.render(<RowOptionsDropdown editor={editor} onClose={onClose} />));
  editor.commands.setContent("<p>Outside</p>");
  const duplicate = [...container.querySelectorAll("button")].find((button) => button.textContent === "Duplicate")!;
  mouse(duplicate, "click");
  expect(editor.getHTML()).toBe("<p>Outside</p>");
  expect(onClose).toHaveBeenCalledOnce();
  editor.destroy();
  mouse(duplicate, "click");
  expect(onClose).toHaveBeenCalledOnce();
});

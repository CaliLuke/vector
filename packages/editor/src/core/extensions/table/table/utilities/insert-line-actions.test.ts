import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { expect, test } from "vitest";
import { Table } from "../table";
import { TableRow } from "../../table-row";
import { TableCell } from "../../table-cell";
import { TableHeader } from "../../table-header";
import { insertLineAboveTableAction, insertLineBelowTableAction } from "./insert-line-around-table-action";

test("table boundary shortcuts only leave the first or last row", () => {
  const editor = new Editor({
    extensions: [StarterKit, Table.configure({ resizable: false }), TableRow, TableCell, TableHeader],
    content: "<table><tr><td><p>First</p></td></tr><tr><td><p>Last</p></td></tr></table>",
  });
  const selectText = (text: string) => {
    editor.state.doc.descendants((node, position) => {
      if (node.isText && node.text === text) editor.commands.setTextSelection(position + 1);
    });
  };
  try {
    selectText("Last");
    expect(insertLineAboveTableAction({ editor })).toBe(false);
    selectText("First");
    expect(insertLineBelowTableAction({ editor })).toBe(false);
    expect(insertLineAboveTableAction({ editor })).toBe(true);
    expect(editor.state.doc.firstChild?.type.name).toBe("paragraph");
    expect(editor.isActive("table")).toBe(false);
    selectText("Last");
    expect(insertLineBelowTableAction({ editor })).toBe(true);
    expect(editor.state.doc.lastChild?.type.name).toBe("paragraph");
    expect(editor.isActive("table")).toBe(false);
  } finally {
    editor.destroy();
  }
});

test("table cell and header widths survive imported HTML", () => {
  const editor = new Editor({
    extensions: [StarterKit, Table.configure({ resizable: false }), TableRow, TableCell, TableHeader],
    content: '<table><tr><th colwidth="180"><p>Header</p></th><td colwidth="220"><p>Value</p></td></tr></table>',
  });
  try {
    const row = editor.state.doc.firstChild!.firstChild!;
    expect(row.child(0).attrs.colwidth).toEqual([180]);
    expect(row.child(1).attrs.colwidth).toEqual([220]);
    expect(editor.getHTML()).toContain('colwidth="180"');
    expect(editor.getHTML()).toContain('colwidth="220"');
  } finally {
    editor.destroy();
  }
});

test("leaving the first row moves to the end of the preceding paragraph without rewriting it", () => {
  const editor = new Editor({
    extensions: [
      StarterKit.configure({ trailingNode: false }),
      Table.configure({ resizable: false }),
      TableRow,
      TableCell,
      TableHeader,
    ],
    content: "<p>Before</p><table><tr><td><p>Cell</p></td></tr></table><p>After</p>",
  });
  try {
    editor.commands.setTextSelection(12);
    const html = editor.getHTML();
    expect(insertLineAboveTableAction({ editor })).toBe(true);
    expect(editor.state.selection.from).toBe(7);
    expect(editor.getHTML()).toBe(html);
  } finally {
    editor.destroy();
  }
});

test("table boundary shortcuts preserve adjacent non-paragraph blocks and ignore read-only or outside selections", () => {
  const editor = new Editor({
    extensions: [
      StarterKit.configure({ trailingNode: false }),
      Table.configure({ resizable: false }),
      TableRow,
      TableCell,
      TableHeader,
    ],
    content: "<h2>Before</h2><table><tr><td><p>Cell</p></td></tr></table><h2>After</h2>",
  });
  try {
    editor.commands.setTextSelection(12);
    const html = editor.getHTML();
    expect(insertLineAboveTableAction({ editor })).toBe(false);
    expect(insertLineBelowTableAction({ editor })).toBe(false);
    expect(editor.getHTML()).toBe(html);
    editor.commands.setTextSelection(2);
    expect(insertLineAboveTableAction({ editor })).toBe(false);
    editor.commands.setContent("<table><tr><td><p>Cell</p></td></tr></table>");
    editor.commands.setTextSelection(4);
    editor.setEditable(false);
    expect(insertLineAboveTableAction({ editor })).toBe(false);
    expect(insertLineBelowTableAction({ editor })).toBe(false);
    editor.setEditable(true);
    editor.commands.setNodeSelection(0);
    expect(insertLineAboveTableAction({ editor })).toBe(false);
  } finally {
    editor.destroy();
  }
});

test("leaving the final table row reaches an existing following paragraph or creates one at document end", () => {
  const editor = new Editor({
    extensions: [
      StarterKit.configure({ trailingNode: false }),
      Table.configure({ resizable: false }),
      TableRow,
      TableCell,
      TableHeader,
    ],
    content: "<table><tr><td><p>Cell</p></td></tr></table><p>After</p>",
  });
  try {
    editor.commands.setTextSelection(4);
    const html = editor.getHTML();
    expect(insertLineBelowTableAction({ editor })).toBe(true);
    expect(editor.state.selection.$from.parent.textContent).toBe("After");
    expect(editor.state.selection.$from.parentOffset).toBe(5);
    expect(editor.getHTML()).toBe(html);
    editor.commands.setContent("<table><tr><td><p>Cell</p></td></tr></table>");
    editor.commands.setTextSelection(4);
    expect(insertLineBelowTableAction({ editor })).toBe(true);
    expect(editor.state.doc.lastChild!.type.name).toBe("paragraph");
    expect(editor.state.selection.$from.parent.textContent).toBe("");
  } finally {
    editor.destroy();
  }
});

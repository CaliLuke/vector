import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { TaskItem, TaskList } from "@tiptap/extension-list";
import { expect, test } from "vitest";
import { ListKeymap } from "./list-keymap";

function fixture(task = false) {
  const items = task
    ? '<ul data-type="taskList"><li data-type="taskItem" data-checked="false"><p>First</p></li><li data-type="taskItem" data-checked="true"><p>Second</p></li></ul>'
    : "<ul><li><p>First</p></li><li><p>Second</p></li></ul>";
  return new Editor({
    extensions: [
      StarterKit.configure({ listKeymap: false, trailingNode: false }),
      ...(task ? [TaskList, TaskItem.configure({ nested: true })] : []),
      ListKeymap({ tabIndex: 0 }),
    ],
    content: items,
  });
}
function selectText(editor: Editor, text: string, end = false) {
  editor.state.doc.descendants((node, position) => {
    if (node.isText && node.text === text) editor.commands.setTextSelection(position + (end ? text.length : 0));
  });
}

test.each(["Backspace", "Mod-Backspace"])("%s lifts the second list item without losing its text", (key) => {
  const editor = fixture();
  try {
    selectText(editor, "Second");
    editor.commands.keyboardShortcut(key);
    expect(editor.getJSON().content?.map((node) => node.type)).toEqual(["bulletList", "paragraph"]);
    expect(editor.state.doc.lastChild!.textContent).toBe("Second");
    expect(editor.state.doc.firstChild!.textContent).toBe("First");
  } finally {
    editor.destroy();
  }
});

test.each(["Delete", "Mod-Delete"])("%s joins list items at their boundary", (key) => {
  const editor = fixture();
  try {
    selectText(editor, "First", true);
    editor.commands.keyboardShortcut(key);
    expect(editor.state.doc.firstChild!.childCount).toBe(1);
    expect(editor.state.doc.firstChild!.textContent).toBe("FirstSecond");
  } finally {
    editor.destroy();
  }
});

test.each([false, true])("list Tab and Shift-Tab preserve item content when task mode is %s", (task) => {
  const editor = fixture(task);
  try {
    selectText(editor, "Second");
    editor.commands.keyboardShortcut("Tab");
    expect(editor.state.doc.firstChild!.childCount).toBe(1);
    expect(editor.state.doc.firstChild!.firstChild!.childCount).toBe(2);
    editor.commands.keyboardShortcut("Shift-Tab");
    expect(editor.state.doc.firstChild!.childCount).toBe(2);
    expect(editor.state.doc.firstChild!.textContent).toBe("FirstSecond");
  } finally {
    editor.destroy();
  }
});

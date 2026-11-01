import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { expect, test } from "vitest";
import { observeSelectionDrag } from "./selection-drag";

const fire = (type: string, target: EventTarget = document) =>
  target.dispatchEvent(new MouseEvent(type, { bubbles: true }));

test("selection drags ignore menu clicks, report selection changes and release listeners on teardown", () => {
  const editor = new Editor({ extensions: [StarterKit], content: "<p>Draft</p>" });
  const menu = document.createElement("div");
  const icon = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  menu.append(icon);
  document.body.append(menu);
  const selected: boolean[] = [];
  const stop = observeSelectionDrag(
    editor,
    () => menu,
    (value) => selected.push(value)
  );

  try {
    fire("mousedown", icon);
    fire("mousemove");
    fire("mouseup");
    expect(selected).toEqual([]);
    fire("mousedown");
    fire("mousemove");
    expect(selected).toEqual([]);
    editor.commands.setTextSelection({ from: 1, to: 6 });
    fire("mousemove");
    fire("mousemove");
    expect(selected).toEqual([true]);
    fire("mouseup");
    expect(selected).toEqual([true, false]);
    fire("mousedown");
    stop();
    editor.destroy();
    fire("mousemove");
    fire("mouseup");
    fire("mousedown");
    expect(selected).toEqual([true, false]);
  } finally {
    stop();
    if (!editor.isDestroyed) editor.destroy();
    menu.remove();
  }
});

test("selection observation works while the menu is absent", () => {
  const editor = new Editor({ extensions: [StarterKit], content: "<p>Draft</p>" });
  const selected: boolean[] = [];
  const stop = observeSelectionDrag(
    editor,
    () => null,
    (value) => selected.push(value)
  );
  try {
    editor.commands.setTextSelection({ from: 1, to: 3 });
    document.dispatchEvent(new MouseEvent("mousedown"));
    document.dispatchEvent(new MouseEvent("mousemove"));
    expect(selected).toEqual([true]);
  } finally {
    stop();
    editor.destroy();
  }
});

import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { TableMap } from "@tiptap/pm/tables";
import { expect, test, vi } from "vitest";
import { Table } from "../../table/table";
import { TableCell } from "../../table-cell";
import { TableHeader } from "../../table-header";
import { TableRow } from "../../table-row";
import { TableInsertPlugin } from "./plugin";

async function fixture(
  content = "<p>Before</p><table><tr><td><p>A</p></td><td><p>B</p></td></tr><tr><td><p>C</p></td><td><p>D</p></td></tr></table><p>After</p>"
) {
  const host = document.createElement("div");
  document.body.append(host);
  const editor = new Editor({
    element: host,
    extensions: [
      StarterKit.configure({ trailingNode: false }),
      Table.extend({
        addProseMirrorPlugins() {
          return [TableInsertPlugin(this.editor)];
        },
      }),
      TableCell,
      TableHeader,
      TableRow,
    ],
    content,
  });
  try {
    await vi.waitFor(() => expect(host.querySelector(".table-row-insert-button")).not.toBeNull());
  } catch (error) {
    editor.destroy();
    host.remove();
    throw error;
  }
  const button = (axis: "row" | "column") => host.querySelector<HTMLButtonElement>(`.table-${axis}-insert-button`)!;
  const table = () => editor.state.doc.child(1);
  return {
    host,
    editor,
    button,
    table,
    close: () => {
      editor.destroy();
      host.remove();
    },
  };
}

test.each(["row", "column"] as const)("native %s insert activation changes only the live table", async (axis) => {
  const f = await fixture();
  try {
    f.button(axis).click();
    const map = TableMap.get(f.table());
    expect(map.width).toBe(axis === "column" ? 3 : 2);
    expect(map.height).toBe(axis === "row" ? 3 : 2);
    expect(f.editor.state.doc.firstChild!.textContent).toBe("Before");
    expect(f.editor.state.doc.lastChild!.textContent).toBe("After");
    expect(f.table().textContent).toBe("ABCD");
  } finally {
    f.close();
  }
});

test("destroying the editor cancels an active insert gesture and restores selection styles", async () => {
  const add = vi.spyOn(document, "addEventListener");
  const remove = vi.spyOn(document, "removeEventListener");
  const f = await fixture();
  try {
    document.body.style.userSelect = "text";
    f.button("column").dispatchEvent(new MouseEvent("mousedown", { button: 0, clientX: 100, bubbles: true }));
    document.dispatchEvent(new MouseEvent("mousemove", { clientX: 120 }));
    expect(document.body.style.userSelect).toBe("none");
    const listener = add.mock.calls.findLast(([type]) => type === "mousemove")![1];
    f.editor.destroy();
    expect(remove).toHaveBeenCalledWith("mousemove", listener);
    expect(document.body.style.userSelect).toBe("text");
  } finally {
    f.close();
    document.body.style.userSelect = "";
    vi.restoreAllMocks();
  }
});

function pointer(axis: "row" | "column", type: string, position: number, target: EventTarget = document) {
  target.dispatchEvent(
    new MouseEvent(type, {
      button: 0,
      bubbles: true,
      clientX: axis === "column" ? position : 0,
      clientY: axis === "row" ? position : 0,
    })
  );
}
function axisSize(f: Awaited<ReturnType<typeof fixture>>, axis: "row" | "column") {
  const map = TableMap.get(f.table());
  return axis === "column" ? map.width : map.height;
}

test.each(["row", "column"] as const)(
  "%s drag grows, removes only its empty edge, and suppresses the finishing pointer click",
  async (axis) => {
    const f = await fixture();
    const threshold = axis === "column" ? 150 : 40;
    try {
      const button = f.button(axis);
      pointer(axis, "mousedown", 500, button);
      pointer(axis, "mousemove", 500 + threshold);
      expect(axisSize(f, axis)).toBe(3);
      pointer(axis, "mousemove", 500);
      expect(axisSize(f, axis)).toBe(2);
      pointer(axis, "mousemove", 500 - threshold);
      expect(axisSize(f, axis)).toBe(2);
      expect(f.table().textContent).toBe("ABCD");
      pointer(axis, "mouseup", 500 - threshold);
      button.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 }));
      expect(axisSize(f, axis)).toBe(2);
      button.click();
      expect(axisSize(f, axis)).toBe(3);
    } finally {
      f.close();
    }
  }
);

test.each(["row", "column"] as const)("%s drag never removes the last empty axis", async (axis) => {
  const f = await fixture(
    "<p>Before</p><table><tr><td><p></p></td><td><p></p></td></tr><tr><td><p></p></td><td><p></p></td></tr></table><p>After</p>"
  );
  const threshold = axis === "column" ? 150 : 40;
  try {
    pointer(axis, "mousedown", 500, f.button(axis));
    pointer(axis, "mousemove", 500 - threshold);
    expect(axisSize(f, axis)).toBe(1);
    pointer(axis, "mousemove", 500 - 2 * threshold);
    expect(axisSize(f, axis)).toBe(1);
  } finally {
    f.close();
  }
});

test.each(["row", "column"] as const)("%s drag preserves atomic content in the final cells", async (axis) => {
  const cells =
    axis === "column" ? "<tr><td><p>A</p></td><td><hr></td></tr>" : "<tr><td><p>A</p></td></tr><tr><td><hr></td></tr>";
  const f = await fixture(`<p>Before</p><table>${cells}</table><p>After</p>`);
  try {
    pointer(axis, "mousedown", 500, f.button(axis));
    pointer(axis, "mousemove", 300);
    expect(axisSize(f, axis)).toBe(2);
    expect(f.editor.getHTML()).toContain("<hr>");
  } finally {
    f.close();
  }
});

test("table removal and read-only transitions retire old insert controls", async () => {
  const f = await fixture();
  try {
    const previous = f.button("row");
    pointer("row", "mousedown", 100, previous);
    pointer("row", "mousemove", 120);
    f.editor.setEditable(false);
    expect(f.host.querySelector(".table-row-insert-button")).toBeNull();
    expect(document.body.style.userSelect).toBe("");
    previous.click();
    expect(axisSize(f, "row")).toBe(2);
    f.editor.setEditable(true);
    const replacement = f.button("row");
    expect(replacement).not.toBe(previous);
    replacement.click();
    expect(axisSize(f, "row")).toBe(3);
    pointer("row", "mousedown", 100, replacement);
    pointer("row", "mousemove", 120);
    f.editor.commands.setContent("<p>Replacement</p>");
    pointer("row", "mousemove", 200);
    pointer("row", "mouseup", 200);
    replacement.click();
    expect(f.editor.getHTML()).toBe("<p>Replacement</p>");
    expect(document.body.style.userSelect).toBe("");
  } finally {
    f.close();
  }
});

test("window blur cancels an insert drag without changing unrelated selection styles", async () => {
  const f = await fixture();
  try {
    document.body.style.userSelect = "text";
    pointer("row", "mousedown", 100, f.button("row"));
    pointer("row", "mousemove", 120);
    window.dispatchEvent(new Event("blur"));
    pointer("row", "mousemove", 500);
    expect(axisSize(f, "row")).toBe(2);
    expect(document.body.style.userSelect).toBe("text");
  } finally {
    f.close();
    document.body.style.userSelect = "";
  }
});

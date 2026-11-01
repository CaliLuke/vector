import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { NodeSelection } from "@tiptap/pm/state";
import { expect, test, vi } from "vitest";
import { createSideMenuExtension } from "@/extensions/side-menu";

function fixture(content: string) {
  const host = document.createElement("div");
  document.body.append(host);
  const editor = new Editor({
    element: host,
    extensions: [StarterKit, createSideMenuExtension({ aiEnabled: true, dragDropEnabled: true })],
    content,
  });
  const handle = host.querySelector<HTMLButtonElement>("#drag-handle");
  if (!handle) throw new Error("Missing drag handle");
  return {
    editor,
    host,
    handle,
    close: () => {
      editor.destroy();
      host.remove();
      vi.restoreAllMocks();
      vi.unstubAllGlobals();
    },
  };
}

test("block selection uses the document hit position and teardown releases side-menu listeners", () => {
  const add = vi.spyOn(document, "addEventListener");
  const remove = vi.spyOn(document, "removeEventListener");
  const f = fixture("<p>Draft</p>");
  try {
    const paragraph = f.host.querySelector("p")!;
    Object.defineProperty(document, "elementsFromPoint", { configurable: true, value: () => [paragraph] });
    vi.spyOn(f.editor.view, "posAtCoords").mockReturnValue({ pos: 1, inside: 0 });
    f.handle.click();
    expect(f.editor.state.selection).toBeInstanceOf(NodeSelection);
    expect(f.editor.state.selection.content().content.textBetween(0, 7)).toBe("Draft");
    const listener = add.mock.calls.find(([type]) => type === "dragover")![1];
    f.editor.destroy();
    expect(f.host.querySelector("#editor-side-menu")).toBeNull();
    expect(remove).toHaveBeenCalledWith("dragover", listener);
  } finally {
    f.close();
  }
});

test("dragging serializes the selected node and preserves copy modifiers", () => {
  const f = fixture("<p>Draft</p>");
  try {
    Object.defineProperty(document, "elementsFromPoint", {
      configurable: true,
      value: () => [f.host.querySelector("p")!],
    });
    vi.spyOn(f.editor.view, "posAtCoords").mockReturnValue({ pos: 1, inside: 0 });
    const values = new Map<string, string>();
    const transfer = {
      clearData: () => values.clear(),
      setData: (type: string, value: string) => values.set(type, value),
      setDragImage: vi.fn(),
      effectAllowed: "none",
    };
    class TestDragEvent extends MouseEvent {
      dataTransfer = transfer;
    }
    vi.stubGlobal("DragEvent", TestDragEvent);
    f.handle.dispatchEvent(new TestDragEvent("dragstart", { bubbles: true, ctrlKey: true, clientY: 300 }));
    expect(values.get("text/plain")).toBe("Draft");
    expect(values.get("text/html")).toContain("Draft");
    expect(transfer.effectAllowed).toBe("copyMove");
    expect(f.editor.view.dragging?.move).toBe(true);
    expect(transfer.setDragImage).toHaveBeenCalledWith(f.host.querySelector("p"), 0, 0);
    f.handle.dispatchEvent(new TestDragEvent("dragend", { bubbles: true }));
  } finally {
    f.close();
  }
});

test.each(["#drag-handle", "#ai-handle"])("%s selects the whole blockquote at its outer edge", (selector) => {
  const f = fixture("<blockquote><p>Draft</p></blockquote>");
  try {
    const blockquote = f.host.querySelector("blockquote")!;
    Object.defineProperty(document, "elementsFromPoint", { configurable: true, value: () => [blockquote] });
    vi.spyOn(blockquote, "getBoundingClientRect").mockReturnValue(new DOMRect(10, 20, 100, 40));
    const position = vi
      .spyOn(f.editor.view, "posAtCoords")
      .mockImplementation(({ left }) => ({ pos: 2, inside: left === 11 ? 0 : 1 }));
    f.host.querySelector<HTMLButtonElement>(selector)!.click();
    expect(f.editor.state.selection).toBeInstanceOf(NodeSelection);
    expect(f.editor.state.selection.$from.nodeAfter?.type.name).toBe("blockquote");
    expect(f.editor.state.selection.content().content.textBetween(0, 9)).toBe("Draft");
    expect(position).toHaveBeenLastCalledWith({ left: 11, top: 21 });
  } finally {
    f.close();
  }
});

test.each(["#drag-handle", "#ai-handle"])(
  "%s preserves selection when the hit has no document position",
  (selector) => {
    const f = fixture("<p>Draft</p>");
    try {
      const paragraph = f.host.querySelector("p")!;
      Object.defineProperty(document, "elementsFromPoint", { configurable: true, value: () => [paragraph] });
      vi.spyOn(paragraph, "getBoundingClientRect").mockReturnValue(new DOMRect(10, 20, 100, 40));
      const position = vi.spyOn(f.editor.view, "posAtCoords").mockReturnValue(null);
      f.editor.commands.setTextSelection(3);
      f.host.querySelector<HTMLButtonElement>(selector)!.click();
      expect(f.editor.state.selection.from).toBe(3);
      expect(f.editor.state.selection.to).toBe(3);
      expect(position).toHaveBeenLastCalledWith({ left: 84, top: 21 });
    } finally {
      f.close();
    }
  }
);

import { Editor } from "@tiptap/core";
import { afterEach, describe, expect, it } from "vitest";
import { CustomCodeInlineExtension } from "../code-inline";
import { CustomStarterKitExtension } from "../starter-kit";
import { CustomCodeBlockExtension } from "./index";
import { CustomCodeBlockExtensionWithoutProps } from "./without-props";

let editor: Editor | undefined;
afterEach(() => {
  editor?.destroy();
  editor = undefined;
});

function paste(current: Editor, text: string, metadata = "") {
  const event = new Event("paste", { bubbles: true, cancelable: true });
  Object.defineProperty(event, "clipboardData", {
    value: {
      getData: (format: string) => (format === "text/plain" ? text : format === "vscode-editor-data" ? metadata : ""),
      files: [],
    },
  });
  current.view.dom.dispatchEvent(event);
}

describe.each([
  ["React code block", CustomCodeBlockExtension],
  ["plain code block", CustomCodeBlockExtensionWithoutProps],
] as const)("%s keyboard behavior", (_name, codeBlock) => {
  function createEditor() {
    editor = new Editor({
      extensions: [
        CustomStarterKitExtension({ enableHistory: false }).configure({ trailingNode: false, gapcursor: false }),
        CustomCodeInlineExtension,
        codeBlock,
      ],
      content: { type: "doc", content: [{ type: "codeBlock", content: [{ type: "text", text: "hello" }] }] },
    });
    return editor;
  }

  it("inserts a literal tab at the cursor without replacing surrounding code", () => {
    const current = createEditor();
    current.commands.setTextSelection(3);
    current.view.dom.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", bubbles: true, cancelable: true }));
    expect(current.getText()).toBe("he\tllo");
  });

  it("leaves the first code block upward by creating a paragraph before it", () => {
    const current = createEditor();
    current.commands.setTextSelection(1);
    current.view.dom.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowUp", bubbles: true, cancelable: true }));
    expect(current.getJSON().content?.map((node) => node.type)).toEqual(["paragraph", "codeBlock"]);
    expect(current.state.selection.$from.parent.type.name).toBe("paragraph");
    expect(current.getText()).toContain("hello");
  });

  it("leaves the last code block downward by creating a paragraph after it", () => {
    const current = createEditor();
    current.commands.setTextSelection(6);
    current.view.dom.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }));
    expect(current.getJSON().content?.map((node) => node.type)).toEqual(["codeBlock", "paragraph"]);
    expect(current.state.selection.$from.parent.type.name).toBe("paragraph");
    expect(current.getText()).toContain("hello");
  });
  it("leaves selections, interior arrow movement, and ordinary paragraphs unchanged", () => {
    const current = createEditor();
    current.commands.setTextSelection({ from: 2, to: 4 });
    current.commands.keyboardShortcut("Tab");
    expect(current.getText()).toBe("hello");
    current.commands.setTextSelection(3);
    current.commands.keyboardShortcut("ArrowUp");
    current.commands.keyboardShortcut("ArrowDown");
    expect(current.getJSON().content?.map((node) => node.type)).toEqual(["codeBlock"]);
    current.commands.setContent("<p>Ordinary</p>");
    current.commands.setTextSelection(3);
    const paragraphHTML = current.getHTML();
    current.commands.keyboardShortcut("Tab");
    expect(current.getHTML()).toBe(paragraphHTML);
  });

  it("creates a code block from VS Code metadata and normalizes line endings", () => {
    const current = createEditor();
    current.commands.setContent("<p></p>");
    current.commands.setTextSelection(1);
    paste(current, "const value = 1;\r\nvalue++;", JSON.stringify({ mode: "javascript" }));
    const block = current.getJSON().content?.find((node) => node.type === "codeBlock");
    expect(block?.attrs?.language).toBe("javascript");
    expect(block?.content).toEqual([{ type: "text", text: "const value = 1;\nvalue++;" }]);
  });

  it("leaves ordinary and invalid-language pastes to the normal text handler", () => {
    const current = createEditor();
    current.commands.setContent("<p></p>");
    current.commands.setTextSelection(1);
    paste(current, "ordinary ");
    paste(current, "text", JSON.stringify({ mode: 42 }));
    expect(current.getText()).toBe("ordinary text");
    expect(current.getJSON().content?.every((node) => node.type === "paragraph")).toBe(true);
  });

  it("replaces an inline code selection while retaining the code mark", () => {
    const current = createEditor();
    current.commands.setContent("<p><code>hello</code></p>");
    current.commands.setTextSelection({ from: 2, to: 4 });
    paste(current, "X");
    expect(current.getText()).toBe("hXlo");
    expect(current.getJSON().content?.[0].content?.[0].marks).toContainEqual({ type: "code" });
  });

  it("pastes into an existing code block without creating a nested block", () => {
    const current = createEditor();
    current.commands.setTextSelection(6);
    paste(current, " world", JSON.stringify({ mode: "javascript" }));
    expect(current.getText()).toBe("hello world");
    expect(current.getJSON().content?.map((node) => node.type)).toEqual(["codeBlock"]);
  });
});

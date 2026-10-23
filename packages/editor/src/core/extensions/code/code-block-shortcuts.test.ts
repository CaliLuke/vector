import { Editor } from "@tiptap/core";
import { afterEach, describe, expect, it } from "vitest";
import { CustomStarterKitExtension } from "../starter-kit";
import { CustomCodeBlockExtension } from "./index";
import { CustomCodeBlockExtensionWithoutProps } from "./without-props";

let editor: Editor | undefined;
afterEach(() => {
  editor?.destroy();
  editor = undefined;
});

describe.each([
  ["React code block", CustomCodeBlockExtension],
  ["plain code block", CustomCodeBlockExtensionWithoutProps],
] as const)("%s keyboard behavior", (_name, codeBlock) => {
  function createEditor() {
    editor = new Editor({
      extensions: [CustomStarterKitExtension({ enableHistory: false }).configure({ trailingNode: false }), codeBlock],
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
});

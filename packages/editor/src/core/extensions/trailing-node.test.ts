import { Editor } from "@tiptap/core";
import { afterEach, describe, expect, it } from "vitest";
import { CustomStarterKitExtension } from "./starter-kit";
import { TrailingNode } from "./trailing-node";

let editor: Editor | undefined;
afterEach(() => {
  editor?.destroy();
  editor = undefined;
});

function createEditor(extension = TrailingNode) {
  editor = new Editor({
    extensions: [CustomStarterKitExtension({ enableHistory: false }).configure({ trailingNode: false }), extension],
  });
  return editor;
}

describe("exported trailing-node behavior", () => {
  it("appends one paragraph after a heading and leaves it unchanged during selection moves", () => {
    const current = createEditor();
    current.commands.setContent("<h1>Heading</h1>");
    expect(current.getJSON().content?.map((node) => node.type)).toEqual(["heading", "paragraph"]);
    expect(current.getText()).toContain("Heading");
    current.commands.setTextSelection(1);
    current.commands.setTextSelection(2);
    expect(current.getJSON().content?.map((node) => node.type)).toEqual(["heading", "paragraph"]);
  });

  it("does not add another paragraph when the document already ends in one", () => {
    const current = createEditor();
    current.commands.setContent("<p>Paragraph</p>");
    current.commands.insertContent("more");
    expect(current.getJSON().content?.map((node) => node.type)).toEqual(["paragraph"]);
  });

  it("honors configured node types that must not receive a trailing paragraph", () => {
    const current = createEditor(TrailingNode.configure({ notAfter: ["heading", "paragraph"] }));
    current.commands.setContent("<h1>No trailer</h1>");
    expect(current.getJSON().content?.map((node) => node.type)).toEqual(["heading"]);
  });
});

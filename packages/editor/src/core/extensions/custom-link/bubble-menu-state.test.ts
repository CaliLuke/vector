import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { expect, test, vi } from "vitest";
import { CustomLinkExtension } from "./index";

test("link bubble-menu state changes only on command execution without changing the document", () => {
  const editor = new Editor({
    extensions: [StarterKit.configure({ link: false }), CustomLinkExtension],
    content: "<p>Text</p>",
  });
  try {
    const transaction = vi.fn();
    editor.on("transaction", transaction);
    expect(editor.can().setLinkBubbleMenuOpen(true)).toBe(true);
    expect(editor.storage.link.isBubbleMenuOpen).toBe(false);
    editor.commands.setLinkBubbleMenuOpen(true);
    expect(editor.storage.link.isBubbleMenuOpen).toBe(true);
    editor.commands.setLinkBubbleMenuOpen(false);
    expect(editor.storage.link.isBubbleMenuOpen).toBe(false);
    expect(editor.getHTML()).toBe("<p>Text</p>");
    expect(transaction).not.toHaveBeenCalled();
  } finally {
    editor.destroy();
  }
});

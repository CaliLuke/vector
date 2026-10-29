import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { find } from "linkifyjs";
import { assert, test, vi } from "vitest";
import { CustomLinkExtension } from "./extension";

const warnings = vi.hoisted(() => vi.spyOn(console, "warn").mockImplementation(() => {}));

const editorStarterKit = StarterKit.configure({ link: false });

function hasCustomLinkMark(editor: Editor, text: string) {
  return (editor.getJSON().content ?? [])
    .flatMap((node) => node.content ?? [])
    .some(
      (node) =>
        "text" in node &&
        node.text.includes(text) &&
        node.marks?.some((mark) => mark.type === "link" && mark.attrs?.href === text)
    );
}

test("late custom protocols rebuild linkify and preserve existing editor behavior", () => {
  const defaultEditor = new Editor({
    element: document.createElement("div"),
    extensions: [editorStarterKit, CustomLinkExtension],
  });
  let lateCustomEditor: Editor | undefined;
  let repeatedCustomEditor: Editor | undefined;
  let postDestroyCustomEditor: Editor | undefined;

  try {
    defaultEditor.commands.insertContent("https://example.com ");
    assert.strictEqual(hasCustomLinkMark(defaultEditor, "https://example.com"), true);
    assert.strictEqual(
      find("https://example.com").some((link) => link.isLink),
      true
    );

    lateCustomEditor = new Editor({
      element: document.createElement("div"),
      extensions: [
        editorStarterKit,
        CustomLinkExtension.configure({
          protocols: ["slack"],
          validate: () => true,
        }),
      ],
    });
    lateCustomEditor.commands.insertContent("slack://channel ");
    assert.strictEqual(hasCustomLinkMark(lateCustomEditor, "slack://channel"), true);
    assert.strictEqual(
      find("slack://channel").some((link) => link.isLink),
      true
    );
    defaultEditor.commands.insertContent("https://another.example ");
    assert.strictEqual(hasCustomLinkMark(defaultEditor, "https://another.example"), true);
    assert.strictEqual(
      find("https://another.example").some((link) => link.isLink),
      true
    );

    repeatedCustomEditor = new Editor({
      element: document.createElement("div"),
      extensions: [
        editorStarterKit,
        CustomLinkExtension.configure({
          protocols: ["slack"],
          validate: () => true,
        }),
      ],
    });
    repeatedCustomEditor.commands.insertContent("slack://another-channel ");
    assert.strictEqual(hasCustomLinkMark(repeatedCustomEditor, "slack://another-channel"), true);
    assert.strictEqual(
      find("slack://another-channel").some((link) => link.isLink),
      true
    );
    lateCustomEditor.destroy();
    lateCustomEditor = undefined;
    repeatedCustomEditor.commands.insertContent("slack://surviving-editor ");
    assert.strictEqual(hasCustomLinkMark(repeatedCustomEditor, "slack://surviving-editor"), true);

    postDestroyCustomEditor = new Editor({
      element: document.createElement("div"),
      extensions: [
        editorStarterKit,
        CustomLinkExtension.configure({
          protocols: ["slack"],
          validate: () => true,
        }),
      ],
    });
    postDestroyCustomEditor.commands.insertContent("slack://after-destroy ");
    assert.strictEqual(hasCustomLinkMark(postDestroyCustomEditor, "slack://after-destroy"), true);

    assert.strictEqual(warnings.mock.calls.length, 1);
    assert.match(warnings.mock.calls[0].join(" "), /already initialized.*custom scheme "slack"/);
  } finally {
    defaultEditor.destroy();
    lateCustomEditor?.destroy();
    repeatedCustomEditor?.destroy();
    postDestroyCustomEditor?.destroy();
    warnings.mockRestore();
  }
});

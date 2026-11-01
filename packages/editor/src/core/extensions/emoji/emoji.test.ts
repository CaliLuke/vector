import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { createEditorHookProps } from "../../../../tests/helpers/editor-hook-props";
import { UtilityExtension } from "../utility";
import { Emoji } from "./emoji";

// jsdom does not implement the clipboard constructors used by Tiptap's paste command.
class TestClipboardEvent extends Event {
  readonly clipboardData: DataTransfer | null;
  constructor(type: string, options: ClipboardEventInit = {}) {
    super(type, options);
    this.clipboardData = options.clipboardData ?? null;
  }
}
beforeEach(() => {
  vi.stubGlobal("ClipboardEvent", TestClipboardEvent);
});
let editor: Editor | undefined;
afterEach(() => {
  editor?.destroy();
  editor = undefined;
  vi.unstubAllGlobals();
});

function fixture(content = "<p></p>") {
  editor = new Editor({
    extensions: [
      StarterKit.configure({ trailingNode: false }),
      UtilityExtension({ ...createEditorHookProps({ editable: true }), isEditable: true, isTouchDevice: false }),
      Emoji.configure({
        emojis: [
          { name: "smile", emoji: "😀", shortcodes: ["smile", "happy"], tags: [] },
          { name: "heart", emoji: "❤️", shortcodes: ["heart"], tags: [] },
        ],
      }),
    ],
    content,
  });
  return editor;
}

function typeText(current: Editor, text: string) {
  for (const character of text) {
    const { from, to } = current.state.selection;
    const transaction = () => current.state.tr.insertText(character, from, to);
    const handled = current.view.someProp("handleTextInput", (handler) =>
      handler(current.view, from, to, character, transaction)
    );
    if (!handled) current.view.dispatch(transaction());
  }
}

test("typing a shortcode preserves surrounding text and the active mark for continued typing", () => {
  const current = fixture("<p><strong>Before </strong>after</p>");
  current.commands.setTextSelection(8);
  typeText(current, ":smile: next");
  expect(current.getText()).toBe("Before 😀 nextafter");
  expect(current.view.dom.querySelector('[data-type="emoji"]')?.getAttribute("data-name")).toBe("smile");
  expect(current.getHTML()).toContain("<strong> next</strong>after");
  expect(current.state.selection.$from.nodeAfter?.text).toBe("after");
});

test("unknown input shortcodes remain literal text", () => {
  const current = fixture();
  typeText(current, ":unavailable:");
  expect(current.getText()).toBe(":unavailable:");
  expect(current.getHTML()).not.toContain('data-type="emoji"');
});

test("pasting multiple shortcodes preserves the selection after the complete pasted text", () => {
  const current = fixture("<p><strong>Before </strong>after</p>");
  current.commands.setTextSelection(8);
  current.view.pasteText(":smile: + :heart:");
  expect(current.getText()).toBe("Before 😀 + ❤️after");
  expect(current.state.selection.$from.nodeAfter?.text).toBe("after");
  expect(current.view.dom.querySelectorAll('[data-type="emoji"]')).toHaveLength(2);
  typeText(current, " next");
  expect(current.getText()).toBe("Before 😀 + ❤️ nextafter");
  expect(current.getHTML()).toContain(" next</strong>after");
});

test("pasting an unknown shortcode keeps it alongside a recognized shortcode", () => {
  const current = fixture();
  current.view.pasteText(":missing: :smile:");
  expect(current.getText()).toBe(":missing: 😀");
  expect(current.view.dom.querySelectorAll('[data-type="emoji"]')).toHaveLength(1);
});

test("emoji commands reject unknown names and store the canonical name for aliases", () => {
  const current = fixture("<p><strong>Bold</strong></p>");
  current.commands.setTextSelection(5);
  const before = current.getJSON();
  expect(current.commands.setEmoji("missing")).toBe(false);
  expect(current.getJSON()).toEqual(before);
  expect(current.commands.setEmoji("happy")).toBe(true);
  typeText(current, " next");
  expect(current.view.dom.querySelector('[data-type="emoji"]')?.getAttribute("data-name")).toBe("smile");
  expect(current.getHTML()).toContain("<strong> next</strong>");
});

test("shortcode rules leave code blocks as literal text", () => {
  const current = fixture("<pre><code></code></pre>");
  typeText(current, ":smile:");
  current.view.pasteText(" :heart:");
  expect(current.getText()).toBe(":smile: :heart:");
  expect(current.view.dom.querySelector('[data-type="emoji"]')).toBeNull();
});

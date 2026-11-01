import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { afterEach, expect, test } from "vitest";
import { customFindSuggestionMatch } from "./find-suggestion-match";
import { insertEmptyParagraphAtNodeBoundaries } from "./insert-empty-paragraph-at-node-boundary";
import { handleBackspace } from "../extensions/custom-list-keymap/list-helpers";

const editors: Editor[] = [];
afterEach(() => {
  for (const editor of editors.splice(0)) editor.destroy();
});
function documentEditor(content: string) {
  const editor = new Editor({ extensions: [StarterKit.configure({ trailingNode: false })], content });
  editors.push(editor);
  return editor;
}
function selectText(editor: Editor, text: string, offset = 0) {
  let found = false;
  editor.state.doc.descendants((node, position) => {
    if (!found && node.isText && node.text === text) {
      editor.commands.setTextSelection(position + offset);
      found = true;
    }
  });
  expect(found).toBe(true);
}

test.each(["up", "down"] as const)("arrow %s at a code block boundary creates a usable paragraph", (direction) => {
  const editor = documentEditor("<pre><code>Code</code></pre>");
  selectText(editor, "Code");
  expect(insertEmptyParagraphAtNodeBoundaries(direction, "codeBlock")({ editor })).toBe(true);
  expect(editor.state.selection.$from.parent.type.name).toBe("paragraph");
  expect(editor.getHTML()).toBe(
    direction === "up" ? "<p></p><pre><code>Code</code></pre>" : "<pre><code>Code</code></pre><p></p>"
  );
});

test.each(["up", "down"] as const)("arrow %s reuses an adjacent paragraph without changing content", (direction) => {
  const editor = documentEditor("<p>Before</p><pre><code>Code</code></pre><p>After</p>");
  selectText(editor, "Code");
  const before = editor.getJSON();
  expect(insertEmptyParagraphAtNodeBoundaries(direction, "codeBlock")({ editor })).toBe(true);
  expect(editor.state.selection.$from.parent.textContent).toBe(direction === "up" ? "Before" : "After");
  expect(editor.getJSON()).toEqual(before);
});

test("boundary navigation leaves a non-target selection untouched", () => {
  const editor = documentEditor("<p>Text</p>");
  selectText(editor, "Text");
  const before = editor.state;
  expect(insertEmptyParagraphAtNodeBoundaries("down", "codeBlock")({ editor })).toBe(false);
  expect(editor.state).toBe(before);
});

test.each([
  { html: "<p>@<strong>ali</strong>ce</p>", text: "ce", query: "alice", allowSpaces: false, startOfLine: false },
  { html: "<p>@Alice Smith</p>", text: "@Alice Smith", query: "Alice Smith", allowSpaces: true, startOfLine: true },
  { html: "<p>email@alice</p>", text: "email@alice", query: null, allowSpaces: false, startOfLine: false },
  { html: "<p>Text @alice</p>", text: "Text @alice", query: null, allowSpaces: false, startOfLine: true },
])(
  "suggestions respect marked text, spaces and trigger prefixes: $html",
  ({ html, text, query, allowSpaces, startOfLine }) => {
    const editor = documentEditor(html);
    selectText(editor, text, text.length);
    const match = customFindSuggestionMatch({
      char: "@",
      allowSpaces,
      allowToIncludeChar: false,
      allowedPrefixes: [" "],
      startOfLine,
      $position: editor.state.selection.$from,
    });
    expect(match?.query ?? null).toBe(query);
    if (match) expect(editor.state.doc.textBetween(match.range.from, match.range.to)).toBe(match.text);
  }
);

test("backspace preserves ordinary text and range selections for the editor's default handling", () => {
  const editor = documentEditor("<p>Text</p><ul><li><p>Item</p></li></ul>");
  selectText(editor, "Text", 2);
  const before = editor.getJSON();
  expect(handleBackspace(editor, "listItem", ["bulletList", "orderedList"])).toBe(false);
  editor.commands.setTextSelection({ from: 1, to: 3 });
  expect(handleBackspace(editor, "listItem", ["bulletList", "orderedList"])).toBe(false);
  expect(editor.getJSON()).toEqual(before);
});

test("backspace lifts a following list item without losing its text", () => {
  const editor = documentEditor("<ul><li><p>First</p></li><li><p>Second</p></li></ul>");
  selectText(editor, "Second");
  expect(handleBackspace(editor, "listItem", ["bulletList", "orderedList"])).toBe(true);
  expect(editor.getHTML()).toBe("<ul><li><p>First</p></li></ul><p>Second</p>");
});

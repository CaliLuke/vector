import { Editor } from "@tiptap/core";
import { common, createLowlight } from "lowlight";
import { afterEach, expect, test } from "vitest";
import { CustomStarterKitExtension } from "../starter-kit";
import { CodeBlockLowlight } from "./code-block-lowlight";
import { LowlightPlugin } from "./lowlight-plugin";

let editor: Editor | undefined;
afterEach(() => {
  editor?.destroy();
  editor = undefined;
});

function createEditor(language: string | null, defaultLanguage: string | null = null) {
  const lowlight = createLowlight(common);
  lowlight.registerAlias({ plaintext: ["literal-text"] });
  editor = new Editor({
    extensions: [
      CustomStarterKitExtension({ enableHistory: false }).configure({ trailingNode: false }),
      CodeBlockLowlight.configure({ lowlight, defaultLanguage }),
    ],
    content: {
      type: "doc",
      content: [
        { type: "paragraph", content: [{ type: "text", text: "before" }] },
        { type: "codeBlock", attrs: { language }, content: [{ type: "text", text: "const value = 42;" }] },
      ],
    },
  });
  return editor;
}

test("explicit language aliases use the configured highlighter instead of automatic detection", () => {
  const current = createEditor("literal-text");
  expect(current.view.dom.querySelector("code")?.textContent).toBe("const value = 42;");
  expect(current.view.dom.querySelector(".hljs-keyword")).toBeNull();
  const codePosition = current.state.doc.firstChild!.nodeSize;
  current.view.dispatch(current.state.tr.setNodeMarkup(codePosition, undefined, { language: "javascript" }));
  expect(current.view.dom.querySelector(".hljs-keyword")?.textContent).toBe("const");
  expect(current.view.dom.querySelector(".hljs-number")?.textContent).toBe("42");
});

test("highlight decorations follow edits outside code and are rebuilt after whole-document replacement", () => {
  const current = createEditor("javascript");
  current.commands.setTextSelection(1);
  current.commands.insertContent("prefix ");
  expect(current.view.dom.querySelector(".hljs-keyword")?.textContent).toBe("const");
  current.commands.setContent({
    type: "doc",
    content: [
      { type: "paragraph", content: [{ type: "text", text: "paragraph" }] },
      { type: "codeBlock", attrs: { language: "javascript" }, content: [{ type: "text", text: "let other = 99;" }] },
    ],
  });
  expect(current.view.dom.querySelector(".hljs-keyword")?.textContent).toBe("let");
  expect(current.view.dom.querySelector(".hljs-number")?.textContent).toBe("99");
  current.commands.setContent("<p>plain text</p>");
  expect(current.view.dom.querySelector("[class^='hljs-']")).toBeNull();
});

test("missing languages honor the configured default and unsupported languages can be detected", () => {
  const current = createEditor(null, "plaintext");
  expect(current.view.dom.querySelector(".hljs-keyword")).toBeNull();
  const codePosition = current.state.doc.firstChild!.nodeSize;
  current.view.dispatch(current.state.tr.setNodeMarkup(codePosition, undefined, { language: "not-registered" }));
  expect(current.view.dom.querySelector("code")?.textContent).toBe("const value = 42;");
  expect(current.view.dom.querySelector("[class^='hljs-']")).not.toBeNull();
});

test("a missing highlighter is rejected when constructing the extension plugin", () => {
  expect(() => LowlightPlugin({ name: "codeBlock", lowlight: null, defaultLanguage: null })).toThrow(
    "provide an instance of lowlight"
  );
});

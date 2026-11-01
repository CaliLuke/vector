import { Editor } from "@tiptap/core";
import { StarterKit } from "@tiptap/starter-kit";
import { expect, test } from "vitest";
import { CustomTypographyExtension } from "./index";
import type { TypographyOptions } from "./inputRules";

function typeText(text: string, options: Partial<TypographyOptions> = {}) {
  const editor = new Editor({ extensions: [StarterKit, CustomTypographyExtension.configure(options)] });
  try {
    for (const character of text) {
      const { from, to } = editor.state.selection;
      const transaction = () => editor.state.tr.insertText(character, from, to);
      const handled = editor.view.someProp("handleTextInput", (handler) =>
        handler(editor.view, from, to, character, transaction)
      );
      if (!handled) editor.view.dispatch(transaction());
    }
    return editor.getText();
  } finally {
    editor.destroy();
  }
}

test.each([
  ["--", "—"],
  ["=>", "⇒"],
  ["...", "…"],
  ["<-", "←"],
  ["->", "→"],
  ["(c)", "©"],
  ["(tm)", "™"],
  ["(sm)", "℠"],
  ["(r)", "®"],
  ["1/2 ", "½ "],
  ["+/-", "±"],
  ["!=", "≠"],
  ["<<", "«"],
  [">>", "»"],
  ["2x3", "2×3"],
  ["^2", "²"],
  ["^3", "³"],
  ["1/4 ", "¼ "],
  ["3/4 ", "¾ "],
])("typing %s applies its default typography rule", (text, expected) => {
  expect(typeText(text)).toBe(expected);
});

test("typography options disable selected rules and override replacements without affecting other rules", () => {
  expect(typeText("-- ...", { emDash: false, ellipsis: "[ellipsis]" })).toBe("-- [ellipsis]");
  expect(typeText("=>", { impliesArrowRight: "then" })).toBe("then");
});

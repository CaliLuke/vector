import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { convertHTMLToMarkdown } = await import("../../packages/utils/src/editor/markdown-parser/root.ts");
const convert = (description_html) =>
  convertHTMLToMarkdown({ description_html, metaData: { file_assets: [], user_mentions: [] } });

void test("markdown export retains bold text nested within italics and removes underline without discarding its children", () => {
  assert.equal(convert("<p><em>Some <strong>bold</strong> text</em></p>"), "_Some **bold** text_\n");
  assert.equal(convert("<p><u><strong>Bold</strong> and <em>italic</em></u></p>"), "**Bold** and _italic_\n");
});
void test("markdown task lists keep checkbox state and content on the same line", () => {
  assert.equal(
    convert(
      '<ul data-type="taskList"><li data-type="taskItem"><label><input type="checkbox" checked><span></span></label><div><p>Done</p></div></li><li data-type="taskItem"><label><input type="checkbox"><span></span></label><div><p>Later</p></div></li></ul>'
    ),
    "* [x] Done\n* [ ] Later\n"
  );
});
void test("markdown export preserves ordinary lists and task items without checkbox wrappers", () => {
  assert.equal(convert("<ul><li>First</li><li>Second</li></ul>"), "* First\n* Second\n");
  assert.equal(convert('<ul><li data-type="taskItem"><p>No checkbox</p></li></ul>'), "* No checkbox\n");
});

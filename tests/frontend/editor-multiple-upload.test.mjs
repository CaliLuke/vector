import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { registerIssueRealEditor } from "./helpers/issue-real-editor.mjs";

registerWebModuleHooks();
registerIssueRealEditor();
const { uploadFirstFileAndInsertRemaining } = await import("../../packages/editor/src/core/hooks/use-file-upload.ts");

function fixture() {
  const upload = Promise.withResolvers();
  const inserted = [];
  const files = [new File(["first"], "first.png"), new File(["next"], "next.png")];
  const editor = {
    state: { doc: { content: { size: 5 } } },
    commands: { insertImageComponent: (node) => inserted.push(node) },
  };
  const result = uploadFirstFileAndInsertRemaining({
    editor,
    filesList: files,
    pos: 5,
    type: "image",
    uploader: () => upload.promise,
  });
  return { upload, inserted, files, result };
}

await test("multiple-file insertion is immediate while completion waits for the first upload", async () => {
  const f = fixture();
  let completed = false;
  const completion = f.result.then(() => {
    completed = true;
    return undefined;
  });
  assert.deepEqual(f.inserted, [{ file: f.files[1], pos: 5, event: "drop" }]);
  await Promise.resolve();
  assert.equal(completed, false);
  f.upload.resolve();
  await completion;
  assert.equal(completed, true);
});

await test("the first upload's rejection reaches the caller after remaining files are inserted", async () => {
  const f = fixture();
  const failure = new Error("upload unavailable");
  const rejected = assert.rejects(f.result, (error) => error === failure);
  f.upload.reject(failure);
  await rejected;
  assert.deepEqual(f.inserted, [{ file: f.files[1], pos: 5, event: "drop" }]);
});

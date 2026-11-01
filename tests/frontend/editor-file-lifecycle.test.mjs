import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import { pathToFileURL } from "node:url";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { registerIssueRealEditor } from "./helpers/issue-real-editor.mjs";

registerWebModuleHooks();
registerIssueRealEditor();
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.startsWith("@/plane-editor/"))
      return nextResolve(
        new URL(`../../packages/editor/src/ce/${specifier.slice("@/plane-editor/".length)}.ts`, import.meta.url).href,
        context
      );
    return nextResolve(specifier, context);
  },
});
const requireEditor = createRequire(new URL("../../packages/editor/package.json", import.meta.url));
const { Schema } = await import(pathToFileURL(requireEditor.resolve("@tiptap/pm/model").replace(/\.cjs$/, ".js")).href);
const { EditorState } = await import(
  pathToFileURL(requireEditor.resolve("@tiptap/pm/state").replace(/\.cjs$/, ".js")).href
);
const { TrackFileDeletionPlugin } = await import("../../packages/editor/src/core/plugins/file/delete.ts");
const { TrackFileRestorationPlugin } = await import("../../packages/editor/src/core/plugins/file/restore.ts");
const schema = new Schema({
  nodes: {
    doc: { content: "block*" },
    text: { group: "inline" },
    paragraph: { group: "block", content: "inline*" },
    image: { group: "block", attrs: { id: { default: null }, src: { default: null }, status: { default: null } } },
    imageComponent: {
      group: "block",
      attrs: { id: { default: null }, src: { default: null }, status: { default: null } },
    },
  },
});
const image = (src, id = "image", status, type = "imageComponent") => schema.node(type, { src, id, status });
const document = (nodes = []) => EditorState.create({ schema, doc: schema.node("doc", null, nodes) });
function fixture() {
  const changes = [];
  const deleted = new Map();
  const editor = {
    storage: { image: { deletedImageSet: deleted }, imageComponent: { deletedImageSet: deleted } },
    commands: { updateAssetsList: (change) => changes.push(change) },
  };
  return { editor, changes, deleted };
}
function apply(plugin, before, after, transactions = [{ docChanged: true, getMeta() {} }]) {
  return plugin.spec.appendTransaction.call(plugin, transactions, before, after);
}

void test("file restoration runs once per final source across multiple transactions and duplicate nodes", async () => {
  const { editor, changes, deleted } = fixture();
  const src = "https://assets.test/image";
  deleted.set(src, true);
  const pending = Promise.withResolvers();
  const restored = [];
  const plugin = TrackFileRestorationPlugin(editor, (source) => {
    restored.push(source);
    return pending.promise;
  });
  const before = document();
  const after = document([image(src, "first"), image(src, "second")]);
  apply(plugin, before, after, [{ docChanged: true }, { docChanged: true }]);
  assert.deepEqual(restored, [src]);
  assert.deepEqual(
    changes.map(({ asset }) => asset.id),
    ["first", "second"]
  );
  assert.equal(deleted.get(src), true);
  pending.resolve();
  await pending.promise;
  assert.equal(deleted.get(src), false);
});

void test("failed restorations retain deletion state and can retry a later undo", async (t) => {
  const { editor, deleted } = fixture();
  const src = "https://assets.test/retry";
  deleted.set(src, true);
  const failure = new Error("restore unavailable");
  const errors = t.mock.method(console, "error", () => {});
  let reject = true;
  const plugin = TrackFileRestorationPlugin(editor, async () => {
    if (reject) throw failure;
  });
  apply(plugin, document(), document([image(src)]));
  await Promise.resolve();
  assert.equal(deleted.get(src), true);
  assert.equal(errors.mock.calls[0].arguments[1], failure);
  reject = false;
  apply(plugin, document(), document([image(src)]));
  await Promise.resolve();
  assert.equal(deleted.get(src), false);
});

void test("file lifecycle ignores malformed sources, retains shared references, and protects provisional copies", async () => {
  const { editor, changes } = fixture();
  const removed = [];
  const restored = [];
  const deletion = TrackFileDeletionPlugin(editor, async (src) => {
    removed.push(src);
  });
  const restoration = TrackFileRestorationPlugin(editor, async (src) => {
    restored.push(src);
  });
  apply(deletion, document([image("shared", "one"), image("shared", "two")]), document([image("shared", "two")]));
  for (const status of ["duplicating", "duplication-failed"])
    apply(deletion, document([image("source", "copy", status)]), document());
  const invalid = document([image(42), image(null), image({}), image("")]);
  apply(deletion, invalid, document());
  apply(restoration, document(), invalid);
  assert.deepEqual(removed, []);
  assert.deepEqual(restored, []);
  assert.deepEqual(changes, []);
  apply(deletion, document([image("owned", "valid", "uploaded")]), document());
  assert.deepEqual(removed, ["owned"]);
  assert.deepEqual(changes, [{ idToRemove: "valid" }]);
});

void test("private inserted images register metadata without a public restore and deletion honors skip metadata", () => {
  const { editor, changes, deleted } = fixture();
  deleted.set("private-id", true);
  const restored = [];
  const restoration = TrackFileRestorationPlugin(editor, async (src) => {
    restored.push(src);
  });
  apply(restoration, document(), document([image("private-id", "private-node")]));
  assert.deepEqual(restored, []);
  assert.equal(changes[0].asset.src, "private-id");
  const removed = [];
  const deletion = TrackFileDeletionPlugin(editor, async (src) => {
    removed.push(src);
  });
  apply(deletion, document([image("private-id")]), document(), [
    { docChanged: true, getMeta: (key) => key === "skipFileDeletion" },
  ]);
  assert.deepEqual(removed, []);
});

void test("removing duplicate nodes deletes their shared asset once and removes each node's metadata", () => {
  const { editor, changes } = fixture();
  const removed = [];
  const deletion = TrackFileDeletionPlugin(editor, async (src) => {
    removed.push(src);
  });
  apply(deletion, document([image("shared", "first"), image("shared", "second")]), document());
  assert.deepEqual(removed, ["shared"]);
  assert.deepEqual(changes, [{ idToRemove: "first" }, { idToRemove: "second" }]);
});

void test("legacy public image restoration deduplicates sources and handles failed restores", async (t) => {
  const { restorePublicImages } = await import("../../packages/editor/src/core/helpers/image-helpers.ts");
  const errors = t.mock.method(console, "error", () => {});
  const restored = [];
  const failure = new Error("offline");
  const state = document([
    image("https://assets.test/public"),
    image("https://assets.test/public"),
    image(42),
    image("private-id"),
  ]);
  restorePublicImages({ state }, async (src) => {
    restored.push(src);
    throw failure;
  });
  await Promise.resolve();
  assert.deepEqual(restored, ["https://assets.test/public"]);
  assert.equal(errors.mock.calls[0].arguments[1], failure);
});

import assert from "node:assert/strict";
import { fileURLToPath } from "node:url";
import { registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
import { seedWorkspaceQuery } from "./helpers/workspace-query-fixture.mjs";
registerWebModuleHooks();
const createFormSource = fileURLToPath(
  import.meta.resolve("../../apps/web/core/components/comments/comment-create.tsx")
);
const editFormSource = fileURLToPath(import.meta.resolve("../../apps/web/core/components/comments/card/edit-form.tsx"));
const formObserver = observeNativeForms([createFormSource, editFormSource]);
globalThis.webCommentReact = React;
const stubs = {
  "@/components/editor/lite-text": `const React = globalThis.webCommentReact;
    export const LiteTextEditor = React.forwardRef(function Editor({initialValue, onChange, onEnterKeyPress, onAssetUpload, isSubmitting, editable}, ref) {
      React.useImperativeHandle(ref, () => ({clearEditor() {}, focus() {}, isEditorReadyToDiscard: () => globalThis.webCommentReady}));
      return React.createElement(React.Fragment, null,
        React.createElement("input", {value: initialValue ?? "", disabled: !editable, onChange: event => onChange(null, event.target.value)}),
        React.createElement("button", {type:"button", onClick: () => onAssetUpload("asset")}, "Upload"),
        React.createElement("button", {type:"button", onClick: onEnterKeyPress, disabled: isSubmitting}, "Submit"));
    });`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const settle = () => act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
async function change(input, value) {
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
}
await test("web comments preserve failed drafts and retry attachments without duplicate posts", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const { CommentCreate } = await import("../../apps/web/core/components/comments/comment-create.tsx");
  const { FileService } = await import("../../apps/web/core/services/file.service.ts");
  const client = new QueryClient();
  await seedWorkspaceQuery(client);
  t.after(async () => {
    await environment.unmount(() => client.clear());
  });
  let failPost = true,
    failAssets = true;
  const posts = [],
    assets = [];
  globalThis.webCommentReady = true;
  t.mock.method(FileService.prototype, "updateBulkProjectAssetsUploadStatus", async (...args) => {
    assets.push(args);
    if (failAssets) throw new Error("offline");
  });
  const activityOperations = {
    isCurrent: () => true,
    isVerified: () => true,
    createComment: async (value) => {
      posts.push(value);
      if (failPost) throw new Error("offline");
      return { id: "comment" };
    },
  };
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(CommentCreate, {
        workspaceSlug: "workspace",
        projectId: "project",
        entityId: "issue",
        activityOperations,
      })
    )
  );
  const button = (label) =>
    [...environment.container.querySelectorAll("button")].find((item) => item.textContent === label);
  const input = environment.container.querySelector("input");
  await change(input, "<p>draft</p>");
  await act(async () => button("Upload").click());
  await act(async () => button("Submit").click());
  assert.equal(input.value, "<p>draft</p>");
  assert.equal(assets.length, 0);
  assert.equal(formObserver.getForms(createFormSource).at(-1)?.state.isSubmitSuccessful, false);
  failPost = false;
  await act(async () => button("Submit").click());
  await settle();
  assert.equal(input.value, "<p></p>");
  assert.equal(button("Submit").disabled, true);
  assert.equal(formObserver.getForms(createFormSource).at(-1)?.state.isSubmitSuccessful, true);
  failAssets = false;
  await act(async () => button("Retry attachments").click());
  await settle();
  assert.equal(posts.length, 2);
  assert.deepEqual(assets, [
    ["workspace", "project", "issue", { asset_ids: ["asset"] }],
    ["workspace", "project", "issue", { asset_ids: ["asset"] }],
  ]);
  assert.equal(button("Submit").disabled, false);
});
await test("comment editing waits for uploads and preserves failed edits until saving succeeds", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { CommentCardEditForm } = await import("../../apps/web/core/components/comments/card/edit-form.tsx");
  let fail = true,
    closed = 0;
  const updates = [];
  globalThis.webCommentReady = false;
  await environment.render(
    React.createElement(CommentCardEditForm, {
      comment: { id: "comment", comment_html: "Original" },
      readOnlyEditorRef: { current: null },
      workspaceId: "workspace",
      workspaceSlug: "workspace",
      setIsEditing: (value) => {
        if (!value) closed++;
      },
      activityOperations: {
        isCurrent: () => true,
        isVerified: () => true,
        updateComment: async (...args) => {
          updates.push(args);
          if (fail) throw new Error("offline");
        },
      },
    })
  );
  const input = environment.container.querySelector("input");
  await change(input, "Updated");
  const save = () => act(async () => environment.container.querySelector('[aria-label="Save comment"]').click());
  await save();
  assert.equal(updates.length, 0);
  globalThis.webCommentReady = true;
  await save();
  assert.equal(closed, 0);
  assert.equal(input.value, "Updated");
  assert.equal(formObserver.getForms(editFormSource).at(-1)?.state.isSubmitSuccessful, false);
  fail = false;
  await save();
  assert.equal(closed, 1);
  assert.deepEqual(updates, [
    ["comment", { comment_html: "Updated" }],
    ["comment", { comment_html: "Updated" }],
  ]);
});

formObserver.restore();

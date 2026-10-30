import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerSpaceModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { observeNativeForms } from "./helpers/native-form-observer.mjs";
registerSpaceModuleHooks();
const addCommentSource = "apps/space/components/issues/peek-overview/comment/add-comment.tsx";
const formObserver = observeNativeForms([addCommentSource]);
test.after(() => formObserver.restore());
const require = createRequire(new URL("../../apps/space/package.json", import.meta.url));
const { SitesIssueService, SitesFileService } = require("@plane/services");
globalThis.spaceCommentReact = React;
const toasts = [];
globalThis.spaceCommentToasts = toasts;
const stubs = {
  "@plane/propel/toast": `export const TOAST_TYPE = { ERROR: "error" }; export const setToast = value => globalThis.spaceCommentToasts.push(value);`,
  "@/components/editor/lite-text-editor": `const React = globalThis.spaceCommentReact;
    export const LiteTextEditor = React.forwardRef(function Editor({initialValue, onChange, onEnterKeyPress, uploadFile, isSubmitting}, ref) {
      React.useImperativeHandle(ref, () => ({clearEditor() {}}));
      return React.createElement(React.Fragment, null,
        React.createElement("input", {value: initialValue ?? "", onChange: event => onChange(null, event.target.value)}),
        React.createElement("button", {onClick: () => uploadFile("block", new File(["asset"], "test.txt"))}, "Upload"),
        React.createElement("button", {onClick: onEnterKeyPress, disabled: isSubmitting}, "Submit"));
    });`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { AddComment } = await import("../../apps/space/components/issues/peek-overview/comment/add-comment.tsx");
const { currentUserOptions } = await import("../../apps/space/queries/session.ts");
const { publishSettingsOptions } = await import("../../apps/space/queries/publishing.ts");
const settle = () => act(async () => new Promise((resolve) => setTimeout(resolve, 20)));
await test("comment failures retain the draft; failed attachment finalization retries without posting twice", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  t.after(async () => {
    await environment.unmount(() => client.clear());
  });
  client.setQueryData(currentUserOptions.queryKey, { id: "reader" });
  client.setQueryData(publishSettingsOptions("A").queryKey, { workspace: "workspace" });
  let failComment = true,
    failAttachments = true;
  const posts = [],
    attachments = [];
  t.mock.method(SitesIssueService.prototype, "addComment", async (anchor, issue, data) => {
    posts.push([anchor, issue, data.comment_html]);
    if (failComment) throw new Error("offline");
    return { id: "saved-comment", ...data };
  });
  t.mock.method(SitesFileService.prototype, "uploadAsset", async () => ({ asset_id: "asset" }));
  t.mock.method(SitesFileService.prototype, "updateBulkAssetsUploadStatus", async (...args) => {
    attachments.push(args);
    if (failAttachments) throw new Error("offline");
  });
  await environment.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(AddComment, { anchor: "A", issueId: "issue" })
    )
  );
  const input = environment.container.querySelector("input");
  await act(async () => {
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, "draft");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  const button = (label) =>
    [...environment.container.querySelectorAll("button")].find((item) => item.textContent === label);
  await act(async () => button("Upload").click());
  await act(async () => button("Submit").click());
  await settle();
  assert.equal(formObserver.getForms(addCommentSource).at(-1)?.state.isSubmitSuccessful, false);
  assert.equal(input.value, "draft");
  assert.equal(button("Submit").disabled, false, "a failed comment can be retried");
  assert.equal(attachments.length, 0);
  failComment = false;
  await act(async () => button("Submit").click());
  await settle();
  assert.equal(formObserver.getForms(addCommentSource).at(-1)?.state.isSubmitSuccessful, true);
  assert.equal(input.value, "");
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /comment was posted/);
  assert.equal(button("Submit").disabled, true);
  failAttachments = false;
  await act(async () => button("Retry attachments").click());
  await settle();
  assert.equal(posts.length, 2, "one failed post and one successful post; attachment retry does not post again");
  assert.deepEqual(attachments, [
    ["A", "saved-comment", { asset_ids: ["asset"] }],
    ["A", "saved-comment", { asset_ids: ["asset"] }],
  ]);
  assert.equal(button("Submit").disabled, false);
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
});

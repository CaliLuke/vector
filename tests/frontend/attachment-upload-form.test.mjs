import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { React, act } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { seedIssueQueryScope } from "./helpers/issue-query-fixtures.mjs";

registerWebModuleHooks();
const stubs = {
  "@/hooks/use-file-size": "export const useFileSize = () => ({ maxFileSize: 20 });",
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "@plane/propel/toast": "export const TOAST_TYPE = { ERROR: 'error' }; export const setToast = () => {};",
  "@/providers/issue-detail-ui-state":
    "export const useIssueDetailUiState = () => ({ setLastWidgetAction: () => {} });",
};
registerHooks({ resolve: (specifier, context, next) => resolvePackageStub(specifier, stubs, context, next) });
const { IssueAttachmentUpload } =
  await import("../../apps/web/core/components/issues/attachment/attachment-upload.tsx");

async function selectFile(view, file) {
  const input = view.container.querySelector("input[type=file]");
  Object.defineProperty(input, "files", { configurable: true, value: [file] });
  await act(async () => input.dispatchEvent(new window.Event("change", { bubbles: true })));
}

await test("failed attachment uploads retain the selected file for retry and release pending state", async (t) => {
  const view = await createReactDomTestEnvironment();
  t.after(() => view.unmount());
  const calls = [];
  const pending = Promise.withResolvers();
  await view.render(
    React.createElement(IssueAttachmentUpload, {
      workspaceSlug: "acme",
      attachmentOperations: {
        create: (file) => {
          calls.push(file);
          return calls.length === 1 ? pending.promise : Promise.resolve();
        },
      },
    })
  );
  const file = new window.File(["hello"], "notes.txt", { type: "text/plain" });
  await selectFile(view, file);
  assert.deepEqual(calls, [file]);
  assert.match(view.container.textContent, /Uploading/);
  await selectFile(view, new window.File(["second"], "other.txt", { type: "text/plain" }));
  assert.deepEqual(calls, [file], "a pending upload must reject another selection");
  await act(async () => pending.reject(new Error("offline")));
  const retry = [...view.container.querySelectorAll("button")].find((button) => button.textContent === "Retry upload");
  assert.ok(retry);
  assert.equal(retry.disabled, false);
  await act(async () => retry.click());
  assert.deepEqual(calls, [file, file]);
  assert.equal(view.container.querySelector("input").disabled, false);
  assert.equal(view.container.textContent.includes("Retry upload"), false);
});

await test("oversize attachment selections cannot submit or offer a retry for the invalid file", async (t) => {
  const view = await createReactDomTestEnvironment();
  t.after(() => view.unmount());
  const create = t.mock.fn(async () => {});
  await view.render(
    React.createElement(IssueAttachmentUpload, { workspaceSlug: "acme", attachmentOperations: { create } })
  );
  await selectFile(view, new window.File(["x".repeat(21)], "large.txt", { type: "text/plain" }));
  assert.equal(create.mock.callCount(), 0);
  assert.match(view.container.textContent, /attachment.file_size_limit/);
  assert.equal(view.container.textContent.includes("Retry upload"), false);
});

await test("the attachment quick action retries its retained file without reopening the file dialog", async (t) => {
  const view = await createReactDomTestEnvironment();
  t.after(() => view.unmount());
  const { IssueAttachmentActionButton } =
    await import("../../apps/web/core/components/issues/issue-detail-widgets/attachments/quick-action-button.tsx");
  const calls = [];
  const helpers = {
    uploadScopeKey: "account/workspace/first",
    operations: {
      create: async (file) => {
        calls.push(file);
        if (calls.length === 1) throw new Error("offline");
      },
    },
  };
  await view.render(
    React.createElement(IssueAttachmentActionButton, { workspaceSlug: "acme", attachmentHelpers: helpers })
  );
  const file = new window.File(["hello"], "notes.txt");
  await selectFile(view, file);
  const input = view.container.querySelector("input");
  const click = t.mock.method(input, "click");
  const retry = [...view.container.querySelectorAll("button")].find((button) => button.textContent === "Retry upload");
  assert.ok(retry);
  await act(async () => retry.click());
  assert.deepEqual(calls, [file, file]);
  assert.equal(click.mock.callCount(), 0);
  assert.equal(view.container.querySelector('[role="alert"]'), null);
});

await test("changing attachment owner discards its draft and ignores late upload feedback", async (t) => {
  const view = await createReactDomTestEnvironment();
  t.after(() => view.unmount());
  const { IssueAttachmentActionButton } =
    await import("../../apps/web/core/components/issues/issue-detail-widgets/attachments/quick-action-button.tsx");
  const pending = Promise.withResolvers();
  const render = (uploadScopeKey, create) =>
    view.render(
      React.createElement(IssueAttachmentActionButton, {
        workspaceSlug: "acme",
        attachmentHelpers: { uploadScopeKey, operations: { create } },
      })
    );
  await render("account/workspace/first", () => pending.promise);
  await selectFile(view, new window.File(["first"], "first.txt"));
  const create = t.mock.fn(async () => {});
  await render("account/workspace/second", create);
  await act(async () => pending.reject(new Error("old upload failed")));
  assert.equal(view.container.querySelector('[role="alert"]'), null);
  const secondFile = new window.File(["second"], "second.txt");
  await selectFile(view, secondFile);
  assert.equal(create.mock.callCount(), 1);
  assert.equal(create.mock.calls[0].arguments[0], secondFile);
});

await test("native attachment progress remains visible until the pending upload settles", async (t) => {
  const view = await createQueryRouterFixture(t);
  seedIssueQueryScope(view.client, { projectId: "p1" });
  const { IssueAttachmentService } = await import("../../apps/web/core/services/issue/issue_attachment.service.ts");
  const { useIssueAttachments } = await import("../../apps/web/core/hooks/use-issue-resources.ts");
  t.mock.method(IssueAttachmentService.prototype, "getIssueAttachments", async () => []);
  const pending = Promise.withResolvers();
  t.mock.method(
    IssueAttachmentService.prototype,
    "uploadIssueAttachment",
    (_workspace, _project, _issue, _file, progress) => {
      progress({ progress: 0.5 });
      return pending.promise;
    }
  );
  let resources;
  function Harness() {
    const current = useIssueAttachments({ workspaceSlug: "acme", projectId: "p1", issueId: "i1" });
    React.useLayoutEffect(() => {
      resources = current;
    });
    return null;
  }
  await view.render(React.createElement(Harness));
  assert.equal(resources.isReady, true);
  let rejected;
  await act(async () => {
    rejected = assert.rejects(resources.create(new window.File(["hello"], "notes.txt")), /offline/);
    await new Promise(setImmediate);
  });
  assert.equal(resources.uploadStatus.length, 1);
  assert.equal(resources.uploadStatus[0].progress, 50);
  await act(async () => {
    pending.reject(new Error("offline"));
    await rejected;
  });
  assert.deepEqual(resources.uploadStatus, []);
});

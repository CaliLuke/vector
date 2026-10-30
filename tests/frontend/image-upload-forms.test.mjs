import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { pathToFileURL } from "node:url";
import { settleQuery } from "./helpers/query-router.mjs";

registerWebModuleHooks();
const reactUrl = pathToFileURL(webRequire.resolve("react")).href;
const ui = `import React from ${JSON.stringify(reactUrl)};
export const EModalPosition = { CENTER: 'center' };
export const EModalWidth = { XL: 'xl' };
export function ModalCore({children, handleClose}) { return React.createElement('section', {role: 'dialog'}, children, React.createElement('button', {onClick: handleClose}, 'Dismiss')); }`;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/ui") return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(ui)}` };
    return nextResolve(specifier, context);
  },
});
const { UserImageUploadModal } = await import("../../apps/web/core/components/core/modals/user-image-upload-modal.tsx");
const { WorkspaceImageUploadModal } =
  await import("../../apps/web/core/components/core/modals/workspace-image-upload-modal.tsx");
const { FileService } = await import("../../apps/web/core/services/file.service.ts");

async function fixture(t, props = {}, Component = UserImageUploadModal) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { mutations: { retry: 3, gcTime: Infinity } } });
  const onClose = t.mock.fn();
  const onSuccess = t.mock.fn();
  const handleRemove = t.mock.fn();
  const urls = [];
  const revoked = [];
  t.mock.method(URL, "createObjectURL", () => {
    const url = `blob:preview-${urls.length}`;
    urls.push(url);
    return url;
  });
  t.mock.method(URL, "revokeObjectURL", (url) => revoked.push(url));
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  const render = (extra = {}) =>
    env.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(Component, {
          isOpen: true,
          value: null,
          onClose,
          onSuccess,
          handleRemove,
          ...props,
          ...extra,
        })
      )
    );
  await render();
  const button = (text) => [...env.container.querySelectorAll("button")].find((b) => b.textContent === text);
  return {
    env,
    client,
    onClose,
    onSuccess,
    handleRemove,
    urls,
    revoked,
    render,
    button,
    async select(file) {
      await act(async () => {
        const input = env.container.querySelector('input[type="file"]');
        Object.defineProperty(input, "files", { configurable: true, value: [file] });
        input.dispatchEvent(new Event("change", { bubbles: true }));
        await Promise.resolve();
      });
    },
    submit: () =>
      act(async () => {
        env.container.querySelector("form").dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
      }),
  };
}
const image = () => new File(["png"], "avatar.png", { type: "image/png" });

await test("image submission waits for its owner and cannot close or submit twice while pending", async (t) => {
  const upload = Promise.withResolvers();
  const saved = Promise.withResolvers();
  const uploadAsset = t.mock.method(FileService.prototype, "uploadUserAsset", () => upload.promise);
  const owner = t.mock.fn(() => saved.promise);
  const f = await fixture(t, { onSuccess: owner });
  await f.select(image());
  await f.submit();
  await f.submit();
  await act(async () => f.button("Dismiss").click());
  assert.equal(uploadAsset.mock.callCount(), 1);
  assert.equal(f.onClose.mock.callCount(), 0);
  assert.equal(f.button("Cancel").disabled, true);
  await act(async () => upload.resolve({ asset_url: "/asset/avatar/" }));
  assert.equal(owner.mock.callCount(), 1);
  assert.equal(f.onClose.mock.callCount(), 0);
  await act(async () => saved.resolve());
  assert.equal(f.onClose.mock.callCount(), 1);
});

await test("owner failure retains the chosen image and retries without uploading again", async (t) => {
  const upload = t.mock.method(FileService.prototype, "uploadUserAsset", async () => ({ asset_url: "/asset/avatar/" }));
  const owner = t.mock.fn(async () => {});
  owner.mock.mockImplementationOnce(async () => {
    throw new Error("owner unavailable");
  });
  const f = await fixture(t, { onSuccess: owner });
  await f.select(image());
  await f.submit();
  assert.match(f.env.container.querySelector('[role="alert"]').textContent, /try again/i);
  assert.equal(f.onClose.mock.callCount(), 0);
  assert.equal(f.env.container.querySelector("img").src, "blob:preview-0");
  await f.submit();
  assert.equal(upload.mock.callCount(), 1);
  assert.equal(owner.mock.callCount(), 2);
  assert.equal(f.onClose.mock.callCount(), 1);
});

await test("rejected files keep the previous valid choice and preview URLs are revoked", async (t) => {
  const upload = t.mock.method(FileService.prototype, "uploadUserAsset", async () => ({ asset_url: "/asset/avatar/" }));
  const f = await fixture(t);
  const valid = image();
  await f.select(valid);
  await f.select(new File(["bad"], "bad.txt", { type: "text/plain" }));
  assert.match(f.env.container.querySelector('[role="alert"]').textContent, /valid format/i);
  assert.equal(f.env.container.querySelector("img").src, "blob:preview-0");
  await f.submit();
  assert.equal(upload.mock.calls[0].arguments[1], valid);
  await f.select(image());
  assert.deepEqual(f.revoked, ["blob:preview-0"]);
  await f.render({ isOpen: false });
  assert.deepEqual(f.revoked, ["blob:preview-0", "blob:preview-1"]);
  await f.render();
  assert.equal(f.button("Upload & Save").disabled, true);
});

await test("failed deletion remains visible and retries without repeating a completed server deletion", async (t) => {
  const deletion = t.mock.method(FileService.prototype, "deleteUserAsset", async () => {});
  const owner = t.mock.fn(async () => {});
  owner.mock.mockImplementationOnce(async () => {
    throw new Error("refresh failed");
  });
  const f = await fixture(t, { value: "/api/assets/v2/user-assets/old/", handleRemove: owner });
  await act(async () => f.button("Remove").click());
  assert.equal(f.onClose.mock.callCount(), 0);
  assert.match(f.env.container.querySelector('[role="alert"]').textContent, /try again/i);
  await act(async () => f.button("Remove").click());
  assert.equal(deletion.mock.callCount(), 1);
  assert.equal(owner.mock.callCount(), 2);
  assert.equal(f.onClose.mock.callCount(), 1);
});

await test("workspace logo upload is scoped explicitly and unmount cancels without publishing late results", async (t) => {
  const pending = Promise.withResolvers();
  const upload = t.mock.method(FileService.prototype, "uploadWorkspaceAsset", () => pending.promise);
  const f = await fixture(t, { workspaceId: "workspace", workspaceSlug: "acme" }, WorkspaceImageUploadModal);
  await f.select(image());
  await f.submit();
  const args = upload.mock.calls[0].arguments;
  assert.equal(args[0], "acme");
  assert.deepEqual(args[1], { entity_identifier: "workspace", entity_type: "WORKSPACE_LOGO" });
  await f.render({ isOpen: false });
  assert.equal(args[4].aborted, true);
  await act(async () => pending.resolve({ asset_url: "/asset/late/" }));
  await settleQuery();
  assert.equal(f.onSuccess.mock.callCount(), 0);
  assert.equal(f.onClose.mock.callCount(), 0);
});

await test("upload failures preserve the file and do not automatically retry creating an asset", async (t) => {
  const upload = t.mock.method(FileService.prototype, "uploadUserAsset", async () => ({
    asset_url: "/asset/retried/",
  }));
  upload.mock.mockImplementationOnce(async () => {
    throw new Error("offline");
  });
  const f = await fixture(t);
  await f.select(image());
  await f.submit();
  assert.equal(upload.mock.callCount(), 1);
  assert.equal(f.onSuccess.mock.callCount(), 0);
  assert.equal(f.onClose.mock.callCount(), 0);
  assert.equal(f.button("Upload & Save").disabled, false);
  await f.submit();
  assert.equal(upload.mock.callCount(), 2);
  assert.equal(f.onSuccess.mock.callCount(), 1);
});

await test("server deletion failure does not clear the owner and permits an explicit retry", async (t) => {
  const deletion = t.mock.method(FileService.prototype, "deleteUserAsset", async () => {});
  deletion.mock.mockImplementationOnce(async () => {
    throw new Error("offline");
  });
  const f = await fixture(t, { value: "/api/assets/v2/user-assets/old/" });
  await act(async () => f.button("Remove").click());
  assert.equal(deletion.mock.callCount(), 1);
  assert.equal(f.handleRemove.mock.callCount(), 0);
  assert.equal(f.onClose.mock.callCount(), 0);
  await act(async () => f.button("Remove").click());
  assert.equal(deletion.mock.callCount(), 2);
  assert.equal(f.handleRemove.mock.callCount(), 1);
});

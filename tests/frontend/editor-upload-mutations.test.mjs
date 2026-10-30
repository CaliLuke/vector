import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedInstanceInfo } from "./helpers/instance-info.mjs";
import { settleQuery } from "./helpers/query-router.mjs";

registerWebModuleHooks();
const { useEditorConfig } = await import("../../apps/web/core/hooks/editor/use-editor-config.ts");
const { FileService } = await import("../../apps/web/core/services/file.service.ts");
const uploadFile = new File(["upload"], "asset.txt", { type: "text/plain" });
const asset = { entity_identifier: "issue", entity_type: "ISSUE_DESCRIPTION" };

async function fixture(t) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { mutations: { retry: 3 } } });
  seedInstanceInfo(client, { file_size_limit: 1000 });
  const handlers = {};
  const completed = [];
  const uploads = [];
  const duplicates = [];
  t.mock.method(
    FileService.prototype,
    "uploadProjectAsset",
    (workspaceSlug, projectId, data, file, progress, signal) => {
      const pending = Promise.withResolvers();
      uploads.push({ workspaceSlug, projectId, data, file, progress, signal, ...pending });
      return pending.promise;
    }
  );
  t.mock.method(FileService.prototype, "uploadWorkspaceAsset", (workspaceSlug, data, file, progress, signal) => {
    const pending = Promise.withResolvers();
    uploads.push({ workspaceSlug, data, file, progress, signal, ...pending });
    return pending.promise;
  });
  t.mock.method(FileService.prototype, "duplicateAsset", (workspaceSlug, assetId, data, signal) => {
    const pending = Promise.withResolvers();
    duplicates.push({ workspaceSlug, assetId, data, signal, ...pending });
    return pending.promise;
  });
  function Editor({ id, ...props }) {
    const handler = useEditorConfig({
      workspaceId: "workspace",
      workspaceSlug: "acme",
      projectId: "project",
      fileAsset: asset,
      onAssetUpload: (assetId) => completed.push({ editor: id, assetId }),
      ...props,
    });
    React.useLayoutEffect(() => {
      handlers[id] = handler;
    }, [id, handler]);
    return React.createElement("output", { "data-editor": id }, JSON.stringify(handler.assetsUploadStatus));
  }
  t.after(async () => {
    await env.unmount(() => client.clear());
  });
  return {
    env,
    client,
    handlers,
    completed,
    uploads,
    duplicates,
    render: (editors = [{ id: "one" }]) =>
      env.render(
        React.createElement(
          QueryClientProvider,
          { client },
          editors.map((props) => React.createElement(Editor, { key: props.id, ...props }))
        )
      ),
    progress: (id = "one") => JSON.parse(env.container.querySelector(`[data-editor="${id}"]`).textContent),
    async upload(id, blockId) {
      /** @type {Promise<string>} */
      let promise;
      await act(async () => {
        promise = handlers[id].upload(blockId, uploadFile);
      });
      return { promise };
    },
  };
}

await test("concurrent blocks retain independent progress and ignore late progress after completion", async (t) => {
  const f = await fixture(t);
  await f.render();
  const a = await f.upload("one", "a");
  const b = await f.upload("one", "b");
  await act(async () => {
    f.uploads[0].progress({ progress: 0.25 });
    f.uploads[1].progress({ progress: 0.6 });
  });
  assert.deepEqual(f.progress(), { a: 25, b: 60 });
  await act(async () => {
    f.uploads[0].resolve({ asset_id: "asset-a" });
    await a.promise;
  });
  await act(async () => f.uploads[0].progress({ progress: 1 }));
  assert.deepEqual(f.progress(), { b: 60 });
  await act(async () => {
    f.uploads[1].resolve({ asset_id: "asset-b" });
    await b.promise;
  });
  assert.deepEqual(f.progress(), {});
  assert.deepEqual(f.completed, [
    { editor: "one", assetId: "asset-a" },
    { editor: "one", assetId: "asset-b" },
  ]);
});

await test("canceling one editor cannot cancel another editor with the same block ID", async (t) => {
  const f = await fixture(t);
  await f.render([{ id: "one" }, { id: "two", workspaceSlug: "other", projectId: undefined }]);
  const a = await f.upload("one", "block");
  const b = await f.upload("two", "block");
  const rejected = assert.rejects(a.promise, { name: "AbortError" });
  await act(async () => f.handlers.one.cancel());
  assert.equal(f.uploads[0].signal.aborted, true);
  assert.equal(f.uploads[1].signal.aborted, false);
  assert.equal(f.uploads[1].workspaceSlug, "other");
  assert.equal(f.uploads[1].projectId, undefined);
  await act(async () => {
    f.uploads[0].resolve({ asset_id: "canceled" });
    await rejected;
  });
  await act(async () => {
    f.uploads[1].resolve({ asset_id: "retained" });
    await b.promise;
  });
  assert.deepEqual(f.completed, [{ editor: "two", assetId: "retained" }]);
});

await test("a superseded upload cannot clear the replacement block's progress", async (t) => {
  const f = await fixture(t);
  await f.render();
  const old = await f.upload("one", "block");
  const rejected = assert.rejects(old.promise, { name: "AbortError" });
  const next = await f.upload("one", "block");
  assert.equal(f.uploads[0].signal.aborted, true);
  await act(async () => {
    f.uploads[1].progress({ progress: 0.4 });
    f.uploads[0].resolve({ asset_id: "old" });
    await rejected;
  });
  assert.deepEqual(f.progress(), { block: 40 });
  await act(async () => {
    f.uploads[1].resolve({ asset_id: "new" });
    await next.promise;
  });
  assert.deepEqual(f.completed, [{ editor: "one", assetId: "new" }]);
});

await test("failed uploads reject, clear progress, and allow an explicit retry without automatic duplicate requests", async (t) => {
  const f = await fixture(t);
  await f.render();
  const attempt = await f.upload("one", "block");
  const failure = new Error("offline");
  const rejected = assert.rejects(attempt.promise, (error) => error === failure);
  await act(async () => {
    f.uploads[0].reject(failure);
    await rejected;
  });
  await settleQuery();
  assert.deepEqual(f.progress(), {});
  assert.deepEqual(f.completed, []);
  assert.equal(f.uploads.length, 1);
  assert.equal(f.client.getMutationCache().getAll()[0].state.status, "error");
  const retry = await f.upload("one", "block");
  await act(async () => {
    f.uploads[1].resolve({ asset_id: "retry" });
    await retry.promise;
  });
  assert.deepEqual(f.completed, [{ editor: "one", assetId: "retry" }]);
});

await test("switching entity scope aborts old work and prevents it attaching to the new entity", async (t) => {
  const f = await fixture(t);
  await f.render();
  const old = await f.upload("one", "block");
  const rejected = assert.rejects(old.promise, { name: "AbortError" });
  await f.render([
    { id: "one", workspaceSlug: "new-workspace", fileAsset: { ...asset, entity_identifier: "new-issue" } },
  ]);
  assert.equal(f.uploads[0].signal.aborted, true);
  assert.deepEqual(f.progress(), {});
  const next = await f.upload("one", "block");
  assert.equal(f.uploads[1].workspaceSlug, "new-workspace");
  assert.equal(f.uploads[1].data.entity_identifier, "new-issue");
  await act(async () => {
    f.uploads[0].resolve({ asset_id: "obsolete" });
    await rejected;
  });
  assert.deepEqual(f.progress(), { block: 0 });
  await act(async () => {
    f.uploads[1].resolve({ asset_id: "current" });
    await next.promise;
  });
  assert.deepEqual(f.completed, [{ editor: "one", assetId: "current" }]);
});

await test("duplicates use the editor entity scope and abort on unmount without reporting success", async (t) => {
  const f = await fixture(t);
  await f.render();
  /** @type {Promise<string>} */
  let duplicate;
  await act(async () => {
    duplicate = f.handlers.one.duplicate("source");
  });
  assert.deepEqual(f.duplicates[0].data, {
    entity_id: "issue",
    entity_type: "ISSUE_DESCRIPTION",
    project_id: "project",
  });
  const rejected = assert.rejects(duplicate, { name: "AbortError" });
  await f.render([]);
  assert.equal(f.duplicates[0].signal.aborted, true);
  await act(async () => {
    f.duplicates[0].resolve({ asset_id: "late" });
    await rejected;
  });
  assert.deepEqual(f.completed, []);
});

await test("successful duplication reports the new asset for attachment finalization", async (t) => {
  const f = await fixture(t);
  await f.render();
  /** @type {Promise<string>} */
  let duplicate;
  await act(async () => {
    duplicate = f.handlers.one.duplicate("source");
  });
  await act(async () => {
    f.duplicates[0].resolve({ asset_id: "copy" });
    assert.equal(await duplicate, "copy");
  });
  assert.deepEqual(f.completed, [{ editor: "one", assetId: "copy" }]);
});

await test("an editor without an asset context rejects upload and duplication without network effects", async (t) => {
  const f = await fixture(t);
  await f.render([{ id: "one", fileAsset: undefined }]);
  await assert.rejects(f.handlers.one.upload("block", uploadFile), /unavailable/);
  await assert.rejects(f.handlers.one.duplicate("source"), /unavailable/);
  assert.equal(f.uploads.length, 0);
  assert.equal(f.duplicates.length, 0);
});

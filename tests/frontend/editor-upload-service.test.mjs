import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { APIService } = await import("../../apps/web/core/services/api.service.ts");
const { FileService } = await import("../../apps/web/core/services/file.service.ts");
const { FileUploadService } = await import("../../apps/web/core/services/file-upload.service.ts");
const file = new File(["upload"], "asset.txt", { type: "text/plain" });
const data = { entity_identifier: "issue", entity_type: "ISSUE_DESCRIPTION" };
const signedAsset = {
  asset_id: "asset",
  asset_url: "url",
  upload_data: { url: "https://storage.invalid/upload", fields: { key: "key" } },
};

await test("project uploads pass one abort signal through signing, storage and finalization in order", async (t) => {
  const writes = [];
  const controller = new AbortController();
  const progress = t.mock.fn();
  t.mock.method(APIService.prototype, "post", async (url, body, config) => {
    writes.push({ url, body, config });
    return { data: signedAsset };
  });
  t.mock.method(APIService.prototype, "patch", async (url, body, config) => {
    writes.push({ url, body, config });
    return { data: {} };
  });
  const result = await new FileService().uploadProjectAsset("acme", "project", data, file, progress, controller.signal);
  assert.equal(result.asset_id, "asset");
  assert.deepEqual(
    writes.map((write) => write.url),
    [
      "/api/assets/v2/workspaces/acme/projects/project/",
      "https://storage.invalid/upload",
      "/api/assets/v2/workspaces/acme/projects/project/asset/",
    ]
  );
  assert.equal(writes[0].body.entity_identifier, "issue");
  assert.equal(writes[0].config.signal, controller.signal);
  assert.equal(writes[1].body.get("file"), file);
  assert.equal(writes[1].config.signal, controller.signal);
  assert.equal(writes[1].config.onUploadProgress, progress);
  assert.equal(writes[1].config.withCredentials, false);
  assert.equal(writes[2].config.signal, controller.signal);
});

await test("a rejected storage upload propagates its error and never finalizes the asset", async (t) => {
  const failure = new DOMException("Canceled", "AbortError");
  t.mock.method(APIService.prototype, "post", async (url) => {
    if (url === signedAsset.upload_data.url) throw failure;
    return { data: signedAsset };
  });
  const finalize = t.mock.method(APIService.prototype, "patch", async () => ({ data: {} }));
  await assert.rejects(new FileService().uploadWorkspaceAsset("acme", data, file), (error) => error === failure);
  assert.equal(finalize.mock.callCount(), 0);
});

await test("cancellation after storage completion still prevents asset finalization", async (t) => {
  const controller = new AbortController();
  t.mock.method(APIService.prototype, "post", async (url) => {
    if (url === signedAsset.upload_data.url) controller.abort();
    return { data: signedAsset };
  });
  const finalize = t.mock.method(APIService.prototype, "patch", async () => ({ data: {} }));
  await assert.rejects(new FileService().uploadWorkspaceAsset("acme", data, file, undefined, controller.signal), {
    name: "AbortError",
  });
  assert.equal(finalize.mock.callCount(), 0);
});

await test("an already canceled upload never creates an asset or posts storage bytes", async (t) => {
  const controller = new AbortController();
  controller.abort();
  const post = t.mock.method(APIService.prototype, "post", async () => ({ data: signedAsset }));
  await assert.rejects(new FileService().uploadWorkspaceAsset("acme", data, file, undefined, controller.signal), {
    name: "AbortError",
  });
  await assert.rejects(
    new FileUploadService().uploadFile("https://storage.invalid/upload", new FormData(), undefined, controller.signal),
    { name: "AbortError" }
  );
  assert.equal(post.mock.callCount(), 0);
});

await test("user uploads use the same cancellable signing, storage and finalization sequence", async (t) => {
  const writes = [];
  const controller = new AbortController();
  t.mock.method(APIService.prototype, "post", async (url, body, config) => {
    writes.push({ url, signal: config.signal });
    return { data: signedAsset };
  });
  t.mock.method(APIService.prototype, "patch", async (url, body, config) => {
    writes.push({ url, signal: config.signal });
    return { data: {} };
  });
  const result = await new FileService().uploadUserAsset(
    { entity_identifier: "", entity_type: "USER_AVATAR" },
    file,
    controller.signal
  );
  assert.equal(result.asset_url, signedAsset.asset_url);
  assert.deepEqual(writes, [
    { url: "/api/assets/v2/user-assets/", signal: controller.signal },
    { url: signedAsset.upload_data.url, signal: controller.signal },
    { url: "/api/assets/v2/user-assets/asset/", signal: controller.signal },
  ]);
});

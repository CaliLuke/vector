import * as ContractEnums from "../../packages/types/dist/index.mjs";
import assert from "node:assert/strict";
import test from "node:test";
import { webRequire } from "./helpers/react.mjs";
const { AxiosError } = webRequire("axios");
import { APIService, SitesFileService, FileUploadService } from "../../packages/services/dist/index.js";

const file = new File(["upload"], "asset.txt", { type: "text/plain" });
const data = { entity_identifier: "issue", entity_type: ContractEnums.EFileAssetType.ISSUE_DESCRIPTION };
const signedAsset = {
  asset_id: "asset",
  asset_url: "url",
  upload_data: { url: "https://storage.invalid/upload", fields: { key: "key" } },
};

void test("public-site uploads finalize only after storage succeeds and preserve the signed result", async (t) => {
  const service = new SitesFileService();
  const writes = [];
  t.mock.method(APIService.prototype, "post", async (url, body, config) => {
    writes.push({ url, body, config });
    return { data: signedAsset };
  });
  t.mock.method(APIService.prototype, "patch", async (url, body, config) => {
    writes.push({ url, body, config });
    return { data: {} };
  });
  const result = await service.uploadAsset("anchor", data, file);
  assert.deepEqual(result, signedAsset);
  assert.deepEqual(
    writes.map(({ url }) => url),
    ["/api/public/assets/v2/anchor/anchor/", signedAsset.upload_data.url, "/api/public/assets/v2/anchor/anchor/asset/"]
  );
  assert.equal(writes[1].body.get("file"), file);
  assert.equal(writes[1].config.withCredentials, false);
  assert.equal(writes[0].config.signal, writes[2].config.signal);
  service.cancelUpload();
  assert.equal(writes[0].config.signal.aborted, false);
});

for (const stage of ["metadata", "signing", "storage"]) {
  void test(`canceling a public-site upload during ${stage} prevents later writes`, async (t) => {
    const service = new SitesFileService();
    const writes = [];
    t.mock.method(APIService.prototype, "post", async (url) => {
      writes.push(url);
      if (
        (stage === "signing" && url !== signedAsset.upload_data.url) ||
        (stage === "storage" && url === signedAsset.upload_data.url)
      )
        service.cancelUpload();
      return { data: signedAsset };
    });
    const finalize = t.mock.method(APIService.prototype, "patch", async () => ({ data: {} }));
    const upload = service.uploadAsset("anchor", data, file);
    if (stage === "metadata") service.cancelUpload();
    await assert.rejects(upload, { name: "AbortError" });
    assert.equal(finalize.mock.callCount(), 0);
    assert.equal(writes.length, stage === "metadata" ? 0 : stage === "signing" ? 1 : 2);
  });
}

void test("public-site upload failures retain the original error and never finalize", async (t) => {
  const failure = new Error("Storage unavailable");
  t.mock.method(APIService.prototype, "post", async (url) => {
    if (url === signedAsset.upload_data.url) throw failure;
    return { data: signedAsset };
  });
  const finalize = t.mock.method(APIService.prototype, "patch", async () => ({ data: {} }));
  await assert.rejects(new SitesFileService().uploadAsset("anchor", data, file), (error) => error === failure);
  assert.equal(finalize.mock.callCount(), 0);
});

void test("storage cancellation rejects every concurrent upload and permits a subsequent upload", async (t) => {
  const service = new FileUploadService();
  service.cancelUpload();
  const signals = [];
  t.mock.method(APIService.prototype, "post", (_url, _body, { signal }) => {
    signals.push(signal);
    return new Promise((_resolve, reject) =>
      signal.addEventListener("abort", () => reject(signal.reason), { once: true })
    );
  });
  const uploads = Promise.allSettled([
    service.uploadFile(signedAsset.upload_data.url, new FormData()),
    service.uploadFile(signedAsset.upload_data.url, new FormData()),
  ]);
  service.cancelUpload();
  const results = await uploads;
  assert.deepEqual(
    results.map((result) => [result.status, result.reason.name]),
    [
      ["rejected", "AbortError"],
      ["rejected", "AbortError"],
    ]
  );
  assert.equal(
    signals.every((signal) => signal.aborted),
    true
  );
  t.mock.method(APIService.prototype, "post", async () => ({ data: {} }));
  await service.uploadFile(signedAsset.upload_data.url, new FormData());
});

for (const kind of ["server response", "transport", "unexpected"]) {
  void test(`public-site finalization preserves the ${kind} failure for its caller`, async (t) => {
    const payload = { detail: "Finalization rejected" };
    const failure = kind === "unexpected" ? new Error("Unexpected failure") : new AxiosError("Request failed");
    if (kind === "server response") failure.response = { data: payload };
    t.mock.method(APIService.prototype, "post", async () => ({ data: signedAsset }));
    t.mock.method(APIService.prototype, "patch", async () => {
      throw failure;
    });
    const service = new SitesFileService("https://public.example.test");
    await assert.rejects(
      service.uploadAsset("anchor", data, file),
      (reason) => reason === (kind === "server response" ? payload : failure)
    );
  });
}

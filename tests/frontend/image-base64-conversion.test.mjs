import assert from "node:assert/strict";
import test from "node:test";
import { JSDOM } from "jsdom";
import { getBase64Image } from "../../packages/utils/src/file.ts";

void test("image conversion reads the fetched blob as a complete data URL", async (t) => {
  const dom = new JSDOM();
  const original = Object.getOwnPropertyDescriptor(globalThis, "FileReader");
  Object.defineProperty(globalThis, "FileReader", { value: dom.window.FileReader, configurable: true });
  t.after(() => {
    if (original) Object.defineProperty(globalThis, "FileReader", original);
    else delete globalThis.FileReader;
    dom.window.close();
  });
  t.mock.method(globalThis, "fetch", async () => ({
    ok: true,
    blob: async () => new dom.window.Blob(["image"], { type: "image/png" }),
  }));
  assert.equal(await getBase64Image("https://images.invalid/image.png"), "data:image/png;base64,aW1hZ2U=");
});

void test("image conversion rejects invalid URLs before fetching and propagates HTTP failure", async (t) => {
  const fetch = t.mock.method(globalThis, "fetch", async () => ({ ok: false, statusText: "Not Found" }));
  await assert.rejects(getBase64Image(""), /Invalid URL/);
  await assert.rejects(getBase64Image("invalid"), /Invalid URL/);
  assert.equal(fetch.mock.callCount(), 0);
  await assert.rejects(getBase64Image("https://images.invalid/missing"), /Failed to fetch image/);
});

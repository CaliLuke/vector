import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const requireFromWeb = createRequire(new URL("../../apps/web/package.json", import.meta.url));
const { default: axios } = await import(
  new URL("./index.js", pathToFileURL(requireFromWeb.resolve("axios/package.json"))).href
);

await test("item schemas send the mode and scope as query parameters, including an explicit type clear", async (t) => {
  const requests = [];
  const originalAdapter = axios.defaults.adapter;
  axios.defaults.adapter = async (config) => {
    requests.push({ url: axios.getUri(config), signal: config.signal });
    return { data: { properties: [] }, status: 200, statusText: "OK", headers: {}, config };
  };
  t.after(() => {
    axios.defaults.adapter = originalAdapter;
  });
  // Import after installing the adapter: the real service creates its Axios client here.
  const { workItemSchemaService } = await import("../../apps/web/core/services/work-item-schema.service.ts");
  const controller = new AbortController();

  const schema = await workItemSchemaService.schema(
    "acme",
    { mode: "publish", project_id: "destination", draft_id: "draft", type_id: "", parent_id: "" },
    controller.signal
  );

  const url = new URL(requests[0].url, "http://localhost");
  assert.equal(url.pathname, "/api/workspaces/acme/work-item-schema/");
  assert.deepEqual(Object.fromEntries(url.searchParams), {
    mode: "publish",
    project_id: "destination",
    draft_id: "draft",
    type_id: "",
    parent_id: "",
  });
  assert.equal(requests[0].signal, controller.signal);
  assert.deepEqual(schema, { properties: [] });

  await workItemSchemaService.schema("acme", { mode: "draft", draft_id: "draft" });
  assert.equal(new URL(requests[1].url, "http://localhost").searchParams.has("type_id"), false);
  assert.equal(new URL(requests[1].url, "http://localhost").searchParams.has("parent_id"), false);
});

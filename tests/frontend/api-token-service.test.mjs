import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import { readFileSync } from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath, pathToFileURL } from "node:url";
import { transformFrontendSource } from "./helpers/source-transform.mjs";
import { createModuleStub } from "./helpers/module-stubs.mjs";

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const servicePath = path.join(repositoryRoot, "packages/services/src/developer/api-token.service.ts");
const serviceUrl = pathToFileURL(servicePath).href;
const baseServiceUrl = new URL("../api.service.ts", serviceUrl).href;

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (context.parentURL === serviceUrl && specifier === "../api.service") return nextResolve(baseServiceUrl, context);
    if (context.parentURL === serviceUrl && specifier === "@plane/constants")
      return createModuleStub(`export const API_BASE_URL = "http://example.test";`);
    return nextResolve(specifier, context);
  },
  load(url, context, nextLoad) {
    if (url === serviceUrl || url === baseServiceUrl) {
      const filename = fileURLToPath(url);
      const source = readFileSync(filename, "utf8");
      return {
        format: "module",
        shortCircuit: true,
        source: transformFrontendSource(source, {
          format: "esm",
          target: "es2022",
          tsconfigRaw: { compilerOptions: { useDefineForClassFields: true } },
          loader: "ts",
          sourcefile: filename,
        }),
      };
    }
    return nextLoad(url, context);
  },
});

const { APITokenService } = await import(serviceUrl);

await test("API token list passes its AbortSignal to the HTTP request", async () => {
  const service = new APITokenService();
  const signal = new AbortController().signal;
  let request;
  service.get = async (...args) => {
    request = args;
    return { data: [] };
  };

  assert.deepEqual(await service.list(signal), []);
  assert.deepEqual(request, ["/api/users/api-tokens/", {}, { signal }]);
});

await test("API token service preserves network errors and surfaces response data", async () => {
  const service = new APITokenService();
  const networkError = new Error("offline");
  service.get = async () => {
    throw networkError;
  };
  await assert.rejects(service.list(), (error) => error === networkError);

  const responseError = { detail: "permission denied" };
  service.post = async () => {
    throw { response: { data: responseError } };
  };
  await assert.rejects(service.create({ label: "token" }), (error) => error === responseError);
});

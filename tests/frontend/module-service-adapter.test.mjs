import assert from "node:assert/strict";
import { existsSync, readFileSync, statSync } from "node:fs";
import { registerHooks } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { transformFrontendSource } from "./helpers/source-transform.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const servicesRoot = path.join(repositoryRoot, "packages/services");
const webRequire = (await import("node:module")).createRequire(path.join(repositoryRoot, "apps/web/package.json"));
const constantsUrl = pathToFileURL(webRequire.resolve("@plane/constants")).href;
const axiosStub = `
  export const isAxiosError = error => typeof error === "object" && error !== null && error.isAxiosError === true;
  const axios = {
    CancelToken: { source: () => ({ token: {}, cancel: () => {} }) },
    isCancel: () => false,
  };
  export const create = (defaults) => {
    let onResponseError;
    const send = (method) => (url, dataOrConfig, config) =>
      globalThis.serviceTransport({
        method,
        url,
        data: method === "get" || method === "delete" ? undefined : dataOrConfig,
        config: method === "get" ? dataOrConfig : config,
        defaults,
      }).catch((error) => (onResponseError ? onResponseError(error) : Promise.reject(error)));
    return {
      interceptors: {
        response: {
          use: (_onResponse, onError) => {
            onResponseError = onError;
          },
        },
      },
      get: send("get"),
      post: send("post"),
      put: send("put"),
      patch: send("patch"),
      delete: send("delete"),
    };
  };
  axios.create = create;
  export default axios;
`;

function findSource(target) {
  const candidates = [`${target}.ts`, `${target}.tsx`, path.join(target, "index.ts"), target];
  return candidates.find((candidate) => existsSync(candidate) && statSync(candidate).isFile());
}

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/services") {
      return {
        shortCircuit: true,
        url: pathToFileURL(path.join(servicesRoot, "src/index.ts")).href,
      };
    }
    if (specifier === "axios") {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(axiosStub)}` };
    }
    if (context.parentURL?.startsWith(pathToFileURL(servicesRoot).href) && specifier.startsWith(".")) {
      const target = path.resolve(path.dirname(fileURLToPath(context.parentURL)), specifier);
      const file = findSource(target);
      if (file) return { shortCircuit: true, url: pathToFileURL(file).href };
    }
    return nextResolve(specifier, context);
  },
  load(url, context, nextLoad) {
    if (url.startsWith(pathToFileURL(servicesRoot).href) && /\.tsx?$/.test(url)) {
      const output = transformFrontendSource(readFileSync(fileURLToPath(url), "utf8"), {
        format: "esm",
        target: "es2022",
        tsconfigRaw: { compilerOptions: { useDefineForClassFields: true } },
        loader: "ts",
        sourcefile: fileURLToPath(url),
      });
      return { format: "module", shortCircuit: true, source: output };
    }
    return nextLoad(url, context);
  },
});

function createServiceTransport(t) {
  const requests = [];
  const responses = [];
  const redirects = [];
  globalThis.window = {
    location: { pathname: "/acme/projects", search: "", hash: "", replace: (url) => redirects.push(url) },
  };
  globalThis.serviceTransport = (request) => {
    requests.push(request);
    const response = responses.shift() ?? { data: { ok: true } };
    return response instanceof Error ? Promise.reject(response) : Promise.resolve(response);
  };
  t.after(() => {
    delete globalThis.serviceTransport;
    delete globalThis.window;
  });
  return { requests, responses, redirects };
}

for (const variant of ["module", "link"]) {
  void test(`${variant} service preserves module-link routes, payloads and error bodies`, async (t) => {
    const { requests, responses } = createServiceTransport(t);
    const { ModuleService, ModuleLinkService } = await import("@plane/services");
    const service =
      variant === "module"
        ? new ModuleService("http://localhost:8000")
        : new ModuleLinkService("http://localhost:8000");
    const names =
      variant === "module"
        ? { create: "createModuleLink", update: "updateModuleLink", destroy: "deleteModuleLink" }
        : { create: "create", update: "update", destroy: "destroy" };
    const scope = ["acme", "project-1", "module-1"];
    const link = { id: "link-1", title: "Reference", url: "https://example.com" };
    responses.push({ data: link }, { data: link }, { data: undefined });
    assert.deepEqual(await service[names.create](...scope, link), link);
    assert.deepEqual(await service[names.update](...scope, link.id, { title: "Renamed" }), link);
    assert.equal(await service[names.destroy](...scope, link.id), undefined);
    const base = "/api/workspaces/acme/projects/project-1/modules/module-1/module-links/";
    assert.deepEqual(
      requests.map(({ method, url, data }) => ({ method, url, data })),
      [
        { method: "post", url: base, data: link },
        { method: "patch", url: `${base}link-1/`, data: { title: "Renamed" } },
        { method: "delete", url: `${base}link-1/`, data: undefined },
      ]
    );
    const response = { status: 400, data: { detail: "Invalid link" } };
    const error = Object.assign(new Error("invalid"), { response });
    responses.push(error, error, error);
    await assert.rejects(service[names.create](...scope, link), (failure) => failure === response);
    await assert.rejects(service[names.update](...scope, link.id, {}), (failure) => failure === response);
    await assert.rejects(service[names.destroy](...scope, link.id), (failure) => failure === response.data);
  });
}

await test("web ModuleService keeps its compatibility methods over the shared service", async (t) => {
  const { requests, responses, redirects } = createServiceTransport(t);

  const { ModuleService } = await import("../../apps/web/core/services/module.service.ts");
  const service = new ModuleService();
  const { API_BASE_URL } = await import(constantsUrl);

  await service.getWorkspaceModules("acme");
  assert.equal(requests[0].url, "/api/workspaces/acme/modules/");
  assert.equal(requests[0].method, "get");
  assert.equal(requests[0].defaults.baseURL, API_BASE_URL);
  assert.equal(requests[0].defaults.withCredentials, true);

  const replacement = { name: "Updated" };
  await service.updateModule("acme", "project-1", "module-1", replacement);
  await service.patchModule("acme", "project-1", "module-1", replacement);
  assert.deepEqual(
    requests.slice(1).map(({ method, url, data }) => ({ method, url, data })),
    [
      {
        method: "put",
        url: "/api/workspaces/acme/projects/project-1/modules/module-1/",
        data: replacement,
      },
      {
        method: "patch",
        url: "/api/workspaces/acme/projects/project-1/modules/module-1/",
        data: replacement,
      },
    ]
  );

  const nonAuthError = { response: { status: 400, data: { detail: "invalid module" } } };
  responses.push(Object.assign(new Error("bad request"), nonAuthError));
  await assert.rejects(
    service.createModule("acme", "project-1", { name: "New" }),
    (error) => error.response === nonAuthError.response
  );
  assert.equal(requests[3].method, "post");
  assert.equal(requests[3].url, "/api/workspaces/acme/projects/project-1/modules/");
  assert.deepEqual(requests[3].data, { name: "New" });
  assert.deepEqual(redirects, []);

  const authError = { response: { status: 401, data: { detail: "authentication required" } } };
  responses.push(Object.assign(new Error("unauthorized"), authError));
  await assert.rejects(service.getModules("acme", "project-1"), (error) => error.response === authError.response);
  assert.equal(requests[4].method, "get");
  assert.equal(requests[4].url, "/api/workspaces/acme/projects/project-1/modules/");
  assert.deepEqual(redirects, ["/?next_path=%2Facme%2Fprojects"]);

  await service.getModuleIssues("acme", "project-1", "module-1", { cursor: "next" }, { timeout: 123 });
  assert.equal(requests[5].method, "get");
  assert.equal(requests[5].url, "/api/workspaces/acme/projects/project-1/modules/module-1/issues/");
  assert.deepEqual(requests[5].config, { params: { cursor: "next" }, timeout: 123 });

  const operationErrorData = { detail: "issue cannot be added" };
  responses.push(Object.assign(new Error("invalid operation"), { response: { data: operationErrorData } }));
  await assert.rejects(
    service.addIssuesToModule("acme", "project-1", "module-1", { issues: ["issue-1"] }),
    (error) => error === operationErrorData
  );
  assert.equal(requests[6].method, "post");
  assert.equal(requests[6].url, "/api/workspaces/acme/projects/project-1/modules/module-1/issues/");
  assert.deepEqual(requests[6].data, { issues: ["issue-1"] });

  const { ModuleService: SharedModuleService } = await import("../../packages/services/src/module/module.service.ts");
  const sharedService = new SharedModuleService("https://api.shared.test");
  const sharedAuthError = { response: { status: 401, data: { detail: "shared client unauthorized" } } };
  responses.push(Object.assign(new Error("unauthorized"), sharedAuthError));
  await assert.rejects(sharedService.workspaceModulesList("acme"), (error) => error === sharedAuthError.response.data);
  assert.deepEqual(redirects, ["/?next_path=%2Facme%2Fprojects"]);

  const { APIService } = await import("../../apps/web/core/services/api.service.ts");
  class LocalAPIService extends APIService {
    list() {
      return this.get("/api/test/");
    }
  }
  const localService = new LocalAPIService("https://api.local.test");
  const localError = Object.assign(new Error("bad request"), { response: { status: 400, data: "bad" } });
  responses.push(localError);
  await assert.rejects(localService.list(), (error) => error === localError);
  assert.deepEqual(redirects, ["/?next_path=%2Facme%2Fprojects"]);

  const localAuthError = Object.assign(new Error("unauthorized"), {
    response: { status: 401, data: "unauthorized" },
  });
  responses.push(localAuthError);
  await assert.rejects(localService.list(), (error) => error === localAuthError);
  assert.deepEqual(redirects, ["/?next_path=%2Facme%2Fprojects", "/?next_path=%2Facme%2Fprojects"]);

  responses.push({ data: { id: "module-1", name: "Module details" } });
  assert.deepEqual(await service.getModuleDetails("acme", "project-1", "module-1"), {
    id: "module-1",
    name: "Module details",
  });
  assert.equal(requests.at(-1).method, "get");
  assert.equal(requests.at(-1).url, "/api/workspaces/acme/projects/project-1/modules/module-1/");

  await service.deleteModule("acme", "project-1", "module-1");
  assert.equal(requests.at(-1).method, "delete");
  assert.equal(requests.at(-1).url, "/api/workspaces/acme/projects/project-1/modules/module-1/");
});

await test("web CycleService preserves requests, query configuration, data errors, and auth redirects", async (t) => {
  const { requests, responses, redirects } = createServiceTransport(t);
  const { CycleService } = await import("../../apps/web/core/services/cycle.service.ts");
  const service = new CycleService();
  const base = "/api/workspaces/acme/projects/project-1/cycles/";
  const created = { id: "cycle-1", name: "New cycle" };
  responses.push({ data: created });
  assert.deepEqual(await service.create("acme", "project-1", { name: "New cycle" }), created);
  assert.deepEqual(requests[0].data, { name: "New cycle" });
  assert.equal(requests[0].method, "post");
  assert.equal(requests[0].url, base);

  const controller = new AbortController();
  await service.getWithParams("acme", "project-1", "current", controller.signal);
  assert.equal(requests.at(-1).url, base);
  assert.deepEqual(requests.at(-1).config, { params: { cycle_view: "current" }, signal: controller.signal });
  await service.retrieve("acme", "project-1", "cycle-1");
  assert.equal(requests.at(-1).url, `${base}cycle-1/`);
  assert.equal(requests.at(-1).method, "get");
  await service.getCycleIssues("acme", "project-1", "cycle-1", { cursor: "next" }, { timeout: 250 });
  assert.equal(requests.at(-1).url, `${base}cycle-1/cycle-issues/`);
  assert.deepEqual(requests.at(-1).config, { params: { cursor: "next" }, timeout: 250 });
  await service.update("acme", "project-1", "cycle-1", { name: "Renamed" });
  assert.equal(requests.at(-1).method, "patch");
  assert.deepEqual(requests.at(-1).data, { name: "Renamed" });
  await service.destroy("acme", "project-1", "cycle-1");
  assert.equal(requests.at(-1).method, "delete");
  assert.equal(requests.at(-1).url, `${base}cycle-1/`);
  const dates = { start_date: "2026-10-01", end_date: "2026-10-14" };
  await service.validateDates("acme", "project-1", dates);
  assert.equal(requests.at(-1).url, `${base}date-check/`);
  assert.deepEqual(requests.at(-1).data, dates);

  const errorData = { detail: "Cycle overlap" };
  responses.push(Object.assign(new Error("invalid"), { response: { status: 400, data: errorData } }));
  await assert.rejects(
    service.create("acme", "project-1", {}),
    (error) => error.response.status === 400 && error.response.data === errorData
  );
  assert.deepEqual(redirects, []);
  responses.push(Object.assign(new Error("unauthorized"), { response: { status: 401, data: errorData } }));
  await assert.rejects(
    service.getWorkspaceCycles("acme"),
    (error) => error.response.status === 401 && error.response.data === errorData
  );
  assert.deepEqual(redirects, ["/?next_path=%2Facme%2Fprojects"]);
});

await test("late unauthorized responses preserve the sign-in return destination", async (t) => {
  const { responses, redirects } = createServiceTransport(t);
  const { APIService } = await import("../../apps/web/core/services/api.service.ts");
  const service = new APIService("http://localhost:8000");
  window.location.pathname = "/";
  window.location.search = "?next_path=%2Facme%2Fprojects%3Fview%3Dboard";
  const error = Object.assign(new Error("unauthorized"), { response: { status: 401 } });
  responses.push(error);
  await assert.rejects(service.get("/api/users/me/"), error);
  assert.deepEqual(redirects, []);
});

await test("expired sessions preserve the full deep link before sign-in", async (t) => {
  const { responses, redirects } = createServiceTransport(t);
  const { APIService } = await import("../../apps/web/core/services/api.service.ts");
  const service = new APIService("http://localhost:8000");
  window.location.search = "?view=board";
  window.location.hash = "#selected";
  const error = Object.assign(new Error("unauthorized"), { response: { status: 401 } });
  responses.push(error);
  await assert.rejects(service.get("/api/users/me/"), error);
  const destination = new URL(redirects[0], "http://localhost:8000").searchParams.get("next_path");
  assert.equal(destination, "/acme/projects?view=board#selected");
});

const { APIService } = await import("../../packages/services/src/api.service.ts");
class ResponseContractService extends APIService {
  unwrap(request, policy) {
    return this.responseData(request, policy);
  }
}
for (const policy of ["data", "response", "data-or-error", "response-or-error"]) {
  void test(`shared response handling preserves the ${policy} error contract`, async () => {
    const service = new ResponseContractService("https://example.test");
    const payload = { id: "accepted" };
    assert.equal(await service.unwrap(Promise.resolve({ data: payload }), policy), payload);
    const response = { status: 409, data: { error: "conflict" } };
    const failure = Object.assign(new Error("request failed"), { response });
    await assert.rejects(
      service.unwrap(Promise.reject(failure), policy),
      (reason) => reason === (policy.startsWith("response") ? response : response.data)
    );
    const transportFailure = new Error("network unavailable");
    await assert.rejects(
      service.unwrap(Promise.reject(transportFailure), policy),
      (reason) => reason === (policy.endsWith("or-error") ? transportFailure : undefined)
    );
    const emptyFailure = { response: { data: null } };
    await assert.rejects(
      service.unwrap(Promise.reject(emptyFailure), policy),
      (reason) =>
        reason ===
        (policy.startsWith("response") ? emptyFailure.response : policy.endsWith("or-error") ? emptyFailure : null)
    );
    const falsyFailure = { response: { data: false } };
    await assert.rejects(
      service.unwrap(Promise.reject(falsyFailure), policy),
      (reason) => reason === (policy.startsWith("response") ? falsyFailure.response : false)
    );
  });
}

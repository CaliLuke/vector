import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const webRoot = path.join(repositoryRoot, "apps/web");
const requireFromWeb = createRequire(path.join(webRoot, "package.json"));
globalThis.__moduleIssuesTestMobx = requireFromWeb("mobx");
globalThis.__moduleIssuesTestMobxUtils = requireFromWeb("mobx-utils");

registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    const cjsShims = {
      mobx: ["action", "computed", "makeObservable", "observable", "runInAction"],
      "mobx-utils": ["computedFn"],
    };
    if (cjsShims[specifier]) {
      const globalName = specifier === "mobx" ? "__moduleIssuesTestMobx" : "__moduleIssuesTestMobxUtils";
      const exports = cjsShims[specifier]
        .map((name) => `export const ${name} = globalThis.${globalName}.${name};`)
        .join("\n");
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(exports)}` };
    }
    return nextResolve(specifier, context);
  },
});

const { ModuleIssues } = await import("../../apps/web/core/store/issue/module/issue.store.ts");

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, reject, resolve };
}

function issue(id) {
  return { id, project_id: "project-id", created_at: "2026-01-01T00:00:00Z" };
}

function response(...issues) {
  return { results: issues, total_count: issues.length };
}

function createModuleIssues() {
  const issuesById = new Map();
  const issues = {
    addIssue(items) {
      for (const item of items) issuesById.set(item.id, item);
    },
    getIssuesByIds(ids) {
      return ids.map((id) => issuesById.get(id)).filter(Boolean);
    },
  };
  const rootStore = {
    router: {},
    module: { fetchModuleDetails() {} },
  };
  const rootIssueStore = {
    rootStore,
    moduleId: "module-id",
    issues,
    issueDetail: { relation: { extractRelationsFromIssues() {} } },
  };
  const filters = { getFilterParams: () => ({}) };
  const store = new ModuleIssues(rootIssueStore, filters);
  const requests = [];
  store.issueService = {
    getIssues(_workspaceSlug, _projectId, _params, config) {
      const request = deferred();
      requests.push({ ...request, signal: config.signal });
      return request.promise;
    },
  };
  return { requests, store };
}

const options = { canGroup: true, perPageCount: 100 };
const fetchModuleIssues = (store) => store.fetchIssues("default", "project-id", "init-loader", options, "module-id");

test("a superseded module request rejection does not escape or clear the active loader", async () => {
  const { requests, store } = createModuleIssues();
  const staleRequest = fetchModuleIssues(store);
  const currentRequest = fetchModuleIssues(store);

  assert.equal(requests[0].signal.aborted, true);
  requests[0].reject(new Error("request aborted"));
  assert.equal(await staleRequest, undefined);
  assert.equal(store.getIssueLoader(), "init-loader");

  requests[1].resolve(response(issue("current-ticket")));
  await currentRequest;

  assert.equal(store.getIssueLoader(), undefined);
  assert.deepEqual(store.getIssueIds(), ["current-ticket"]);
});

test("a late stale module response cannot overwrite the current request's issues", async () => {
  const { requests, store } = createModuleIssues();
  const staleRequest = fetchModuleIssues(store);
  const currentRequest = fetchModuleIssues(store);

  requests[1].resolve(response(issue("current-ticket")));
  await currentRequest;
  requests[0].resolve(response(issue("stale-ticket")));
  assert.equal(await staleRequest, undefined);

  assert.deepEqual(store.getIssueIds(), ["current-ticket"]);
});

test("an active module request failure remains rejected and clears its loader", async () => {
  const { requests, store } = createModuleIssues();
  const requestError = new Error("module issues unavailable");
  const fetch = fetchModuleIssues(store);

  requests[0].reject(requestError);
  await fetch.then(
    () => assert.fail("the active request failure should remain visible"),
    (error) => assert.equal(error, requestError)
  );

  assert.equal(store.getIssueLoader(), undefined);
  assert.equal(store.getIssueIds(), undefined);
});

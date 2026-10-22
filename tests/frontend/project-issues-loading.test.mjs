import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { existsSync, readFileSync, statSync } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const repositoryRoot = fileURLToPath(new URL("../../", import.meta.url));
const webRoot = path.join(repositoryRoot, "apps/web");
const requireFromWeb = createRequire(path.join(webRoot, "package.json"));
const typescript = requireFromWeb("typescript");
globalThis.__projectIssuesTestMobx = requireFromWeb("mobx");
globalThis.__projectIssuesTestMobxUtils = requireFromWeb("mobx-utils");

// Load the real store implementation under node:test. The small resolver bridges
// the web app's `@/` aliases and extensionless TS imports; TypeScript transpiles
// enums that Node's strip-only TS loader cannot handle.
const candidates = (target) => [
  `${target}.ts`,
  `${target}.tsx`,
  path.join(target, "index.ts"),
  path.join(target, "index.tsx"),
  target,
];

registerHooks({
  resolve(specifier, context, nextResolve) {
    let target;
    const cjsShims = {
      mobx: ["action", "computed", "makeObservable", "observable", "runInAction"],
      "mobx-utils": ["computedFn"],
    };
    if (cjsShims[specifier]) {
      const globalName = specifier === "mobx" ? "__projectIssuesTestMobx" : "__projectIssuesTestMobxUtils";
      const exports = cjsShims[specifier]
        .map((name) => `export const ${name} = globalThis.${globalName}.${name};`)
        .join("\n");
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(exports)}` };
    }
    if (specifier.startsWith("@/")) {
      target = path.resolve(webRoot, "core", specifier.slice(2));
    } else if (
      (specifier.startsWith("./") || specifier.startsWith("../")) &&
      context.parentURL?.startsWith(pathToFileURL(webRoot).href)
    ) {
      target = fileURLToPath(new URL(specifier, context.parentURL));
    }

    if (target) {
      const file = candidates(target).find((candidate) => existsSync(candidate) && statSync(candidate).isFile());
      if (file) return nextResolve(pathToFileURL(file).href, context);
    }

    return nextResolve(specifier, context);
  },
  load(url, context, nextLoad) {
    if (url.startsWith(pathToFileURL(webRoot).href) && /\.tsx?$/.test(url)) {
      const source = readFileSync(fileURLToPath(url), "utf8");
      const compilerOptions = {
        module: typescript.ModuleKind.ESNext,
        target: typescript.ScriptTarget.ES2022,
      };
      if (url.endsWith(".tsx")) compilerOptions.jsx = typescript.JsxEmit.ReactJSX;
      const output = typescript.transpileModule(source, {
        compilerOptions,
      }).outputText;
      return { format: "module", shortCircuit: true, source: output };
    }

    return nextLoad(url, context);
  },
});

const { ProjectIssues } = await import("../../apps/web/core/store/issue/project/issue.store.ts");

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

function createProjectIssues() {
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
    projectRoot: { project: { fetchProjectDetails() {} } },
  };
  const rootIssueStore = {
    rootStore,
    issues,
    issueDetail: { relation: { extractRelationsFromIssues() {} } },
  };
  const filters = { getFilterParams: () => ({}) };
  const store = new ProjectIssues(rootIssueStore, filters);
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

test("an initial undefined rejection exposes an error and a successful retry loads records", async () => {
  const { requests, store } = createProjectIssues();
  const failedLoad = store.fetchIssues("default", "project-id", "init-loader", options);
  requests[0].reject(undefined);

  await failedLoad.then(
    () => assert.fail("the failed request should reject"),
    (error) => assert.equal(error, undefined)
  );

  assert.equal(store.loadError, true);
  assert.equal(store.getGroupIssueCount(undefined, undefined, false), undefined);

  const retry = store.fetchIssues("default", "project-id", "init-loader", options);
  requests[1].resolve(response(issue("ticket-1"), issue("ticket-2")));
  await retry;

  assert.equal(store.loadError, false);
  assert.equal(store.getGroupIssueCount(undefined, undefined, false), 2);
  assert.deepEqual(store.getIssueIds(), ["ticket-1", "ticket-2"]);
});

test("a stale cancelled rejection cannot clear the current loader or set its error state", async () => {
  const { requests, store } = createProjectIssues();
  const staleRequest = store.fetchIssues("default", "project-id", "init-loader", options);
  const currentRequest = store.fetchIssues("default", "project-id", "init-loader", options);

  assert.equal(requests[0].signal.aborted, true);
  requests[0].reject(undefined);
  await staleRequest;

  assert.equal(store.getIssueLoader(), "init-loader");
  assert.equal(store.loadError, false);

  requests[1].resolve(response(issue("current-ticket")));
  await currentRequest;
  assert.equal(store.loadError, false);
  assert.equal(store.getIssueLoader(), undefined);
  assert.deepEqual(store.getIssueIds(), ["current-ticket"]);
});

test("a stale response that completes after the current request cannot overwrite its records", async () => {
  const { requests, store } = createProjectIssues();
  const staleRequest = store.fetchIssues("default", "project-id", "init-loader", options);
  const currentRequest = store.fetchIssues("default", "project-id", "init-loader", options);

  requests[1].resolve(response(issue("current-ticket")));
  await currentRequest;
  requests[0].resolve(response(issue("stale-ticket")));
  await staleRequest;

  assert.equal(store.loadError, false);
  assert.equal(store.getGroupIssueCount(undefined, undefined, false), 1);
  assert.deepEqual(store.getIssueIds(), ["current-ticket"]);
});

test("clearing while a request is pending prevents its late response from repopulating the store", async () => {
  const { requests, store } = createProjectIssues();
  const pendingRequest = store.fetchIssues("default", "project-id", "init-loader", options);

  store.clear();
  assert.equal(requests[0].signal.aborted, true);
  requests[0].resolve(response(issue("cleared-ticket")));
  await pendingRequest;

  assert.equal(store.getGroupIssueCount(undefined, undefined, false), undefined);
  assert.equal(store.getIssueIds(), undefined);
});

test("a response-processing failure after the fetch resolves remains visible as a load error", async () => {
  const { requests, store } = createProjectIssues();
  const processingError = new Error("issue relation processing failed");
  store.rootIssueStore.issueDetail.relation.extractRelationsFromIssues = () => {
    throw processingError;
  };

  const fetch = store.fetchIssues("default", "project-id", "init-loader", options);
  requests[0].resolve(response(issue("ticket-1")));

  await fetch.then(
    () => assert.fail("response processing should reject"),
    (error) => assert.equal(error, processingError)
  );

  assert.equal(store.loadError, true);
  assert.equal(store.getIssueLoader(), undefined);
});

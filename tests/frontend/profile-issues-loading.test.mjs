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
globalThis.__profileIssuesTestMobx = requireFromWeb("mobx");
globalThis.__profileIssuesTestMobxUtils = requireFromWeb("mobx-utils");

registerHooks({
  resolve(specifier, context, nextResolve) {
    let target;
    const cjsShims = {
      mobx: ["action", "computed", "makeObservable", "observable", "runInAction"],
      "mobx-utils": ["computedFn"],
    };
    if (cjsShims[specifier]) {
      const globalName = specifier === "mobx" ? "__profileIssuesTestMobx" : "__profileIssuesTestMobxUtils";
      const exports = cjsShims[specifier]
        .map((name) => `export const ${name} = globalThis.${globalName}.${name};`)
        .join("\n");
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(exports)}` };
    }
    if (specifier === "@/services/user.service") {
      return { shortCircuit: true, url: "data:text/javascript,export%20class%20UserService%20%7B%7D" };
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
      const file = [
        `${target}.ts`,
        `${target}.tsx`,
        path.join(target, "index.ts"),
        path.join(target, "index.tsx"),
        target,
      ].find((candidate) => existsSync(candidate) && statSync(candidate).isFile());
      if (file) return nextResolve(pathToFileURL(file).href, context);
    }

    return nextResolve(specifier, context);
  },
  load(url, context, nextLoad) {
    if (url.startsWith(pathToFileURL(webRoot).href) && /\.tsx?$/.test(url)) {
      const source = readFileSync(fileURLToPath(url), "utf8");
      const output = typescript.transpileModule(source, {
        compilerOptions: {
          module: typescript.ModuleKind.ESNext,
          target: typescript.ScriptTarget.ES2022,
          ...(url.endsWith(".tsx") ? { jsx: typescript.JsxEmit.ReactJSX } : {}),
        },
      }).outputText;
      return { format: "module", shortCircuit: true, source: output };
    }

    return nextLoad(url, context);
  },
});

const { ProfileIssues } = await import("../../apps/web/core/store/issue/profile/issue.store.ts");

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

function createProfileIssues() {
  const issuesById = new Map();
  const issues = {
    addIssue(items) {
      for (const item of items) issuesById.set(item.id, item);
    },
    getIssuesByIds(ids) {
      return ids.map((id) => issuesById.get(id)).filter(Boolean);
    },
  };
  const rootStore = { router: {} };
  const rootIssueStore = {
    rootStore,
    issues,
    issueDetail: { relation: { extractRelationsFromIssues() {} } },
  };
  const filters = { getFilterParams: () => ({}) };
  const store = new ProfileIssues(rootIssueStore, filters);
  const requests = [];
  store.userService = {
    getUserProfileIssues(_workspaceSlug, _userId, _params, config) {
      const request = deferred();
      requests.push({ ...request, params: _params, signal: config.signal });
      return request.promise;
    },
  };
  return { requests, store };
}

const options = { canGroup: true, perPageCount: 100 };
const fetchCreatedIssues = (store) => store.fetchIssues("default", "user-id", "init-loader", options, "created");

test("a superseded profile request rejection does not clear the active loader or escape as an error", async () => {
  const { requests, store } = createProfileIssues();
  const staleRequest = fetchCreatedIssues(store);
  const currentRequest = fetchCreatedIssues(store);

  assert.equal(requests[0].signal.aborted, true);
  requests[0].reject(new Error("request aborted"));
  assert.equal(await staleRequest, undefined);
  assert.equal(store.getIssueLoader(), "init-loader");

  requests[1].resolve(response(issue("created-ticket")));
  await currentRequest;

  assert.equal(store.getIssueLoader(), undefined);
  assert.deepEqual(store.getIssueIds(), ["created-ticket"]);
});

test("a superseded profile response cannot overwrite the current request's issues", async () => {
  const { requests, store } = createProfileIssues();
  const staleRequest = fetchCreatedIssues(store);
  const currentRequest = fetchCreatedIssues(store);

  requests[1].resolve(response(issue("current-ticket")));
  await currentRequest;
  requests[0].resolve(response(issue("stale-ticket")));
  assert.equal(await staleRequest, undefined);

  assert.deepEqual(store.getIssueIds(), ["current-ticket"]);
});

test("a stale response from a prior profile view cannot replace the active view's issues", async () => {
  const { requests, store } = createProfileIssues();
  const createdRequest = fetchCreatedIssues(store);
  const subscribedRequest = store.fetchIssues("default", "user-id", "init-loader", options, "subscribed");

  assert.equal(requests[0].signal.aborted, true);
  assert.equal(requests[0].params.created_by, "user-id");
  assert.equal(requests[1].params.subscriber, "user-id");
  requests[1].resolve(response(issue("subscribed-ticket")));
  await subscribedRequest;
  requests[0].resolve(response(issue("created-ticket")));
  assert.equal(await createdRequest, undefined);

  assert.equal(store.currentView, "subscribed");
  assert.deepEqual(store.getIssueIds(), ["subscribed-ticket"]);
});

test("an active profile request failure remains rejected and clears its loader", async () => {
  const { requests, store } = createProfileIssues();
  const requestError = new Error("profile issues unavailable");
  const fetch = fetchCreatedIssues(store);

  requests[0].reject(requestError);
  await fetch.then(
    () => assert.fail("the active request failure should remain visible"),
    (error) => assert.equal(error, requestError)
  );

  assert.equal(store.getIssueLoader(), undefined);
  assert.equal(store.getIssueIds(), undefined);
});

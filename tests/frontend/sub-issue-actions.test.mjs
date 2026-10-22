import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const requireFromWeb = createRequire(new URL("../../apps/web/package.json", import.meta.url));
const mobx = requireFromWeb("mobx");
globalThis.__subIssueMobx = mobx;
globalThis.__subIssueMobxUtils = requireFromWeb("mobx-utils");
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    const names = {
      mobx: ["action", "computed", "makeObservable", "observable", "runInAction"],
      "mobx-utils": ["computedFn"],
    }[specifier];
    if (names) {
      const source = specifier === "mobx" ? "__subIssueMobx" : "__subIssueMobxUtils";
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(names.map((name) => `export const ${name} = globalThis.${source}.${name};`).join("\n"))}`,
      };
    }
    return nextResolve(specifier, context);
  },
});

const { IssueSubIssuesStore } = await import("../../apps/web/core/store/issue/issue-details/sub_issues.store.ts");

function setup() {
  const issuesMap = mobx.observable({ parent: { id: "parent", sub_issues_count: 0 } });
  const issues = {
    issuesMap,
    addIssue: mobx.action((rows) => {
      for (const row of rows) issuesMap[row.id] = row;
    }),
    updateIssue: mobx.action((id, changes) => Object.assign(issuesMap[id], changes)),
  };
  const store = new IssueSubIssuesStore({ rootIssueStore: { issues } }, "issues");
  const child = { id: "child", project_id: "project" };
  const response = { sub_issues: [child], state_distribution: { backlog: ["child"] } };
  store.issueService = {
    subIssues: async () => response,
    addSubIssues: async () => response,
  };
  return { store, issuesMap };
}

test("adding a sub-item publishes its list and parent count in one action after the request", async (t) => {
  mobx.configure({ enforceActions: "always" });
  t.after(() => mobx.configure({ enforceActions: "observed" }));
  const warnings = [];
  t.mock.method(console, "warn", (message) => warnings.push(message));
  const { store, issuesMap } = setup();
  const snapshots = [];
  const dispose = mobx.autorun(() => {
    snapshots.push({ ids: [...(store.subIssues.parent ?? [])], count: issuesMap.parent.sub_issues_count });
  });
  t.after(dispose);

  await store.createSubIssues("default", "project", "parent", ["child"]);

  assert.deepEqual(snapshots, [
    { ids: [], count: 0 },
    { ids: ["child"], count: 1 },
  ]);
  assert.equal(issuesMap.child.id, "child");
  assert.deepEqual(warnings, []);
});

test("fetching sub-items clears the observed loader within an action", async (t) => {
  mobx.configure({ enforceActions: "always" });
  t.after(() => mobx.configure({ enforceActions: "observed" }));
  const warnings = [];
  t.mock.method(console, "warn", (message) => warnings.push(message));
  const { store, issuesMap } = setup();
  const loaders = [];
  const dispose = mobx.autorun(() => loaders.push(store.loader));
  t.after(dispose);

  await store.fetchSubIssues("default", "project", "parent");

  assert.deepEqual(loaders, [undefined, "init-loader", undefined]);
  assert.equal(issuesMap.parent.sub_issues_count, 1);
  assert.deepEqual(warnings, []);
});

test("duplicate root sub-item fetch completions keep the child list visible", () => {
  const { store } = setup();
  const helperKey = "parent_root";

  store.setSubIssueHelperState(helperKey, "preview_loader", "parent", true);
  store.setSubIssueHelperState(helperKey, "preview_loader", "parent", true);
  store.setSubIssueHelperState(helperKey, "issue_visibility", "parent", true);
  store.setSubIssueHelperState(helperKey, "issue_visibility", "parent", true);
  store.setSubIssueHelperState(helperKey, "preview_loader", "parent", false);
  store.setSubIssueHelperState(helperKey, "preview_loader", "parent", false);

  assert.deepEqual(store.subIssueHelpersByIssueId(helperKey), {
    preview_loader: [],
    issue_visibility: ["parent"],
    issue_loader: [],
  });

  store.setSubIssueHelperState(helperKey, "issue_visibility", "parent", false);
  assert.deepEqual(store.subIssueHelpersByIssueId(helperKey).issue_visibility, []);
});

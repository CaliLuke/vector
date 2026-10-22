import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const requireFromWeb = createRequire(new URL("../../apps/web/package.json", import.meta.url));
const mobx = requireFromWeb("mobx");
globalThis.__workspaceDraftMobx = mobx;
globalThis.__workspaceDraftMobxUtils = requireFromWeb("mobx-utils");
globalThis.__workspaceDraftService = {
  createIssue: async () => undefined,
};
registerWebModuleHooks();
registerHooks({
  resolve(specifier, context, nextResolve) {
    const exports = {
      mobx: ["action", "computed", "makeObservable", "observable", "runInAction"],
      "mobx-utils": ["computedFn"],
    }[specifier];
    if (exports) {
      const source = specifier === "mobx" ? "__workspaceDraftMobx" : "__workspaceDraftMobxUtils";
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent(exports.map((name) => `export const ${name} = globalThis.${source}.${name};`).join("\n"))}`,
      };
    }
    if (specifier === "@/services/issue/workspace_draft.service") {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent("export default globalThis.__workspaceDraftService;")}`,
      };
    }
    return nextResolve(specifier, context);
  },
});

const { WorkspaceDraftIssues } = await import("../../apps/web/core/store/issue/workspace-draft/issue.store.ts");

function setup() {
  const workspaceUserInfo = mobx.observable({ default: { draft_issue_count: 0 } });
  const store = new WorkspaceDraftIssues({
    workspaceSlug: "default",
    rootStore: { user: { permission: { workspaceUserInfo } } },
  });
  return { store, workspaceUserInfo };
}

test("creating a workspace draft issue clears its observed loader in an action", async (t) => {
  mobx.configure({ enforceActions: "always" });
  t.after(() => mobx.configure({ enforceActions: "observed" }));
  const warnings = [];
  t.mock.method(console, "warn", (message) => warnings.push(message));
  const created = { id: "draft-1", created_at: "2026-10-10T12:00:00Z" };
  globalThis.__workspaceDraftService.createIssue = async () => created;
  const { store, workspaceUserInfo } = setup();
  const snapshots = [];
  const dispose = mobx.autorun(() =>
    snapshots.push({ loader: store.loader, ids: [...(store.issueMapIds.default ?? [])] })
  );
  t.after(dispose);

  const result = await store.createIssue("default", { name: "Draft" });

  assert.equal(result, created);
  assert.deepEqual(snapshots, [
    { loader: undefined, ids: [] },
    { loader: "create", ids: [] },
    { loader: undefined, ids: ["draft-1"] },
  ]);
  assert.equal(workspaceUserInfo.default.draft_issue_count, 1);
  assert.deepEqual(warnings, []);
});

test("failed draft creation clears its observed loader in an action", async (t) => {
  mobx.configure({ enforceActions: "always" });
  t.after(() => mobx.configure({ enforceActions: "observed" }));
  const warnings = [];
  t.mock.method(console, "warn", (message) => warnings.push(message));
  const failure = new Error("create failed");
  globalThis.__workspaceDraftService.createIssue = async () => {
    throw failure;
  };
  const { store } = setup();
  const loaders = [];
  const dispose = mobx.autorun(() => loaders.push(store.loader));
  t.after(dispose);

  await assert.rejects(store.createIssue("default", { name: "Draft" }), failure);

  assert.deepEqual(loaders, [undefined, "create", undefined]);
  assert.deepEqual(warnings, []);
});

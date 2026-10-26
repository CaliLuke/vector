import assert from "node:assert/strict";
import { createRequire } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import path from "node:path";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { stickyListOptions, changeStickyOptions, createStickyOptions } = await fromRoot(
  "apps/web/core/queries/stickies.ts"
);
const { StickyService } = await fromRoot("apps/web/core/services/sticky.service.ts");
const { reorderedSortOrder } = await fromRoot("apps/web/core/lib/reorder.ts");
const { homeWidgetsOptions, updateHomeWidgetOptions, homeKeys, homeLinkMutationOptions } =
  await import("../../apps/web/core/queries/home.ts");
const { WorkspaceService } = await fromRoot("apps/web/core/services/workspace.service.ts");
const { issueListGroupOptions, issueListKeys, issueListOptions } =
  await import("../../apps/web/core/queries/issue-lists.ts");
const { identityKeys, advanceIdentitySession } = await fromRoot("apps/web/core/queries/identity.ts");
const { permissionKeys } = await fromRoot("apps/web/core/queries/permissions.ts");
const { QueryClient, MutationObserver, InfiniteQueryObserver } = createRequire(path.resolve("apps/web/package.json"))(
  "@tanstack/react-query"
);

const workspaceIssueResponse = (id) => ({
  results: [{ id, project_id: "p1" }],
  total_count: 2,
  next_page_results: true,
  next_cursor: "next",
  prev_cursor: "",
});

await test("sticky pagination and search retain each workspace's query state", async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  t.after(() => client.clear());
  const requests = [];
  t.mock.method(StickyService.prototype, "getStickies", async (workspace, cursor, search) => {
    requests.push({ workspace, cursor, search });
    return {
      results: [{ id: `${workspace}-${search || "sticky"}`, sort_order: 1 }],
      next_cursor: `${workspace}-next`,
      next_page_results: true,
    };
  });
  await client.fetchInfiniteQuery(stickyListOptions("A", ""));
  await client.fetchInfiniteQuery(stickyListOptions("B", ""));
  const observer = new InfiniteQueryObserver(client, stickyListOptions("A", ""));
  await observer.fetchNextPage();
  assert.deepEqual(requests.at(-1), { workspace: "A", cursor: "A-next", search: "" });
  await client.fetchInfiniteQuery(stickyListOptions("A", "needle"));
  assert.equal(client.getQueryData(stickyListOptions("A", "").queryKey).pages[0].results[0].id, "A-sticky");
  assert.equal(client.getQueryData(stickyListOptions("B", "").queryKey).pages[0].results[0].id, "B-sticky");
});

await test("failed sticky deletion restores the row without losing another sticky's edit", async (t) => {
  const client = new QueryClient();
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  const key = stickyListOptions("A", "").queryKey;
  client.setQueryData(key, {
    pages: [
      {
        results: [
          { id: "removed", name: "old", sort_order: 2 },
          { id: "kept", name: "old", sort_order: 1 },
        ],
      },
    ],
    pageParams: ["first"],
  });
  const pending = Promise.withResolvers();
  t.mock.method(StickyService.prototype, "deleteSticky", () => pending.promise);
  t.mock.method(StickyService.prototype, "updateSticky", async (_workspace, id, patch) => ({ id, ...patch }));
  const deleting = new MutationObserver(client, changeStickyOptions(client, "A", "removed"));
  const updating = new MutationObserver(client, changeStickyOptions(client, "A", "kept"));
  const deletion = deleting.mutate({ kind: "remove", id: "removed" });
  await updating.mutate({ kind: "update", id: "kept", patch: { name: "edited" } });
  pending.reject(new Error("offline"));
  await assert.rejects(deletion, /offline/);
  const rows = client.getQueryData(key).pages.flatMap((page) => page.results);
  assert.equal(rows.find((item) => item.id === "kept").name, "edited");
  assert.equal(rows.find((item) => item.id === "removed").name, "old");
});

await test("sticky autosaves preserve falsy values and execute in order", async (t) => {
  const client = new QueryClient();
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  const key = stickyListOptions("A", "").queryKey;
  client.setQueryData(key, {
    pages: [{ results: [{ id: "n", name: "initial", sort_order: 4 }] }],
    pageParams: ["first"],
  });
  const first = Promise.withResolvers();
  const entered = Promise.withResolvers();
  const requests = [];
  t.mock.method(StickyService.prototype, "updateSticky", (_workspace, _id, patch) => {
    requests.push(patch);
    entered.resolve();
    return requests.length === 1 ? first.promise : Promise.resolve(patch);
  });
  const observer = new MutationObserver(client, changeStickyOptions(client, "A", "n"));
  const one = observer.mutate({ kind: "update", id: "n", patch: { name: "first" } });
  await entered.promise;
  const two = observer.mutate({ kind: "update", id: "n", patch: { name: "", sort_order: 0 } });
  assert.equal(requests.length, 1);
  first.resolve({ id: "n", name: "first" });
  await Promise.all([one, two]);
  assert.deepEqual(requests, [{ name: "first" }, { name: "", sort_order: 0 }]);
  assert.equal(client.getQueryData(key).pages[0].results[0].name, "");
  assert.equal(client.getQueryData(key).pages[0].results[0].sort_order, 0);
});

await test("reordering uses both neighbors, including a zero sort order", () => {
  const rows = [
    { id: "high", sort_order: 10 },
    { id: "zero", sort_order: 0 },
    { id: "low", sort_order: -10 },
  ];
  assert.equal(reorderedSortOrder(rows, "low", "zero", "reorder-above"), 5);
  assert.equal(reorderedSortOrder(rows, "high", "zero", "reorder-below"), -5);
});

await test("late widget loads and toggles update only their originating workspace", async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  t.after(() => client.clear());
  const first = Promise.withResolvers();
  const toggle = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceWidgets", (workspace) =>
    workspace === "A" ? first.promise : Promise.resolve([{ key: "recents", is_enabled: false, sort_order: 1 }])
  );
  t.mock.method(WorkspaceService.prototype, "updateWorkspaceWidget", () => toggle.promise);
  const pending = client.fetchQuery(homeWidgetsOptions("A"));
  await client.fetchQuery(homeWidgetsOptions("B"));
  first.resolve([{ key: "recents", is_enabled: true, sort_order: 1 }]);
  await pending;
  assert.equal(client.getQueryData(homeKeys.widgets("B"))[0].is_enabled, false);
  const mutation = new MutationObserver(client, updateHomeWidgetOptions(client, "B"));
  const pendingToggle = mutation.mutate({ key: "recents", patch: { is_enabled: true } });
  toggle.resolve({ key: "recents", is_enabled: true, sort_order: 1 });
  await pendingToggle;
  assert.equal(client.getQueryData(homeKeys.widgets("A"))[0].is_enabled, true);
  assert.equal(client.getQueryData(homeKeys.widgets("B"))[0].is_enabled, true);
});

await test("pagination from a previous workspace cannot append rows after switching workspace", async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  for (const workspaceSlug of ["A", "B"])
    client.setQueryData(permissionKeys.workspaceMembership("alice", workspaceSlug), {
      is_active: true,
      role: 15,
      updated_at: `revision-${workspaceSlug}`,
    });
  for (const workspaceSlug of ["A", "B"])
    client.setQueryData(permissionKeys.workspaceProjectRoles("alice", workspaceSlug, 15, `revision-${workspaceSlug}`), {
      p1: 20,
    });
  const pendingPage = Promise.withResolvers();
  const pageStarted = Promise.withResolvers();

  let activeWorkspace = "A";
  const makeRead = (workspaceSlug) => ({
    scope: {
      accountId: "alice",
      sessionGeneration: 0,
      workspaceSlug,
      workspaceRole: 15,
      membershipRevision: `revision-${workspaceSlug}`,
      projectRoles: [["p1", 20]],
    },
    pagination: { canGroup: false, perPageCount: 1 },
    request: {
      family: "workspace",
      input: {
        kind: "workspace",
        workspaceSlug,
        viewId: "all-issues",
        accountId: "alice",
        richFilters: {},
        displayFilters: { order_by: "sort_order" },
        isScopeCurrent: () => activeWorkspace === workspaceSlug,
        isReadScopeCurrent: () => activeWorkspace === workspaceSlug,
      },
      isReadCurrent: () => activeWorkspace === workspaceSlug,
    },
  });
  const readA = makeRead("A");
  const readB = makeRead("B");
  t.mock.method(WorkspaceService.prototype, "getViewIssues", async (workspace, params) => {
    if (workspace === "A" && params.cursor === "1:1:0") {
      pageStarted.resolve();
      return pendingPage.promise;
    }
    return workspaceIssueResponse(`${workspace}-first`);
  });
  const firstA = await client.fetchQuery(issueListOptions(readA));
  const groupA = { groupId: undefined, subGroupId: undefined };
  const groupOptionsA = issueListGroupOptions(readA, firstA, groupA);
  const pending = new InfiniteQueryObserver(client, groupOptionsA).fetchNextPage();
  await pageStarted.promise;
  activeWorkspace = "B";
  const firstB = await client.fetchQuery(issueListOptions(readB));
  const groupOptionsB = issueListGroupOptions(readB, firstB, groupA);
  await client.fetchInfiniteQuery(groupOptionsB);
  pendingPage.resolve(workspaceIssueResponse("A-late"));
  await pending;
  const queryA = client.getQueryCache().find({ queryKey: groupOptionsA.queryKey, exact: true });
  const queryB = client.getQueryCache().find({ queryKey: groupOptionsB.queryKey, exact: true });
  assert.deepEqual(
    queryA.state.data.pages[0].results.map((row) => row.id),
    ["A-first"]
  );
  assert.equal(queryA.state.data.pages.length, 1, "the stale A page is not appended after the workspace switch");
  assert.deepEqual(
    queryB.state.data.pages[0].results.map((row) => row.id),
    ["B-first"]
  );
  assert.notDeepEqual(issueListKeys.list(readA), issueListKeys.list(readB));
});

await test("quick-link mutations preserve workspace scope and keep failed deletions visible", async (t) => {
  const client = new QueryClient();
  t.after(() => client.clear());
  client.setQueryData(homeKeys.links("A"), [{ id: "a", title: "original" }]);
  client.setQueryData(homeKeys.links("B"), [{ id: "b", title: "other workspace" }]);
  const saved = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "updateWorkspaceLink", () => saved.promise);
  t.mock.method(WorkspaceService.prototype, "deleteWorkspaceLink", async () => {
    throw new Error("offline");
  });
  const mutation = new MutationObserver(client, homeLinkMutationOptions(client, "A"));
  const request = mutation.mutate({ kind: "update", id: "a", data: { title: "changed" } });
  saved.resolve({ id: "a", title: "changed" });
  await request;
  await assert.rejects(mutation.mutate({ kind: "remove", id: "a" }), /offline/);
  assert.deepEqual(client.getQueryData(homeKeys.links("A")), [{ id: "a", title: "changed" }]);
  assert.deepEqual(client.getQueryData(homeKeys.links("B")), [{ id: "b", title: "other workspace" }]);
});

function stickySessionFixture(t) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } },
  });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: "alice" });
  const key = stickyListOptions("A", "").queryKey;
  const replaceSession = () => {
    advanceIdentitySession(client);
    client.setQueryData(identityKeys.currentUser(), { id: "bob" });
    client.setQueryData(key, {
      pages: [{ results: [{ id: "replacement", name: "Bob's sticky" }] }],
      pageParams: ["first"],
    });
  };
  return { client, key, replaceSession };
}

await test("queued sticky writes cannot start in a replacement account session", async (t) => {
  const { client, key, replaceSession } = stickySessionFixture(t);
  const started = Promise.withResolvers();
  const finish = Promise.withResolvers();
  const transport = t.mock.method(StickyService.prototype, "updateSticky", () => {
    started.resolve();
    return finish.promise;
  });
  const writer = new MutationObserver(client, changeStickyOptions(client, "A", "n"));
  const first = writer.mutate({ kind: "update", id: "n", patch: { name: "old draft" } });
  await started.promise;
  const queued = assert.rejects(
    writer.mutate({ kind: "update", id: "n", patch: { name: "queued old draft" } }),
    /session changed/
  );
  replaceSession();
  finish.resolve({ id: "n", name: "old draft" });
  await Promise.all([first, queued]);
  assert.equal(transport.mock.callCount(), 1);
  assert.deepEqual(client.getQueryData(key).pages[0].results, [{ id: "replacement", name: "Bob's sticky" }]);
  assert.equal(client.getQueryState(key).isInvalidated, false);
});

await test("a late sticky rollback cannot restore another session's removed row", async (t) => {
  const { client, key, replaceSession } = stickySessionFixture(t);
  client.setQueryData(key, { pages: [{ results: [{ id: "old", name: "Alice's sticky" }] }], pageParams: ["first"] });
  const started = Promise.withResolvers();
  const finish = Promise.withResolvers();
  t.mock.method(StickyService.prototype, "deleteSticky", () => {
    started.resolve();
    return finish.promise;
  });
  const writer = new MutationObserver(client, changeStickyOptions(client, "A", "old"));
  const rejected = assert.rejects(writer.mutate({ kind: "remove", id: "old" }), /offline/);
  await started.promise;
  replaceSession();
  finish.reject(new Error("offline"));
  await rejected;
  assert.deepEqual(client.getQueryData(key).pages[0].results, [{ id: "replacement", name: "Bob's sticky" }]);
  assert.equal(client.getQueryState(key).isInvalidated, false);
});

await test("a failed sticky mutation leaves a replacement Query instance intact", async (t) => {
  const { client, key } = stickySessionFixture(t);
  client.setQueryData(key, { pages: [{ results: [{ id: "old", name: "old" }] }], pageParams: ["first"] });
  const started = Promise.withResolvers();
  const finish = Promise.withResolvers();
  t.mock.method(StickyService.prototype, "deleteSticky", () => {
    started.resolve();
    return finish.promise;
  });
  const rejected = assert.rejects(
    new MutationObserver(client, changeStickyOptions(client, "A", "old")).mutate({ kind: "remove", id: "old" }),
    /offline/
  );
  await started.promise;
  client.removeQueries({ queryKey: key, exact: true });
  client.setQueryData(key, { pages: [{ results: [{ id: "fresh", name: "refetched" }] }], pageParams: ["first"] });
  finish.reject(new Error("offline"));
  await rejected;
  assert.deepEqual(client.getQueryData(key).pages[0].results, [{ id: "fresh", name: "refetched" }]);
  assert.equal(client.getQueryState(key).isInvalidated, false);
});

await test("late sticky creation receipts cannot enter a replacement session", async (t) => {
  const { client, key, replaceSession } = stickySessionFixture(t);
  const started = Promise.withResolvers();
  const finish = Promise.withResolvers();
  t.mock.method(StickyService.prototype, "createSticky", () => {
    started.resolve();
    return finish.promise;
  });
  const creation = new MutationObserver(client, createStickyOptions(client, "A")).mutate({ name: "Alice's draft" });
  await started.promise;
  replaceSession();
  finish.resolve({ id: "old-created", name: "Alice's draft" });
  await creation;
  assert.deepEqual(client.getQueryData(key).pages[0].results, [{ id: "replacement", name: "Bob's sticky" }]);
  assert.equal(client.getQueryState(key).isInvalidated, false);
});

await test("the replacement session's sticky save does not wait for an old account's request", async (t) => {
  const { client, key, replaceSession } = stickySessionFixture(t);
  const started = Promise.withResolvers();
  const finish = Promise.withResolvers();
  const transport = t.mock.method(StickyService.prototype, "updateSticky", (_workspace, _id, patch) => {
    if (patch.name === "old") {
      started.resolve();
      return finish.promise;
    }
    return Promise.resolve({ id: "replacement", ...patch });
  });
  const old = new MutationObserver(client, changeStickyOptions(client, "A", "replacement")).mutate({
    kind: "update",
    id: "replacement",
    patch: { name: "old" },
  });
  await started.promise;
  replaceSession();
  await new MutationObserver(client, changeStickyOptions(client, "A", "replacement")).mutate({
    kind: "update",
    id: "replacement",
    patch: { name: "new session save" },
  });
  assert.equal(transport.mock.callCount(), 2);
  assert.equal(client.getQueryData(key).pages[0].results[0].name, "new session save");
  assert.equal(client.getQueryState(key).isInvalidated, true);
  finish.resolve({ id: "replacement", name: "old" });
  await old;
  assert.equal(client.getQueryData(key).pages[0].results[0].name, "new session save");
});

await test("same-session sticky creation keeps its canonical receipt in the captured list", async (t) => {
  const { client, key } = stickySessionFixture(t);
  client.setQueryData(key, { pages: [{ results: [] }], pageParams: ["first"] });
  const saved = { id: "canonical", name: "Saved by server", sort_order: 10 };
  t.mock.method(StickyService.prototype, "createSticky", async () => saved);
  await new MutationObserver(client, createStickyOptions(client, "A")).mutate({ name: "draft" });
  assert.deepEqual(client.getQueryData(key).pages[0].results, [saved]);
  assert.equal(client.getQueryState(key).isInvalidated, true);
});

await test("sticky deletion waits for its active save and keeps the row until acceptance", async (t) => {
  const { client, key } = stickySessionFixture(t);
  client.setQueryData(key, { pages: [{ results: [{ id: "n", description_html: "initial" }] }], pageParams: ["first"] });
  const saveStarted = Promise.withResolvers();
  const save = Promise.withResolvers();
  const deleteStarted = Promise.withResolvers();
  const deletion = Promise.withResolvers();
  t.mock.method(StickyService.prototype, "updateSticky", () => {
    saveStarted.resolve();
    return save.promise;
  });
  const destroy = t.mock.method(StickyService.prototype, "deleteSticky", () => {
    deleteStarted.resolve();
    return deletion.promise;
  });
  const options = changeStickyOptions(client, "A", "n");
  const saving = new MutationObserver(client, options).mutate({
    kind: "update",
    id: "n",
    patch: { description_html: "active" },
  });
  await saveStarted.promise;
  const removing = new MutationObserver(client, options).mutate({ kind: "remove", id: "n" });
  assert.equal(destroy.mock.callCount(), 0);
  save.resolve({ id: "n", description_html: "active" });
  await deleteStarted.promise;
  assert.deepEqual(client.getQueryData(key).pages[0].results, [{ id: "n", description_html: "active" }]);
  deletion.resolve();
  await Promise.all([saving, removing]);
  assert.deepEqual(client.getQueryData(key).pages[0].results, []);
});

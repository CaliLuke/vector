import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { webRequire } = await import("./helpers/react.mjs");
const { MutationObserver, QueryClient } = webRequire("@tanstack/react-query");
const { EUserWorkspaceRoles } = webRequire("@plane/types");
const {
  acceptWorkspaceDraftReceipt,
  captureWorkspaceDraftReceiptOwner,
  removePublishedWorkspaceDraft,
  workspaceDraftKeys,
  workspaceDraftDetailOptions,
  workspaceDraftListOptions,
  workspaceDraftMutationOptions,
} = await import("../../apps/web/core/queries/workspace-drafts.ts");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { WorkspaceDraftService } = await import("../../apps/web/core/services/issue/workspace_draft.service.ts");
const { advanceIdentitySession } = await import("../../apps/web/core/queries/identity.ts");

const scope = {
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "workspace",
  membershipRole: EUserWorkspaceRoles.ADMIN,
  membershipRevision: "membership-v1",
};
const draft = (id, overrides = {}) => ({
  id,
  name: id,
  created_by: "alice",
  updated_by: "alice",
  created_at: "2026-10-25T12:00:00Z",
  updated_at: "2026-10-25T12:00:00Z",
  is_draft: true,
  ...overrides,
});
const page = (results, totalCount = results.length) => ({
  results,
  next_cursor: "50:1:0",
  next_page_results: true,
  prev_cursor: undefined,
  prev_page_results: false,
  total_count: totalCount,
});

function fixture(t) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  t.after(() => client.clear());
  client.setQueryData(identityKeys.currentUser(), { id: scope.accountId });
  client.setQueryData(permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug), {
    is_active: true,
    role: scope.membershipRole,
    updated_at: scope.membershipRevision,
  });
  return client;
}

function visibleDraftIds(client, key) {
  return client.getQueryData(key).pages[0].results.map((row) => row.id);
}

await test("workspace draft list reads deduplicate, paginate and forward cancellation", async (t) => {
  const client = fixture(t);
  const pending = Promise.withResolvers();
  let requests = 0;
  let signal;
  t.mock.method(WorkspaceDraftService.prototype, "getIssues", (_workspace, params, requestSignal) => {
    requests++;
    signal = requestSignal;
    assert.equal(params.cursor, "50:0:0");
    return pending.promise;
  });
  const options = workspaceDraftListOptions(client, scope);
  const first = client.fetchInfiniteQuery(options);
  const second = client.fetchInfiniteQuery(options);
  assert.equal(requests, 1);
  const rejected = Promise.all([assert.rejects(first), assert.rejects(second)]);
  await client.cancelQueries({ queryKey: options.queryKey, exact: true });
  assert.equal(signal.aborted, true);
  pending.resolve(page([draft("one")]));
  await rejected;
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("workspace draft reads reject records owned by another account", async (t) => {
  const client = fixture(t);
  t.mock.method(WorkspaceDraftService.prototype, "getIssues", async () =>
    page([draft("private", { created_by: "bob" })])
  );
  const options = workspaceDraftListOptions(client, scope);
  await assert.rejects(client.fetchInfiniteQuery(options), /outside the current account/);
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("a replaced workspace membership query prevents a late draft detail response entering the cache", async (t) => {
  const client = fixture(t);
  const membershipKey = permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug);
  const pending = Promise.withResolvers();
  t.mock.method(WorkspaceDraftService.prototype, "getIssueById", () => pending.promise);
  const options = workspaceDraftDetailOptions(client, scope, "one");
  const read = client.fetchQuery(options);
  const rejected = assert.rejects(read, /access changed/);
  client.removeQueries({ queryKey: membershipKey, exact: true });
  client.setQueryData(membershipKey, {
    is_active: true,
    role: scope.membershipRole,
    updated_at: scope.membershipRevision,
  });
  pending.resolve(draft("one"));
  await rejected;
  assert.equal(client.getQueryData(options.queryKey), undefined);
});

await test("an accepted draft update from a prior session cannot rewrite current list data", async (t) => {
  const client = fixture(t);
  const key = workspaceDraftKeys.list(scope);
  client.setQueryData(key, { pages: [page([draft("one")])], pageParams: ["50:0:0"] });
  const started = Promise.withResolvers();
  const pending = Promise.withResolvers();
  t.mock.method(WorkspaceDraftService.prototype, "updateIssue", () => {
    started.resolve();
    return pending.promise;
  });
  const mutation = new MutationObserver(client, workspaceDraftMutationOptions(client, scope));
  const save = mutation.mutate({ kind: "update", issueId: "one", data: { name: "Saved" } });
  await started.promise;
  advanceIdentitySession(client);
  const nextScope = { ...scope, sessionGeneration: 1 };
  const currentKey = workspaceDraftKeys.list(nextScope);
  client.setQueryData(currentKey, {
    pages: [page([draft("one", { name: "Current session" })])],
    pageParams: ["50:0:0"],
  });
  pending.resolve(draft("one", { name: "Saved" }));
  await save;
  assert.equal(client.getQueryData(key).pages[0].results[0].name, "one");
  assert.equal(client.getQueryData(currentKey).pages[0].results[0].name, "Current session");
});

await test("reviewed draft update cannot reconcile into a replacement exact list query", async (t) => {
  const client = fixture(t);
  const key = workspaceDraftKeys.list(scope);
  client.setQueryData(key, { pages: [page([draft("one")])], pageParams: ["50:0:0"] });
  const owner = captureWorkspaceDraftReceiptOwner(client, scope, "one");
  assert.ok(owner);

  const pending = Promise.withResolvers();
  const save = pending.promise.then(() => acceptWorkspaceDraftReceipt(client, owner, draft("one", { name: "Saved" })));
  client.removeQueries({ queryKey: key, exact: true });
  client.setQueryData(key, {
    pages: [page([draft("one", { name: "Current query" })])],
    pageParams: ["50:0:0"],
  });
  pending.resolve();
  await save;

  assert.equal(client.getQueryData(key).pages[0].results[0].name, "Current query");
});

await test("reviewed draft publication cannot reconcile after the account session changes", async (t) => {
  const client = fixture(t);
  const key = workspaceDraftKeys.list(scope);
  client.setQueryData(key, { pages: [page([draft("one")])], pageParams: ["50:0:0"] });
  const owner = captureWorkspaceDraftReceiptOwner(client, scope, "one");
  assert.ok(owner);

  const pending = Promise.withResolvers();
  const publish = pending.promise.then(() => removePublishedWorkspaceDraft(client, owner, "one"));
  advanceIdentitySession(client);
  pending.resolve();
  await publish;

  assert.deepEqual(visibleDraftIds(client, key), ["one"]);
});

await test("reviewed publication cannot remove a draft from a replacement exact list query", async (t) => {
  const client = fixture(t);
  const key = workspaceDraftKeys.list(scope);
  client.setQueryData(key, { pages: [page([draft("one")])], pageParams: ["50:0:0"] });
  const owner = captureWorkspaceDraftReceiptOwner(client, scope, "one");
  assert.ok(owner);

  const pending = Promise.withResolvers();
  const publish = pending.promise.then(() => removePublishedWorkspaceDraft(client, owner, "one"));
  client.removeQueries({ queryKey: key, exact: true });
  client.setQueryData(key, {
    pages: [page([draft("one", { name: "Current query" })])],
    pageParams: ["50:0:0"],
  });
  pending.resolve();
  await publish;

  assert.deepEqual(visibleDraftIds(client, key), ["one"]);
});

await test("accepted create and delete receipts reconcile only the scoped workspace list", async (t) => {
  const client = fixture(t);
  const key = workspaceDraftKeys.list(scope);
  client.setQueryData(key, { pages: [page([draft("old")], 1)], pageParams: ["50:0:0"] });
  t.mock.method(WorkspaceDraftService.prototype, "createIssue", async (_workspace, payload) => draft("new", payload));
  t.mock.method(WorkspaceDraftService.prototype, "deleteIssue", async () => {});
  const mutation = new MutationObserver(client, workspaceDraftMutationOptions(client, scope));
  await mutation.mutate({ kind: "create", data: { name: "new" } });
  let saved = client.getQueryData(key).pages[0];
  assert.deepEqual(
    saved.results.map((row) => row.id),
    ["new", "old"]
  );
  assert.equal(saved.total_count, 1);
  assert.equal(client.getQueryState(key).isInvalidated, true);
  assert.equal(
    client.getQueryState(permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug)).isInvalidated,
    true
  );
  await mutation.mutate({ kind: "delete", issueId: "new" });
  saved = client.getQueryData(key).pages[0];
  assert.deepEqual(
    saved.results.map((row) => row.id),
    ["old"]
  );
  assert.equal(saved.total_count, 1);
  assert.equal(client.getQueryState(key).isInvalidated, true);
});

await test("accepted publication preserves captured owners while refreshing list and membership counts", async (t) => {
  const client = fixture(t);
  const listKey = workspaceDraftKeys.list(scope);
  const detailKey = workspaceDraftKeys.detail(scope, "one");
  // Detail first catches reconciliation that deletes its owner before reaching the list.
  client.setQueryData(detailKey, draft("one"));
  client.setQueryData(listKey, { pages: [page([draft("one"), draft("two")], 2)], pageParams: ["50:0:0"] });
  const owner = captureWorkspaceDraftReceiptOwner(client, scope, "one");
  assert.ok(owner);
  await removePublishedWorkspaceDraft(client, owner, "one");

  assert.equal(client.getQueryData(detailKey), null);
  assert.deepEqual(
    client.getQueryData(listKey).pages[0].results.map(({ id }) => id),
    ["two"]
  );
  assert.equal(client.getQueryState(listKey).isInvalidated, true);
  assert.equal(
    client.getQueryState(permissionKeys.workspaceMembership(scope.accountId, scope.workspaceSlug)).isInvalidated,
    true
  );
});

await test("a read started during a draft update cannot replace the accepted canonical receipt", async (t) => {
  const client = fixture(t);
  const detailKey = workspaceDraftKeys.detail(scope, "one");
  const listKey = workspaceDraftKeys.list(scope);
  client.setQueryData(detailKey, draft("one"));
  client.setQueryData(listKey, { pages: [page([draft("one")])], pageParams: ["50:0:0"] });
  const started = Promise.withResolvers();
  const writeResponse = Promise.withResolvers();
  const readResponse = Promise.withResolvers();
  let readSignal;
  t.mock.method(WorkspaceDraftService.prototype, "updateIssue", () => {
    started.resolve();
    return writeResponse.promise;
  });
  t.mock.method(WorkspaceDraftService.prototype, "getIssueById", (_workspace, _id, signal) => {
    readSignal = signal;
    return readResponse.promise;
  });
  const mutation = new MutationObserver(client, workspaceDraftMutationOptions(client, scope));
  const save = mutation.mutate({ kind: "update", issueId: "one", data: { name: "Saved" } });
  await started.promise;
  const read = client.fetchQuery({ ...workspaceDraftDetailOptions(client, scope, "one"), staleTime: 0 });
  writeResponse.resolve(draft("one", { name: "Saved" }));
  await save;
  assert.equal(readSignal.aborted, true);
  readResponse.resolve(draft("one", { name: "Old read" }));
  await read;
  assert.equal(client.getQueryData(detailKey).name, "Saved");
  assert.equal(client.getQueryData(listKey).pages[0].results[0].name, "Saved");
});

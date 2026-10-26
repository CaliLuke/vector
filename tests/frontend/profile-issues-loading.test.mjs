import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedIssueReadAccess } from "./helpers/issue-query-fixtures.mjs";
import { deferredIssueListRequest, observeIssueList } from "./helpers/issue-list-loading-fixtures.mjs";

registerWebModuleHooks();
const { issueListKeys, issueListOptions } = await import("../../apps/web/core/queries/issue-lists.ts");
const { UserService } = await import("../../apps/web/core/services/user.service.ts");

const scope = {
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  workspaceRole: 15,
  membershipRevision: "read-scope-1",
  projectRoles: [["project-id", 20]],
};
const options = { canGroup: true, perPageCount: 100 };
const issue = (id) => ({ id, project_id: "project-id", created_at: "2026-01-01T00:00:00Z" });
const response = (...issues) => ({ results: issues, total_count: issues.length });

function profileRead(view) {
  const input = {
    kind: "profile",
    workspaceSlug: "acme",
    userId: "user-id",
    accountId: "alice",
    view,
    richFilters: {},
    displayFilters: { layout: "list" },
    isCurrent: () => true,
    isReadScopeCurrent: () => true,
  };
  return { scope, pagination: options, request: { family: "profile", input, isReadCurrent: () => true } };
}

function fixture() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  seedIssueReadAccess(client, {
    workspaceSlug: scope.workspaceSlug,
    projectRoles: { "project-id": 20 },
    revision: scope.membershipRevision,
  });
  return client;
}

test("changing profile views aborts the superseded read and preserves the active result", async (t) => {
  const client = fixture();
  const created = deferredIssueListRequest();
  const subscribed = deferredIssueListRequest();
  let call = 0;
  t.mock.method(UserService.prototype, "getUserProfileIssues", (...args) =>
    [created, subscribed][call++].call(...args)
  );

  const createdQuery = profileRead("created");
  const observer = observeIssueList(t, client, createdQuery, QueryObserver, issueListOptions);
  const oldRequest = await created.started.promise;
  observer.setOptions(issueListOptions(profileRead("subscribed")));
  const activeRequest = await subscribed.started.promise;

  assert.equal(oldRequest.params.created_by, "user-id");
  assert.equal(activeRequest.params.subscriber, "user-id");
  assert.equal(oldRequest.signal.aborted, true);
  created.result.reject(new Error("aborted old read"));
  subscribed.result.resolve(response(issue("subscribed-ticket")));
  await new Promise((resolve) => setTimeout(resolve, 20));

  assert.equal(observer.getCurrentResult().isSuccess, true);
  assert.deepEqual(
    observer.getCurrentResult().data.response.results.map((row) => row.id),
    ["subscribed-ticket"]
  );
  assert.equal(client.getQueryData(issueListKeys.list(createdQuery)), undefined);
});

test("a late prior profile response cannot replace the selected view query", async (t) => {
  const client = fixture();
  const created = deferredIssueListRequest();
  const subscribed = deferredIssueListRequest();
  let call = 0;
  t.mock.method(UserService.prototype, "getUserProfileIssues", (...args) =>
    [created, subscribed][call++].call(...args)
  );

  const oldRead = profileRead("created");
  const observer = observeIssueList(t, client, oldRead, QueryObserver, issueListOptions);
  const oldRequest = await created.started.promise;
  observer.setOptions(issueListOptions(profileRead("subscribed")));
  const activeRequest = await subscribed.started.promise;
  subscribed.result.resolve(response(issue("current-ticket")));
  await new Promise((resolve) => setTimeout(resolve, 20));
  created.result.resolve(response(issue("stale-ticket")));
  await new Promise((resolve) => setTimeout(resolve, 20));

  assert.equal(oldRequest.signal.aborted, true);
  assert.deepEqual(
    observer.getCurrentResult().data.response.results.map((row) => row.id),
    ["current-ticket"]
  );
  assert.equal(client.getQueryData(issueListKeys.list(oldRead)), undefined);
  assert.ok(activeRequest.signal instanceof AbortSignal);
});

test("an active profile failure remains visible as a native Query error", async (t) => {
  const client = fixture();
  const request = deferredIssueListRequest();
  t.mock.method(UserService.prototype, "getUserProfileIssues", request.call);
  const observer = observeIssueList(t, client, profileRead("created"), QueryObserver, issueListOptions);
  await request.started.promise;
  request.result.reject(new Error("profile issues unavailable"));
  await new Promise((resolve) => setTimeout(resolve, 20));

  assert.equal(observer.getCurrentResult().isError, true);
  assert.match(observer.getCurrentResult().error.message, /profile issues unavailable/);
});

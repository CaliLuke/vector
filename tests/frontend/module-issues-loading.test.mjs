import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryObserver } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedIssueReadAccess } from "./helpers/issue-query-fixtures.mjs";
import { deferredIssueListRequest, observeIssueList } from "./helpers/issue-list-loading-fixtures.mjs";

registerWebModuleHooks();
const { issueListKeys, issueListOptions } = await import("../../apps/web/core/queries/issue-lists.ts");
const { IssueService } = await import("../../apps/web/core/services/issue/index.ts");

const scope = {
  accountId: "alice",
  sessionGeneration: 0,
  workspaceSlug: "acme",
  workspaceRole: 15,
  membershipRevision: "read-scope-1",
  projectId: "project-id",
  projectRole: 20,
  projectMembershipRole: 20,
};
const pagination = { canGroup: true, perPageCount: 100 };
const issue = (id) => ({ id, project_id: scope.projectId, created_at: "2026-01-01T00:00:00Z" });
const response = (...issues) => ({ results: issues, total_count: issues.length });

function moduleRead(moduleId) {
  const input = {
    kind: "module",
    workspaceSlug: scope.workspaceSlug,
    projectId: scope.projectId,
    viewId: moduleId,
    accountId: scope.accountId,
    richFilters: {},
    displayFilters: { layout: "list", group_by: null, order_by: "-created_at" },
    isScopeCurrent: () => true,
    isReadScopeCurrent: () => true,
  };
  return { scope, pagination, request: { family: "module", input, isReadCurrent: () => true } };
}

function fixture() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  seedIssueReadAccess(client, {
    workspaceSlug: scope.workspaceSlug,
    projectRoles: { [scope.projectId]: 20 },
    revision: scope.membershipRevision,
  });
  return client;
}

async function switchModules(t) {
  const client = fixture(),
    first = deferredIssueListRequest(),
    second = deferredIssueListRequest();
  let call = 0;
  t.mock.method(IssueService.prototype, "getIssues", (...args) => [first, second][call++].call(...args));
  const oldRead = moduleRead("module-a");
  const observer = observeIssueList(t, client, oldRead, QueryObserver, issueListOptions);
  const oldRequest = await first.started.promise;
  observer.setOptions(issueListOptions(moduleRead("module-b")));
  const currentRequest = await second.started.promise;
  return { client, first, second, oldRead, observer, oldRequest, currentRequest };
}

test("switching modules cancels the old Query and keeps the current read active", async (t) => {
  const { client, first, second, oldRead, observer, oldRequest, currentRequest } = await switchModules(t);

  assert.equal(oldRequest.params.module, "module-a");
  assert.equal(currentRequest.params.module, "module-b");
  assert.equal(oldRequest.signal.aborted, true);
  first.result.reject(new Error("aborted old read"));
  second.result.resolve(response(issue("current-ticket")));
  await new Promise((resolve) => setTimeout(resolve, 20));

  assert.equal(observer.getCurrentResult().isSuccess, true);
  assert.deepEqual(
    observer.getCurrentResult().data.response.results.map((row) => row.id),
    ["current-ticket"]
  );
  assert.equal(client.getQueryData(issueListKeys.list(oldRead)), undefined);
});

test("a late module response cannot overwrite its replacement module query", async (t) => {
  const { client, first, second, oldRead, observer, oldRequest, currentRequest } = await switchModules(t);
  second.result.resolve(response(issue("replacement-ticket")));
  await new Promise((resolve) => setTimeout(resolve, 20));
  first.result.resolve(response(issue("stale-ticket")));
  await new Promise((resolve) => setTimeout(resolve, 20));

  assert.equal(oldRequest.signal.aborted, true);
  assert.deepEqual(
    observer.getCurrentResult().data.response.results.map((row) => row.id),
    ["replacement-ticket"]
  );
  assert.equal(client.getQueryData(issueListKeys.list(oldRead)), undefined);
  assert.ok(currentRequest.signal instanceof AbortSignal);
});

test("an active module failure remains visible as a native Query error", async (t) => {
  const client = fixture();
  const request = deferredIssueListRequest();
  t.mock.method(IssueService.prototype, "getIssues", request.call);
  const observer = observeIssueList(t, client, moduleRead("module-a"), QueryObserver, issueListOptions);
  await request.started.promise;
  request.result.reject(new Error("module issues unavailable"));
  await new Promise((resolve) => setTimeout(resolve, 20));

  assert.equal(observer.getCurrentResult().isError, true);
  assert.match(observer.getCurrentResult().error.message, /module issues unavailable/);
});
